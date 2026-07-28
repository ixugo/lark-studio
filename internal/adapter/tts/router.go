package tts

import (
	"context"
	"fmt"
)

// Router 按任务参数选择 Edge 或 OpenAI 配音。
type Router struct {
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
	if defaultEngine == "" {
		defaultEngine = "edge"
	}
	return &Router{
		defaultEngine: defaultEngine,
		edge:          NewEdgeTTS(voice),
		openAI:        NewOpenAITTS(baseURL, apiKey, model, voice),
	}
}

// Synthesize 使用全局默认引擎和正常语速，兼容基础流水线接口。
func (r *Router) Synthesize(ctx context.Context, text, outputPath, voice string) error {
	return r.SynthesizeWithOptions(ctx, text, outputPath, r.defaultEngine, voice, 1)
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
	if engine == "" {
		engine = r.defaultEngine
	}
	switch engine {
	case "edge":
		return r.edge.SynthesizeWithSpeed(ctx, text, outputPath, voice, speed)
	case "openai":
		return r.openAI.SynthesizeWithSpeed(ctx, text, outputPath, voice, speed)
	default:
		return fmt.Errorf("不支持的配音引擎: %s", engine)
	}
}
