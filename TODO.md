# vdub 待办事项

## 已完成

- [x] goddd 初始化项目 + Task/Step 领域 CRUD 生成
- [x] 流水线核心 6 步骤 (whisper/split/translate/tts/merge/burn)
- [x] 调度器 (2 worker goroutine + 翻译/TTS 互斥锁)
- [x] 适配器层 (whisper.cpp / OpenAI LLM / edge-tts / OpenAI TTS)
- [x] 配置结构体 (Pipeline/LLM/TTS)
- [x] SRT 解析工具

## 进行中

- [ ] Wire 接线 + main.go 串联所有组件
- [ ] 数据目录管理 (`~/dsub/configs/`, `~/dsub/backup/`)，跨平台 `os.UserHomeDir()`

## 待实现

### 核心功能

- [ ] 断点恢复: Task 状态持久化到 SQLite，恢复时从 CurrentStep 继续
- [ ] 暂停/恢复: context cancel + 重新 Submit
- [ ] 智能字幕检测: 输入文件同名 .srt 存在时跳过 whisper
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
- [ ] 单元测试: SRT 解析、分句规则、时间轴对齐
- [ ] 集成测试: 端到端处理一个短视频
