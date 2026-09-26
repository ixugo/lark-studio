# 流水线错误记录

### 错误：直接翻译服务仍进入大模型预处理

**场景：** 任务使用必应或 DeepLX 翻译。

**正确做法：** 仅 OpenAI 任务执行语义分句、上下文拼接和术语提示词。必应、DeepLX 直接将 Whisper 字幕分块交给各自 HTTP 接口；进度权重也不包含分句步骤。

---

### 错误：Whisper 集成测试桩落后于运行接口

**场景：** WhisperRunner 的 `Transcribe` 增加进度和原始日志回调。

**正确做法：** 所有测试桩实现六参数接口，并调用回调。真实集成测试使用绝对路径、WAV 音频和环境变量提供模型与样本。

---

### 记录：whisper.cpp v1.8.6 运行时探测

**场景：** 内嵌的 `whisper-cli` 需要判断是否真的可执行。

**正确做法：** 不使用 `--version` 或 `--help`。v1.8.6 对前者报参数错误，后者不退出。以 `-m __vdub_runtime_probe_missing__.bin -f /dev/null`（Windows 使用 `NUL`）探测，并确认输出含 `failed to open`。
