# vdub 待办事项

## 已完成

- [x] goddd 初始化项目 + Task/Step 领域 CRUD 生成
- [x] 流水线核心 6 步骤 (whisper/split/translate/tts/merge/burn)
- [x] 调度器 (2 worker goroutine + 翻译/TTS 互斥锁)
- [x] 适配器层 (whisper.cpp / ffmpeg whisper / OpenAI LLM / edge-tts / OpenAI TTS)
- [x] 配置结构体 (Pipeline/LLM/TTS) + 环境变量覆盖
- [x] SRT 解析工具 + 单元测试
- [x] DefaultConfig 补全 Pipeline/LLM/TTS 默认值
- [x] CLI `-run` 模式（含 ffmpeg 裁剪 `-ss`/`-to`）
- [x] Wire 接线：Usecase 持有 Scheduler，createTask 自动提交流水线
- [x] 翻译直接读取 src.srt 条目，保证时间轴 1:1 对齐（移除 split 中间步骤）
- [x] 单元测试: SRT 解析、分句规则、步骤构建、parseSRTTime
- [x] 集成测试: 裁剪视频 10~30s + 跑 ModeTranslate 流水线（build tag: integration）
- [x] dbNotifier 将流水线进度回写 Task DB

- [x] 数据目录管理 (`~/dsub/configs/`, `~/dsub/backup/`, `~/dsub/logs/`)，跨平台 `os.UserHomeDir()`
- [x] LLM 翻译鲁棒性：数量不符时自动重试（3 次 + 指数退避）
- [x] TTS 错误重试：指数退避 (3s/6s/12s)
- [x] 字幕样式修复：BorderStyle=1 + OutlineWidth=1，纯白字透明底
- [x] 字幕间距优化：中英文 MarginV 调整 (36/12)
- [x] 跳过已存在的 src.srt（输出目录 / 同名 .srt）
- [x] 字幕-音频同步单元测试 (sync_test.go)

- [x] 自定义翻译提示词：用户可自定义 LLM 翻译系统提示词，内置长度约束默认模板
- [x] 语义断行：字幕换行按语义（标点/连词）而非固定字数，CJK≤20字/行，Latin≤45字/行
- [x] 翻译+TTS 流水线并行：ModeDub 下翻译与 TTS goroutine 并行（producer-consumer channel）
- [x] 全步骤错误重试：每个步骤失败自动指数退避重试（3次，3s/6s/12s）
- [x] ffmpeg 进度回调：解析 stderr time= 字段实时计算编码百分比
- [x] 移除 TTS 语速控制（atempo 加速导致效果不佳）

### Flutter Desktop UI (Cupertino 风格)

- [x] Flutter 项目初始化 (ui/)，支持 macOS/Linux/Windows
- [x] Cupertino 侧边栏主页 + 任务列表页 + 设置页
- [x] 创建任务: 文件选择 + 模式选择 + 输出目录
- [x] 任务详情: 步骤进度展示 (轮询刷新)
- [x] API 客户端: 对接 Go 后端 HTTP API
- [x] Go 后端进程管理 (BackendService)
- [x] WebSocket 实时推送: task_progress / task_step_done / task_done / task_failed / task_log
- [x] WebSocket 心跳保活 (25s 间隔 + 自动重连)
- [x] 配置页完善: 编辑 Pipeline/LLM/TTS 配置 (GET/PUT /config API)
- [x] 任务删除确认
- [x] 任务详情实时日志 (WebSocket 驱动)
- [x] 后端连接状态指示器 (侧边栏 + 设置页)
- [x] Go 后端自动启动 + 存活监控

### 断点恢复 / 暂停 / 批量提交

- [x] 断点恢复: 失败/暂停任务从 CurrentStep 恢复（POST /tasks/:id/resume）
- [x] 暂停/恢复: Scheduler per-task context cancel + WasPaused 判定 + WebSocket task_paused 事件
- [x] 批量提交: POST /tasks/batch 扫描目录视频文件自动建 Task + Flutter 批量导入弹窗
- [x] 修复 status 零值覆盖 bug: SetTaskStatus 替代 copier 全量覆盖
- [x] Flutter status 映射修正: 0=等待/1=处理中/2=已暂停/3=已完成/4=失败
- [x] Flutter 任务卡片暂停/恢复/重试按钮
- [x] Flutter 任务详情暂停/恢复操作栏 + 错误信息展示

### 翻译/调速/批量优化

- [x] 翻译上下文窗口: 每个 chunk 携带前后各 3 句上下文提升连贯性
- [x] TTS 温和调速: 超时音频段 atempo 加速，上限可配置（MaxSpeedFactor）
- [x] 批量提交改为视频路径数组: POST /tasks/batch 接受 videos[] 替代目录
- [x] Flutter 批量导入弹窗改为多文件选择器
- [x] Flutter 配置页新增 TTS 调速上限滑块
- [x] 新增单元测试: semanticBreak / 翻译上下文窗口 / 调速因子逻辑
- [x] 生成 CHANGELOG.md

## 待实现

### 质量提升

- [ ] 术语锁定: 视频摘要提取术语表，翻译时强制使用
- [ ] 停顿节奏保留: 原视频停顿检测 → 配音对应位置插入同样停顿

### 工程优化

- [ ] 日志系统: 每个 Task 独立日志文件
- [ ] 资源清理: 处理完成后可选删除中间产物
