package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ixugo/goddd/pkg/reason"
	"github.com/ixugo/vdub/internal/conf"
)

// 配置 API —— 仅暴露 Pipeline/LLM/TTS 三个业务相关段

type configOutput struct {
	Pipeline conf.Pipeline `json:"pipeline"`
	LLM      llmOutput     `json:"llm"`
	TTS      conf.TTS      `json:"tts"`
}

// llmOutput 遮蔽 APIKey，不将明文推到前端
type llmOutput struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func (uc *Usecase) getConfig(_ *gin.Context, _ *struct{}) (configOutput, error) {
	c := uc.Conf
	return configOutput{
		Pipeline: c.Pipeline,
		LLM: llmOutput{
			BaseURL: c.LLM.BaseURL,
			APIKey:  maskKey(c.LLM.APIKey),
			Model:   c.LLM.Model,
		},
		TTS: c.TTS,
	}, nil
}

type updateConfigInput struct {
	Pipeline *pipelineInput `json:"pipeline,omitempty"`
	LLM      *llmInput      `json:"llm,omitempty"`
	TTS      *ttsInput      `json:"tts,omitempty"`
}

type pipelineInput struct {
	Workers           *int     `json:"workers,omitempty"`
	WhisperMode       *string  `json:"whisper_mode,omitempty"`
	WhisperModel      *string  `json:"whisper_model,omitempty"`
	FFmpegBin         *string  `json:"ffmpeg_bin,omitempty"`
	DefaultTargetLang *string  `json:"default_target_lang,omitempty"`
	TranslatePrompt   *string  `json:"translate_prompt,omitempty"`
	MaxSpeedFactor    *float64 `json:"max_speed_factor,omitempty"`
}

type llmInput struct {
	BaseURL *string `json:"base_url,omitempty"`
	APIKey  *string `json:"api_key,omitempty"`
	Model   *string `json:"model,omitempty"`
}

type ttsInput struct {
	Type    *string `json:"type,omitempty"`
	Voice   *string `json:"voice,omitempty"`
	BaseURL *string `json:"base_url,omitempty"`
	APIKey  *string `json:"api_key,omitempty"`
	Model   *string `json:"model,omitempty"`
}

func (uc *Usecase) updateConfig(_ *gin.Context, in *updateConfigInput) (configOutput, error) {
	c := uc.Conf

	if p := in.Pipeline; p != nil {
		if p.Workers != nil {
			c.Pipeline.Workers = *p.Workers
		}
		if p.WhisperMode != nil {
			c.Pipeline.WhisperMode = *p.WhisperMode
		}
		if p.WhisperModel != nil {
			c.Pipeline.WhisperModel = *p.WhisperModel
		}
		if p.FFmpegBin != nil {
			c.Pipeline.FFmpegBin = *p.FFmpegBin
		}
		if p.DefaultTargetLang != nil {
			c.Pipeline.DefaultTargetLang = *p.DefaultTargetLang
		}
		if p.TranslatePrompt != nil {
			c.Pipeline.TranslatePrompt = *p.TranslatePrompt
		}
		if p.MaxSpeedFactor != nil {
			c.Pipeline.MaxSpeedFactor = *p.MaxSpeedFactor
		}
	}

	if l := in.LLM; l != nil {
		if l.BaseURL != nil {
			c.LLM.BaseURL = *l.BaseURL
		}
		if l.APIKey != nil {
			c.LLM.APIKey = *l.APIKey
		}
		if l.Model != nil {
			c.LLM.Model = *l.Model
		}
	}

	if t := in.TTS; t != nil {
		if t.Type != nil {
			c.TTS.Type = *t.Type
		}
		if t.Voice != nil {
			c.TTS.Voice = *t.Voice
		}
		if t.BaseURL != nil {
			c.TTS.BaseURL = *t.BaseURL
		}
		if t.APIKey != nil {
			c.TTS.APIKey = *t.APIKey
		}
		if t.Model != nil {
			c.TTS.Model = *t.Model
		}
	}

	if err := conf.WriteConfig(c, c.Runtime.ConfigPath); err != nil {
		return configOutput{}, reason.ErrServer.Withf("保存配置失败: %s", err)
	}

	return configOutput{
		Pipeline: c.Pipeline,
		LLM: llmOutput{
			BaseURL: c.LLM.BaseURL,
			APIKey:  maskKey(c.LLM.APIKey),
			Model:   c.LLM.Model,
		},
		TTS: c.TTS,
	}, nil
}
