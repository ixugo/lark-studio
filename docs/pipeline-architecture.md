# 流水线架构

vdub 的视频处理流水线由 6 个可组合步骤构成，按处理模式动态编排。

---

## 步骤总览

| 步骤 | 输入 | 输出 | 用途 |
|------|------|------|------|
| whisper | 视频/音频 | `src.srt` | 语音识别，生成原文字幕 |
| split | `src.srt` | `src.srt`（覆写） | LLM 语义分句 + 时间轴重对齐 |
| translate | `src.srt` | `trans.srt` + `trans.txt` | 翻译字幕，保持时间轴对齐 |
| tts | `trans.txt` | `audio_segs/*.wav` | 文本转语音 |
| merge | `audio_segs/*.wav` + `src.srt` | `dub.mp3` | 按时间轴拼接音频段 |
| burn | 视频 + 字幕 + 音频 | 最终视频 | 烧录字幕 / 混合配音 |

---

## 模式编排

```
ModeSubtitle  (1):  whisper → burn
ModeTranslate (2):  whisper → split → translate → burn
ModeDub       (3):  whisper → split → translate → tts → merge → burn
```

每个模式裁剪出不同的步骤子集。`buildSteps()` 根据 mode 返回有序步骤名列表，
`Run()` 按此列表顺序执行。

---

## 翻译+TTS 流水线并行

ModeDub 模式下，translate 步骤内部同时启动 TTS goroutine，
实现 **producer-consumer** 并行：

```
┌─────────────────────┐     channel     ┌─────────────────────┐
│  翻译 goroutine      │ ──────────────→ │  TTS goroutine       │
│  (持有 llmMu)        │   ttsPair{     │  (持有 ttsMu)         │
│  每 8 句一批翻译      │    index, text │  逐句合成语音          │
│  翻译完推入 channel   │   }            │  写入 audio_segs/N.wav│
└─────────────────────┘                 └─────────────────────┘
```

翻译完成后关闭 channel，TTS goroutine 排空后汇合。
两个资源互斥锁（`llmMu` / `ttsMu`）各自独立，不产生死锁。

TTS 步骤在 ModeDub 中检测到 `audio_segs/` 已有 wav 文件时自动跳过。

### 断点恢复支持

TTS goroutine 在合成前检查音频文件是否已存在：
- 存在且大小 > 0 → 跳过
- 不存在 → 合成

任务中途中断后重新提交，已完成的音频段不会重复合成。

---

## 步骤级指数退避重试

每个步骤执行时被 `runStepWithRetry` 包装：

```
最多 3 次尝试
延迟: 3s → 6s → 12s (2^attempt × baseDelay)
```

Context 取消（用户主动暂停）不触发重试。

此机制覆盖所有步骤，与 LLM/TTS 适配器内部的重试叠加：
- LLM 翻译数量不符 → 适配器内重试 3 次
- TTS 合成网络错误 → 适配器内重试 3 次
- 步骤整体失败 → 步骤级重试 3 次

三层重试确保网络波动下的鲁棒性，同时 context 取消可随时中止。

---

## ffmpeg 进度回调

`runFFmpegWithProgress` 封装了 ffmpeg 进度解析：

1. 用 `ffprobe` 获取输入媒体总时长
2. 启动 ffmpeg，获取 stderr pipe
3. 自定义 `SplitFunc` 按 `\r` / `\n` 分割 stderr 行（ffmpeg 用 `\r` 覆盖进度）
4. 正则匹配 `time=HH:MM:SS.cs` 字段
5. 计算 `当前时间 / 总时长 × 100` 得出百分比
6. 通过 `Notifier.OnProgress` 推送到 UI

烧录步骤（burn）和音频提取步骤均使用此机制，
UI 端通过 WebSocket 接收实时编码进度。

---

## 文件约定

每个任务的输出目录结构：

```
<output_dir>/
├── clipped.mp4          # 裁剪后的输入视频（可选）
├── raw.mp3              # 提取的音频
├── src.srt              # 原文字幕（split 覆写后为语义分句版本）
├── trans.srt            # 翻译字幕（带语义断行）
├── trans.txt            # 纯翻译文本（TTS 读取用）
├── audio_segs/          # TTS 音频段
│   ├── 0.wav
│   ├── 1.wav
│   └── ...
├── silence_*.wav         # 填充用静音文件
├── concat_list.txt       # ffmpeg concat 拼接列表
├── dub.mp3              # 拼接后的配音音频
└── <basename>.final.mp4 # 最终输出视频
```

---

## Notifier 接口

流水线通过 `Notifier` 接口与外部解耦：

```go
type Notifier interface {
    OnProgress(taskID, step string, progress int)
    OnStepDone(taskID, step string)
    OnTaskDone(taskID string)
    OnTaskFailed(taskID string, err error)
    OnLog(taskID, msg string)
}
```

生产环境中由 `dbNotifier` 实现：
- 将进度写入 SQLite（Task 表）
- 通过 WebSocket Hub 广播到 Flutter UI

CLI 模式下使用 `noopNotifier`，所有事件静默丢弃。

---

## 资源互斥模型

```
         ┌── Worker 0 ──┐
Job Queue─┤              ├─ 竞争 llmMu / ttsMu
         └── Worker 1 ──┘
```

- 2 个 worker goroutine 并行处理不同视频
- 同一时刻只有一个 worker 占用 LLM（`llmMu`）
- 同一时刻只有一个 worker 占用 TTS（`ttsMu`）
- whisper 和 ffmpeg 不受互斥限制，可真正并行

这避免了多任务同时请求 LLM/TTS 导致的过载或计费翻倍，
同时允许 I/O 密集型步骤（whisper、ffmpeg）充分利用硬件。
