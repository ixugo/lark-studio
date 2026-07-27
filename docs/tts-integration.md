# TTS 集成指南

vdub 通过 OpenAI 兼容 API 对接多种 TTS 引擎，
一套配置支持 Edge TTS、OpenAI TTS 及自部署的开源 TTS 服务。

---

## 架构

```
vdub 配置 → TTSClient 接口 → OpenAI 兼容适配器 → TTS 服务
                                      │
                   ┌──────────────────┤──────────────────┐
                   │                  │                  │
              Edge TTS          OpenAI TTS         自部署服务
              (内置)           (官方 API)     (CosyVoice/F5/ChatTTS)
```

`TTSClient` 接口：

```go
type TTSClient interface {
    Synthesize(ctx context.Context, text, voice string) ([]byte, error)
}
```

所有 TTS 引擎只需实现此接口。当前内置两个适配器：

| 适配器 | 协议 | 说明 |
|--------|------|------|
| Edge TTS | 微软 WebSocket | 免费、无需 API Key、质量中等 |
| OpenAI TTS | HTTP REST | 标准 OpenAI `/v1/audio/speech` |

---

## 内置引擎

### Edge TTS

微软免费 TTS 服务，语音自然度中等偏上。

```toml
[pipeline]
tts_provider = "edge"
```

无需额外配置。默认语音为 `zh-CN-XiaoxiaoNeural`（中文女声）。

适用场景：
- 快速体验，无需注册账号
- 对延迟不敏感的批量任务
- 不方便部署本地服务的环境

局限性：
- 语速不可精确控制
- 不支持语音克隆
- 依赖网络，微软可能随时调整服务策略

### OpenAI TTS

通过 `/v1/audio/speech` 端点，支持 OpenAI 官方和所有兼容此协议的服务。

```toml
[pipeline]
tts_provider = "openai"

[pipeline.tts]
base_url = "https://api.openai.com/v1"
api_key  = "sk-..."
model    = "tts-1"
voice    = "alloy"
```

---

## 自部署开源 TTS

以下开源 TTS 项目均可通过 OpenAI 兼容 API 封装后接入 vdub，
无需修改 vdub 代码，只需配置 `base_url`。

### CosyVoice

阿里达摩院开源，支持中英双语、语音克隆、情感控制。

**部署：**

```bash
# 1. 克隆并安装
git clone https://github.com/FunAudioLLM/CosyVoice.git
cd CosyVoice
pip install -r requirements.txt

# 2. 启动 OpenAI 兼容服务（推荐使用社区封装）
# https://github.com/v3ucn/CosyVoice_For_Windows
# 或使用 cosyvoice-api：
pip install cosyvoice-api
cosyvoice-api --port 8880
```

**vdub 配置：**

```toml
[pipeline]
tts_provider = "openai"

[pipeline.tts]
base_url = "http://localhost:8880/v1"
api_key  = "not-needed"
model    = "cosyvoice"
voice    = "中文女"
```

### F5-TTS

轻量级高质量 TTS，支持多语言、零样本语音克隆。

**部署：**

```bash
pip install f5-tts
# 启动 OpenAI 兼容服务
f5-tts_infer-gradio --port 7860
# 或使用社区 API 封装：
# https://github.com/FunAudioLLM/F5-TTS
```

**vdub 配置：**

```toml
[pipeline]
tts_provider = "openai"

[pipeline.tts]
base_url = "http://localhost:7860/v1"
api_key  = "not-needed"
model    = "f5-tts"
voice    = "default"
```

### ChatTTS

专为对话场景优化的 TTS，音色自然、支持笑声和停顿等韵律控制。

**部署：**

```bash
pip install chattts
# 使用 ChatTTS-OpenAI-API 封装：
# https://github.com/6drf21e/ChatTTS_colab
git clone https://github.com/6drf21e/ChatTTS_colab.git
cd ChatTTS_colab
pip install -r requirements.txt
python api.py --port 8890
```

**vdub 配置：**

```toml
[pipeline]
tts_provider = "openai"

[pipeline.tts]
base_url = "http://localhost:8890/v1"
api_key  = "not-needed"
model    = "chattts"
voice    = "default"
```

---

## 引擎对比

| 特性 | Edge TTS | OpenAI TTS | CosyVoice | F5-TTS | ChatTTS |
|------|----------|------------|-----------|--------|---------|
| 费用 | 免费 | 按量付费 | 免费(本地) | 免费(本地) | 免费(本地) |
| 语音克隆 | 不支持 | 不支持 | 支持 | 支持 | 不支持 |
| 中文质量 | 中 | 中高 | 高 | 高 | 高 |
| 英文质量 | 中 | 高 | 中 | 高 | 中 |
| GPU 需求 | 无 | 无 | 4GB+ | 4GB+ | 4GB+ |
| 延迟 | 中 | 低 | 低(本地) | 低(本地) | 低(本地) |
| 情感控制 | 有限 | 不支持 | 支持 | 有限 | 支持 |

### 选择建议

- **快速体验 / 无 GPU**：Edge TTS
- **商用 / 最简部署**：OpenAI TTS
- **中文为主 / 需要克隆**：CosyVoice
- **多语言 / 零样本克隆**：F5-TTS
- **对话风格 / 韵律控制**：ChatTTS

---

## 通用原则

所有开源 TTS 项目接入 vdub 的核心步骤相同：

1. 部署 TTS 服务
2. 用 OpenAI 兼容 API 封装（社区通常已有现成方案）
3. 配置 `base_url` 指向本地服务地址
4. `api_key` 可填任意值（本地服务通常不校验）
5. `model` 和 `voice` 根据具体服务文档填写

vdub 不内置任何开源 TTS 的直接依赖，
通过 OpenAI 兼容协议实现松耦合，未来出现新 TTS 引擎可零修改接入。
