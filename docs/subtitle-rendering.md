# 字幕渲染与语义断行

vdub 的字幕渲染追求清晰可读与视觉一致性，
核心策略是 **描边透底 + 双层字幕 + 语义断行**。

---

## 字幕样式

### ASS 样式参数

| 参数 | 翻译字幕 | 原文字幕 | 说明 |
|------|---------|---------|------|
| FontSize | 18 | 14 | 翻译大、原文小 |
| PrimaryColour | &HFFFFFF | &HCCCCCC | 翻译纯白、原文浅灰 |
| OutlineColour | &H000000 | &H000000 | 黑色描边 |
| OutlineWidth | 1 | 1 | 1 像素描边 |
| BorderStyle | 1 | 1 | 描边+阴影（非色块） |
| Alignment | 2 | 2 | 底部居中 |
| MarginV | 36 | 12 | 翻译在上、原文在下 |

### 关键决策

**BorderStyle=1（描边模式）而非 BorderStyle=3（色块模式）**

色块模式会在字幕下方显示不透明矩形背景，遮挡画面且视觉突兀。
描边模式仅在文字边缘添加 1 像素黑色勾勒，在任何背景上都清晰可读，
同时保持画面通透。

**MarginV 间距控制**

翻译字幕 MarginV=36，原文字幕 MarginV=12：
- 两层字幕自然分开，不重叠
- 间距适中（约 10px），不会显得割裂
- 原文紧贴底部，翻译在其上方

### 跨平台字体

```go
switch runtime.GOOS {
case "linux":   → "NotoSansCJK-Regular"
case "darwin":  → "Arial Unicode MS"
default:        → "Arial"
}
```

选择各平台预装的 Unicode 字体，确保中日韩字符正常显示，无需用户额外安装字体。

---

## 语义断行

### 问题

字幕渲染时，过长的单行文本会导致：
- 字体被压缩到极小以适配宽度
- 观众阅读时间不足
- 在不同比例的屏幕上溢出

传统方案按固定字数（如每 20 字）截断，但这可能在词中间或语义单元中间断开，
降低阅读流畅度。

### 方案

`semanticBreak` 函数在 SRT 生成时自动对长行做语义断行：

```
原文: 过去六个月里我学到不少新东西，虽然不能说改动很大，但确实有些有趣的变化。
断行: 过去六个月里我学到不少新东西，
      虽然不能说改动很大，但确实有些有趣的变化。
```

### 断点选择算法

1. 确定文本是否需要断行：
   - CJK 文本：> 20 字符则断行
   - 拉丁文本：> 45 字符则断行

2. 在文本中点 ±(limit/2) 范围内搜索最佳断点

3. 断点优先级（数字越小越优先）：

| 优先级 | 匹配条件 | 示例 |
|--------|---------|------|
| 1 | 句号/问号/感叹号 | `。！？!?` |
| 2 | 逗号/分号/顿号/冒号 | `，,、；;：:` |
| 3 | 英文连词/介词前的空格 | `and`, `but`, `because` |
| 4 | 普通空格 | 单词边界 |

4. 同优先级取距中点最近者

5. 所有候选都找不到时，在中点处强制断开

### CJK 检测

通过 Unicode 范围判断是否包含中日韩字符：
- `unicode.Han`（汉字）
- `unicode.Katakana`（片假名）
- `unicode.Hiragana`（平假名）

只要包含任一 CJK 字符，即采用 CJK 行宽上限。

### 英文连词表

以下词被识别为英文连词/介词，优先在其前方断行：

```
and, or, but, that, which, when, where,
because, so, if, while, for, with, from,
into, about
```

---

## 双语字幕

ModeTranslate 和 ModeDub 输出双语字幕：

```
[翻译字幕 - 白色大字 - 离底部 36px]
这是这门课的第六版。

[原文字幕 - 浅灰小字 - 离底部 12px]
This is the sixth edition of this class.
```

通过 ffmpeg 的 `subtitles` 滤镜实现，两层滤镜依次叠加：

```
-vf "subtitles=trans.srt:force_style='...',subtitles=src.srt:force_style='...'"
```

### 路径转义

字幕文件路径中的特殊字符（冒号、方括号、单引号）会被 `escapeFFmpegPath` 转义，
避免 ffmpeg 解析失败：

```
: → \:
' → \'
[ → \[
] → \]
```

---

## ModeSubtitle

仅烧录原文字幕（单层），字体为白色 18pt，无翻译层。

---

## ModeDub 音频混合

烧录字幕的同时，将原视频音频降至 15% 音量，与配音音频混合：

```
[0:a] volume=0.15 → [bg]
[bg] + [1:a] → amix → [a]
```

- 原声作为背景环境音保留
- 配音为主音轨
- `dropout_transition=3` 避免音频结尾突然中断

输出编码为 AAC 128kbps，兼顾质量与文件大小。

---

## 性能考量

| 环节 | 优化 |
|------|------|
| 编码 | ffmpeg 默认使用硬件加速（如有） |
| 进度 | 解析 stderr 实时上报，不阻塞编码 |
| 静音填充 | `anullsrc` lavfi 源零 I/O |
| 拼接 | concat demuxer 无重编码 |
