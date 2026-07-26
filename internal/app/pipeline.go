package app

import (
	"github.com/ixugo/vdub/internal/adapter/llm"
	"github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/adapter/whisper"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
)

// NewPipelineCore 根据配置创建完整的流水线核心，组装所有适配器
func NewPipelineCore(bc *conf.Bootstrap) *pipeline.Core {
	cfg := pipeline.Config{
		WhisperBin:   bc.Pipeline.WhisperBin,
		WhisperModel: bc.Pipeline.WhisperModel,
		FFmpegBin:    bc.Pipeline.FFmpegBin,
	}

	var whisperRunner pipeline.WhisperRunner
	switch bc.Pipeline.WhisperMode {
	case "whisper-cpp":
		whisperRunner = whisper.NewRunner(bc.Pipeline.WhisperBin, bc.Pipeline.WhisperModel)
	default:
		whisperRunner = whisper.NewFFmpegRunner(bc.Pipeline.FFmpegBin, bc.Pipeline.WhisperModel)
	}

	llmClient := llm.NewClient(bc.LLM.BaseURL, bc.LLM.APIKey, bc.LLM.Model)

	var ttsClient pipeline.TTSClient
	switch bc.TTS.Type {
	case "openai":
		ttsClient = tts.NewOpenAITTS(bc.TTS.BaseURL, bc.TTS.APIKey, bc.TTS.Model, bc.TTS.Voice)
	default:
		ttsClient = tts.NewEdgeTTS(bc.TTS.Voice)
	}

	return pipeline.NewCore(cfg, whisperRunner, llmClient, ttsClient)
}
