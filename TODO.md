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

## 待实现

### 核心功能

- [ ] 断点恢复: Task 状态持久化到 SQLite，恢复时从 CurrentStep 继续
- [ ] 暂停/恢复: context cancel + 重新 Submit
- [ ] 批量提交: 用户指定目录，自动扫描视频文件建立多个 Task

### Web UI (HTMX + WebSocket)

- [ ] 主页: 任务列表 (状态/进度/文件名)
- [ ] 创建任务: 填写本地路径 + 选择模式 + 选择输出目录
- [ ] 任务详情: 实时进度/日志 (WebSocket 推送)
- [ ] 配置页: 编辑 Pipeline/LLM/TTS 配置 (表单保存到 config.toml)
- [ ] WebSocket 集成 (pkg/ws): task_update / progress / log 三类消息推送
- [ ] HTMX 局部刷新: ws 收到推送后 trigger 对应组件更新

### 质量提升（超越 VideoLingo）

- [ ] 翻译三轮校对: 直译 → 意译 → 审校（术语一致性+上下文连贯性）
- [ ] 术语锁定: 视频摘要提取术语表，翻译时强制使用
- [ ] 长度约束: 翻译时给 LLM 提供原句时长，要求译文可在时长内自然语速读完
- [ ] TTS 语速控制: 根据原句时长动态调整语速参数
- [ ] 停顿节奏保留: 原视频停顿检测 → 配音对应位置插入同样停顿
- [ ] 语义断行: 字幕换行按语义而非固定字数
- [ ] 单视频内步骤流水线化: 边翻译边对已翻译句子做 TTS

### 工程优化

- [ ] 错误重试: 翻译/TTS 失败自动重试 (指数退避)
- [ ] 日志系统: 每个 Task 独立日志文件
- [ ] 资源清理: 处理完成后可选删除中间产物
- [ ] ffmpeg 进度回调: 解析 ffmpeg stderr 获取编码进度
