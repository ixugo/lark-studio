package tts

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/ixugo/vdub/internal/adapter/remoteapi"
)

const (
	capabilitiesMaxURLBytes   = 4096
	capabilitiesMaxModelBytes = 512
	capabilitiesMaxVoices     = 1000
)

// Voice 是服务音色或已确认的 Qwen 内置说话人。
type Voice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Capabilities 只启用服务协议及所选模型已明确支持的功能。
type Capabilities struct {
	Protocol     string   `json:"protocol"`
	Voices       []Voice  `json:"voices"`
	VoiceSource  string   `json:"voice_source"`
	Instructions bool     `json:"instructions"`
	Languages    []string `json:"languages"`
}

// DiscoverCapabilities 查询服务音色及 OpenAPI，避免把兼容地址视为支持所有扩展。
func DiscoverCapabilities(ctx context.Context, baseURL, apiKey, model string) (Capabilities, error) {
	base, err := capabilitiesBaseURL(baseURL, model)
	if err != nil {
		return Capabilities{}, err
	}
	result := Capabilities{Protocol: "openai", Voices: []Voice{}, VoiceSource: "manual", Languages: []string{}}
	schema, err := fetchSpeechSchema(ctx, base, apiKey)
	if err != nil {
		return result, err
	}
	configureCapabilities(&result, model, schema)
	endpoint := base.Clone()
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/audio/voices"
	query := endpoint.Query()
	query.Set("model", model)
	endpoint.RawQuery = query.Encode()
	var response struct {
		Data []Voice `json:"data"`
	}
	err = remoteapi.GetJSON(ctx, endpoint.String(), apiKey, &response)
	if err != nil && !optionalCapabilityEndpoint(err) {
		return result, fmt.Errorf("查询音色失败: %w", err)
	}
	if err == nil {
		if err := validateVoices(response.Data); err != nil {
			return result, err
		}
		if len(response.Data) > 0 {
			result.Voices = response.Data
			result.VoiceSource = "remote"
			return result, nil
		}
	}
	lower := strings.ToLower(model)
	if result.Protocol == "mlx" && strings.Contains(lower, "qwen3-tts") && strings.Contains(lower, "customvoice") {
		result.Voices = qwenBuiltinVoices()
		result.VoiceSource = "qwen_builtin"
	}
	return result, nil
}

func capabilitiesBaseURL(raw, model string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > capabilitiesMaxURLBytes {
		return nil, fmt.Errorf("请填写有效的语音服务地址")
	}
	base, err := url.Parse(raw)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("语音服务地址必须是 HTTP 或 HTTPS 地址，不可包含账号、查询参数或片段")
	}
	if model == "" || len(model) > capabilitiesMaxModelBytes || strings.TrimSpace(model) != model || strings.ContainsFunc(model, unicode.IsControl) {
		return nil, fmt.Errorf("请选择有效的语音模型")
	}
	base.Path = strings.TrimRight(base.Path, "/")
	base.Path = strings.TrimSuffix(base.Path, "/audio/speech")
	base.RawPath = ""
	return base, nil
}

func fetchSpeechSchema(ctx context.Context, base *url.URL, key string) (map[string]any, error) {
	endpoint := base.Clone()
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/v1") + "/openapi.json"
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]any `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := remoteapi.GetJSON(ctx, endpoint.String(), key, &document); err != nil {
		if optionalCapabilityEndpoint(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询语音服务能力失败: %w", err)
	}
	return document.Components.Schemas["SpeechRequest"].Properties, nil
}

func configureCapabilities(result *Capabilities, model string, schema map[string]any) {
	lower := strings.ToLower(model)
	_, instruct := schema["instruct"]
	_, language := schema["lang_code"]
	_, audio := schema["ref_audio"]
	_, text := schema["ref_text"]
	// MLX 的 SpeechRequest 使用独有字段组合；仅标准 OpenAI instructions 不足以识别它。
	if !(audio && text || instruct && language) {
		result.Instructions = strings.HasPrefix(lower, "gpt-4o-mini-tts")
		return
	}
	result.Protocol = "mlx"
	if language {
		result.Languages = []string{"Auto", "Chinese", "English", "Japanese", "Korean", "German", "French", "Russian", "Portuguese", "Spanish", "Italian"}
	}
	qwen := strings.Contains(lower, "qwen3-tts")
	result.Instructions = instruct && qwen && strings.Contains(lower, "1.7b") && strings.Contains(lower, "customvoice")
}

func validateVoices(voices []Voice) error {
	if len(voices) > capabilitiesMaxVoices {
		return fmt.Errorf("音色列表过大")
	}
	for _, voice := range voices {
		if voice.ID == "" || len(voice.ID) > capabilitiesMaxModelBytes || strings.TrimSpace(voice.ID) != voice.ID || strings.ContainsFunc(voice.ID, unicode.IsControl) || len(voice.Name) > capabilitiesMaxModelBytes || strings.ContainsFunc(voice.Name, unicode.IsControl) {
			return fmt.Errorf("服务返回了无效的音色列表格式")
		}
	}
	return nil
}

func qwenBuiltinVoices() []Voice {
	// 性别、特点和原生语言来自 Qwen 官方音色表。
	// https://huggingface.co/Qwen/Qwen3-TTS-12Hz-0.6B-CustomVoice#supported-speakers
	return []Voice{
		{"Vivian", "Vivian（中文·年轻女声·明亮）"},
		{"Serena", "Serena（中文·年轻女声·温暖柔和）"},
		{"Uncle_Fu", "Uncle_Fu（中文·成熟男声·醇厚音色）"},
		{"Dylan", "Dylan（中文·北京·青年男声）"},
		{"Eric", "Eric（中文·四川成都·男声·活泼）"},
		{"Ryan", "Ryan（英语·男声·动感且富有节奏）"},
		{"Aiden", "Aiden（英语·美式男声·阳光）"},
		{"Ono_Anna", "Ono_Anna（日语·女声·俏皮）"},
		{"Sohee", "Sohee（韩语·女声·温暖）"},
	}
}

func optionalCapabilityEndpoint(err error) bool {
	status, ok := errors.AsType[*remoteapi.HTTPError](err)
	return ok && (status.StatusCode == 404 || status.StatusCode == 405)
}
