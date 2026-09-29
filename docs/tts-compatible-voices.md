# OpenAI 兼容模型与固定音色

翻译、语音识别和语音合成的模型由配置地址的 `/models` 查询，只能选择返回的模型。服务失联、模型不在目录、音色无效时，保存和任务提交均返回具体错误；失败的任务预检不会暂存输入或删除已有产物。

语音合成继续使用 OpenAI 兼容引擎。根据服务 OpenAPI 识别 MLX 的 `lang_code`、`instruct` 字段，标准服务使用 `instructions`。能力按所选模型展示，不因接口接受字段就假定模型支持它。Qwen3-TTS 0.6B CustomVoice 不启用情绪指令，1.7B CustomVoice 才启用。模型能力依据：[Qwen 官方说明](https://github.com/QwenLM/Qwen3-TTS)。

先查询 `/audio/voices?model=...`；服务返回音色时直接选择。已识别 MLX 的 Qwen3-TTS CustomVoice 服务若未返回音色，则明确显示模型内置的九个固定音色。未知服务不伪造音色列表，使用该服务实际要求的音色标识。

每次任务开始配音时冻结客户端、模型、音色、语言和情绪参数，所有句子共用这一份配置。任务中修改全局配置只影响后续任务。Edge 与 OpenAI 音色独立保存，避免把 Edge 默认音色传给 Qwen。更换模型、音色、语速或其他声音参数后，恢复任务会清除旧的中间配音缓存再生成，避免同一成片混用新旧声音。

验证入口：`internal/core/pipeline/qwen_consistency_live_test.go`。只有显式启用 `LARK_QWEN_LIVE=1` 才访问本机 8399 服务；普通自动测试不连接用户服务。实际验收用 12 句受控中文、固定 Vivian，经过配音、合并、视频输出，再以另一女声 Serena 与男声 Ryan 作独立说话人特征对照。此方法验证本次样本，不能取代用户原视频的听感验收。
