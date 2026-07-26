// Package pipeline 负责视频处理流水线编排。
//
// 流水线包含以下步骤（按 Mode 组合）：
//   - whisper: 语音识别生成 SRT 字幕
//   - split: 句子分割（规则+LLM语义分割）
//   - translate: 翻译字幕
//   - tts: 文本转语音生成配音音频
//   - merge: 合并配音音频到时间轴
//   - burn: 烧录字幕到视频 / 导出字幕文件
//
// 并发模型：
//   - 2 worker 并行处理不同视频
//   - 翻译和 TTS 资源互斥（同一时刻只有一个 worker 使用）
//   - whisper 和 ffmpeg 可并行
package pipeline
