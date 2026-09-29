package tts

import (
	"context"
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"github.com/ixugo/vdub/internal/adapter/remoteapi"
	"strings"
	"sync"
	"unicode/utf8"
)

// Router 按任务参数选择 Edge 或 OpenAI 配音。
type Router struct {
	mu            sync.RWMutex
	defaultEngine string
	edge          *EdgeTTS
	openAI        *OpenAITTS
}

// NewRouter 创建同时持有两种配音实现的路由器。
func NewRouter(
	defaultEngine string,
	voice string,
	baseURL string,
	apiKey string,
	model string,
) *Router {
	r := &Router{}
	r.SetTTSConfig(defaultEngine, voice, baseURL, apiKey, model)
	return r
}

// SetTTSConfig 原子替换后续请求使用的端点，不等待正在进行的网络请求。
func (r *Router) SetTTSConfig(defaultEngine, voice, baseURL, apiKey, model string, options ...SpeechOptions) {
	if defaultEngine == "" {
		defaultEngine = "edge"
	}
	edge := NewEdgeTTS(voice)
	openAI := NewOpenAITTS(baseURL, apiKey, model, voice)
	if len(options) > 0 {
		openAI.options = options[0]
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultEngine, r.edge, r.openAI = defaultEngine, edge, openAI
}

// Synthesize 使用全局默认引擎和正常语速，兼容基础流水线接口。
func (r *Router) Synthesize(ctx context.Context, text, outputPath, voice string) error {
	return r.SynthesizeWithOptions(ctx, text, outputPath, "", voice, 1)
}

// SynthesizeWithOptions 使用任务快照中的引擎、音色和语速。
func (r *Router) SynthesizeWithOptions(
	ctx context.Context,
	text string,
	outputPath string,
	engine string,
	voice string,
	speed float64,
) error {
	r.mu.RLock()
	defaultEngine, edge, openAI := r.defaultEngine, r.edge, r.openAI
	r.mu.RUnlock()
	if task, ok := ctx.Value(speechTaskKey{}).(speechTask); ok {
		defaultEngine, edge, openAI = task.engine, task.edge, task.openAI
	}

	if engine == "" {
		engine = defaultEngine
	}
	switch engine {
	case "edge":
		return edge.SynthesizeWithSpeed(ctx, text, outputPath, voice, speed)
	case "openai":
		return openAI.SynthesizeWithSpeed(ctx, text, outputPath, voice, speed)
	default:
		return fmt.Errorf("不支持的配音引擎: %s", engine)
	}
}

// speechTask 将单个任务的客户端与声音参数固定下来，避免配置热更新影响后半段配音。
type speechTaskKey struct{}
type speechTask struct {
	engine string
	edge   *EdgeTTS
	openAI *OpenAITTS
}

func (r *Router) SnapshotTask(ctx context.Context, engine, targetLang, voice string) (context.Context, error) {
	r.mu.RLock()
	task := speechTask{engine: engine, edge: r.edge, openAI: r.openAI}
	if task.engine == "" {
		task.engine = r.defaultEngine
	}
	r.mu.RUnlock()
	if task.engine != "openai" {
		return context.WithValue(ctx, speechTaskKey{}, task), nil
	}
	client := *task.openAI
	task.openAI = &client
	if err := remoteapi.ValidateSelection(ctx, client.baseURL, client.apiKey, client.model); err != nil {
		return ctx, err
	}
	if client.options.Protocol == "mlx" {
		cap, err := DiscoverCapabilities(ctx, client.baseURL, client.apiKey, client.model)
		if err != nil {
			return ctx, err
		}

		client.options.Protocol = cap.Protocol
		client.options.Language = taskSpeechLanguage(client.options, targetLang)
		if err := validateSpeechCapability(cap, client.options, firstVoice(voice, client.voice)); err != nil {
			return ctx, err
		}
	}
	client.options.Language = taskSpeechLanguage(client.options, targetLang)
	return context.WithValue(ctx, speechTaskKey{}, task), nil
}

func firstVoice(selected, fallback string) string {
	if selected != "" {
		return selected
	}
	return fallback
}

// SpeechIdentity 只包含声音生成参数，密钥不写入磁盘或日志。
func (r *Router) SpeechIdentity(ctx context.Context, voice string, speed float64) string {
	task, ok := ctx.Value(speechTaskKey{}).(speechTask)
	if !ok {
		return ""
	}
	identity := struct {
		Engine, Endpoint, Model, Voice string
		Speed                          float64
		Options                        SpeechOptions
	}{Engine: task.engine, Voice: voice, Speed: speed}
	if task.engine == "openai" {
		identity.Endpoint = task.openAI.baseURL
		identity.Model = task.openAI.model
		identity.Options = task.openAI.options
		identity.Voice = firstVoice(voice, task.openAI.voice)
	} else {
		identity.Voice = firstVoice(voice, task.edge.voice)
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func taskSpeechLanguage(options SpeechOptions, target string) string {
	if options.Protocol != "mlx" {
		return ""
	}
	if options.Language != "" && !strings.EqualFold(options.Language, "auto") {
		return options.Language
	}
	language, _, _ := strings.Cut(strings.ToLower(target), "-")
	names := map[string]string{"zh": "Chinese", "en": "English", "ja": "Japanese", "ko": "Korean", "de": "German", "fr": "French", "ru": "Russian", "pt": "Portuguese", "es": "Spanish", "it": "Italian"}
	if name, ok := names[language]; ok {
		return name
	}
	return "Auto"
}

// ValidateSpeechOptions 拒绝服务没有声明的扩展参数，而不是静默忽略用户选择。
func ValidateSpeechOptions(ctx context.Context, baseURL, key, model string, options SpeechOptions) error {
	if options.Protocol != "" && options.Protocol != "openai" && options.Protocol != "mlx" {
		return fmt.Errorf("无效的语音服务协议")
	}
	if utf8.RuneCountInString(options.Instructions) > 4096 || len(options.Language) > 64 {
		return fmt.Errorf("语音指令或语言过长")
	}
	if options.Instructions == "" && options.Protocol != "mlx" {
		return nil
	}
	cap, err := DiscoverCapabilities(ctx, baseURL, key, model)
	if err != nil {
		return err
	}
	if options.Protocol == "mlx" && cap.Protocol != "mlx" {
		return fmt.Errorf("当前服务没有声明 MLX 语音扩展")
	}
	if options.Instructions != "" && !cap.Instructions {
		return fmt.Errorf("当前语音模型不支持情绪指令，请选择支持该能力的模型")
	}

	return nil
}

func validateSpeechCapability(cap Capabilities, options SpeechOptions, voice string) error {

	if options.Instructions != "" && !cap.Instructions {
		return fmt.Errorf("当前模型不支持情绪指令")
	}
	if cap.Protocol == "mlx" && options.Language != "" && !strings.EqualFold(options.Language, "auto") {
		found := false
		for _, language := range cap.Languages {
			if language == options.Language {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("当前模型不支持配音语言 %s", options.Language)
		}
	}
	if strings.TrimSpace(voice) == "" || len(voice) > 200 {
		return fmt.Errorf("请选择有效音色")
	}
	if len(cap.Voices) > 0 {
		found := false
		for _, candidate := range cap.Voices {
			if candidate.ID == voice {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("当前模型不支持音色 %s", voice)
		}
	}

	return nil
}

// ValidateSpeechSelection 校验模型、音色、语言及扩展能力，供配置保存和试听共用。
func ValidateSpeechSelection(ctx context.Context, baseURL, key, model, voice string, options SpeechOptions) error {
	if err := remoteapi.ValidateSelection(ctx, baseURL, key, model); err != nil {
		return err
	}
	if err := ValidateSpeechOptions(ctx, baseURL, key, model, options); err != nil {
		return err
	}
	cap, err := DiscoverCapabilities(ctx, baseURL, key, model)
	if err != nil {
		return err
	}
	return validateSpeechCapability(cap, options, voice)
}
