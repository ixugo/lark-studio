# Changelog

## Unreleased

### feat
- 翻译上下文窗口：每个 chunk 携带前后各 3 句上下文，提升跨块翻译连贯性
- TTS 温和调速：超时音频段自动 atempo 加速，上限可配置（MaxSpeedFactor）
- 批量提交改为视频路径数组：POST /tasks/batch 接受 videos[] 而非目录
- Flutter 批量导入弹窗改为多文件选择器
- Flutter 配置页新增 TTS 调速上限滑块

### fix
- 修复 status 零值覆盖 bug（SetTaskStatus 替代 copier 全量覆盖）
- Flutter status 映射修正（0=等待/1=处理中/2=已暂停/3=已完成/4=失败）

---

## v0.3.0

### feat
- 断点恢复：失败/暂停任务从 CurrentStep 恢复（POST /tasks/:id/resume）
- 暂停/恢复：Scheduler per-task context cancel + WasPaused + WebSocket task_paused
- 批量提交：POST /tasks/batch 扫描目录视频自动建 Task + Flutter 批量导入弹窗
- Flutter 任务卡片暂停/恢复/重试按钮
- Flutter 任务详情暂停/恢复操作栏 + 错误信息展示

## v0.2.0

### feat
- 自定义翻译提示词：用户可自定义 LLM 翻译系统提示词，内置长度约束默认模板
- 语义断行：字幕换行按语义（标点/连词）而非固定字数
- 翻译+TTS 流水线并行：ModeDub 下 producer-consumer channel 并行
- 全步骤错误重试：每个步骤失败自动指数退避重试（3次，3s/6s/12s）
- ffmpeg 进度回调：解析 stderr time= 字段实时计算编码百分比

### perf
- 移除 TTS 语速控制（atempo 加速效果不佳，改为可配置温和调速）

## v0.1.0

### feat
- Flutter Desktop UI（Cupertino 风格），支持 macOS/Linux/Windows
- Go 后端进程管理（BackendService 自动启停）
- WebSocket 实时推送：task_progress / task_step_done / task_done / task_failed / task_log
- WebSocket 心跳保活（25s 间隔 + 自动重连）
- 配置页：Pipeline/LLM/TTS 全量编辑（GET/PUT /config）
- 任务创建/列表/详情/删除完整功能
- 后端连接状态指示器

## v0.0.1

### feat
- goddd 六边形架构初始化 + Task/Step 领域 CRUD
- 流水线核心 6 步骤（whisper/split/translate/tts/merge/burn）
- 调度器（2 worker goroutine + 翻译/TTS 互斥锁）
- 适配器层（whisper.cpp / ffmpeg / OpenAI LLM / edge-tts / OpenAI TTS）
- 配置结构体（Pipeline/LLM/TTS）+ 环境变量覆盖
- CLI `-run` 模式（含 ffmpeg 裁剪）
- SRT 解析工具 + 单元测试
- 数据目录管理（~/dsub/）
- LLM/TTS 指数退避重试
- 字幕样式修复（BorderStyle=1 纯白字透明底）
- 跳过已存在的 src.srt
