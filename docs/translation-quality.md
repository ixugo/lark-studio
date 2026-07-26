# 翻译质量设计

vdub 的翻译链路经过多层优化，确保字幕翻译在准确性、自然度和时间适配上达到高水准。

---

## 三阶段流程

```
1. 语音识别（whisper）   → 原始碎片字幕
2. 语义分句（LLM split） → 完整自然句
3. 翻译（LLM translate） → 长度约束下的目标语言字幕
```

每一阶段解决特定问题，避免单一步骤承担过多职责。

---

## 阶段一：语音识别

使用 whisper（ffmpeg 内置滤镜或 whisper.cpp）将音频转为 SRT。

whisper 的原始输出通常是 **碎片化** 的：
- 一句话可能被切成 3-5 个短条目
- 句子在非语义边界被截断
- 时间轴精度到词级但条目划分不自然

这些碎片直接用于翻译会导致：
- 每个碎片独立翻译，丢失上下文
- 短碎片翻译后长度失控
- 字幕闪烁频繁

因此需要阶段二的语义分句。

---

## 阶段二：语义分句（split）

### 核心思路

将 whisper 碎片的全部文本拼接为连续文本，交给 LLM 重新按自然句切分，
然后通过字符-时间映射将新句子对齐回原始时间轴。

### 字符-时间映射

`buildCharTimes` 函数为每个字符插值出精确时间点：

```
原始条目 "Hello world" (0.0s - 2.0s)

字符:  H   e   l   l   o       w   o   r   l   d
时间:  0.0 0.2 0.4 0.6 0.8 1.0 1.2 1.4 1.6 1.8 2.0
```

每个字符的时间 = `startSec + (endSec - startSec) × charIndex / (charCount - 1)`

多个条目的文本用空格连接，空格的时间取前一条目的 endSec。

### 句子对齐

`alignSentences` 将 LLM 返回的自然句在字符序列中定位：

1. 对 LLM 返回的每个句子，在全文 rune 序列中做子串匹配（大小写不敏感）
2. 匹配位置的首字符时间 → startSec
3. 末字符时间 → endSec
4. 生成新的 SRT 条目

### 间隙消除

相邻条目间小于 1 秒的间隙自动填平（前一条目 endSec 延伸到下一条目 startSec），
避免字幕出现短暂闪烁。

### 失败降级

若 LLM 分句失败（网络错误或返回格式异常），保留原始 whisper 字幕继续流水线。
不会因分句环节出错导致整个任务失败。

---

## 阶段三：翻译

### 分块翻译

每次将 8 句（`translateChunkSize`）发给 LLM，平衡上下文窗口与延迟：
- 太小（逐句）→ 丢失语境，翻译质量下降
- 太大（全部一次）→ Token 限制、数量对齐困难

8 句是经验值，在 7B-70B 模型上均表现稳定。

### 编号锁定

每句前缀编号（`1. xxx`），要求 LLM 返回同样编号的翻译：

```
输入:
1. This is the sixth edition of this class.
2. There have been some interesting changes.

输出:
1. 这是这门课的第六版。
2. 有一些有趣的变化。
```

编号锁定确保输出与输入严格 1:1 对应，
`parseNumberedLines` 解析时自动去除编号前缀。

### 数量校验与重试

翻译完成后校验返回行数是否与输入一致：
- 一致 → 采用
- 不一致 → 重试（最多 3 次，指数退避）
- 重试耗尽 → 用原文填充缺失行 / 截断多余行

这种防御性对齐保证后续 TTS 和时间轴匹配不会因翻译行数错误而崩溃。

### 长度约束

内置默认提示词包含长度约束指令：

> Keep translations concise: each translated line should be short enough
> to read naturally at normal speaking speed within the original subtitle's
> display duration

这引导 LLM 产出适合字幕显示的简短译文，而非逐字直译的冗长文本。

### 自定义提示词

用户可通过 `Pipeline.TranslatePrompt` 完全替换系统提示词。
支持两个模板变量：

| 变量 | 替换为 |
|------|--------|
| `{{target_lang}}` | 目标语言（如 `zh-CN`） |
| `{{count}}` | 当前批次句数 |

留空则使用内置默认提示词。

典型自定义场景：
- 专业领域（医学、法律）需要术语规范
- 目标受众为儿童，需要简化用词
- 翻译风格偏好（口语化 / 书面化）

---

## 质量保障机制

| 机制 | 解决的问题 |
|------|-----------|
| LLM 语义分句 | whisper 碎片化输出 → 自然句 |
| 字符级时间插值 | 重分句后时间轴精确还原 |
| 编号锁定翻译 | 输入输出 1:1 严格对齐 |
| 数量校验重试 | LLM 输出行数不稳定 |
| 长度约束提示词 | 译文过长无法在字幕窗口内阅读 |
| 语义断行 | 长字幕按标点/连词换行 |
| 间隙消除 | 字幕闪烁问题 |
| 三层重试 | 网络波动不丢数据 |

---

## 适配器抽象

翻译逻辑不绑定特定 LLM 服务商。`LLMClient` 接口：

```go
type LLMClient interface {
    SplitSentences(ctx, text, lang) ([]string, error)
    Translate(ctx, sentences, targetLang, systemPrompt) ([]string, error)
}
```

当前实现为 OpenAI 兼容协议（支持 Ollama、DeepSeek、通义千问等），
替换为其他 LLM 只需实现此接口。
