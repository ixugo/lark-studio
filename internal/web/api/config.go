package api

import (
	"net/http"

	"github.com/ixugo/goddd/pkg/reason"
	"github.com/ixugo/vdub/internal/conf"
)

type configOutput struct {
	Pipeline conf.Pipeline `json:"pipeline"`
	LLM      llmOutput     `json:"llm"`
	TTS      conf.TTS      `json:"tts"`
	LipSync  lipSyncOutput `json:"lip_sync"`
}

type lipSyncOutput struct {
	Enabled bool   `json:"Enabled"`
	BaseURL string `json:"BaseURL"`
	APIKey  string `json:"APIKey"`
}

type llmOutput struct {
	Provider  string `json:"provider"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	DeepLXURL string `json:"deeplx_url"`
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func (uc *Usecase) getConfig(_ *http.Request, _ *struct{}) (configOutput, error) {
	c := uc.Conf
	return configOutput{
		Pipeline: c.Pipeline,
		LLM: llmOutput{
			Provider:  c.LLM.Provider,
			BaseURL:   c.LLM.BaseURL,
			APIKey:    maskKey(c.LLM.APIKey),
			Model:     c.LLM.Model,
			DeepLXURL: c.LLM.DeepLXURL,
		},
		TTS: c.TTS,
		LipSync: lipSyncOutput{
			Enabled: c.LipSync.Enabled,
			BaseURL: c.LipSync.BaseURL,
			APIKey:  maskKey(c.LipSync.APIKey),
		},
	}, nil
}

type updateConfigInput struct {
	Pipeline *pipelineInput `json:"pipeline,omitempty"`
	LLM      *llmInput      `json:"llm,omitempty"`
	TTS      *ttsInput      `json:"tts,omitempty"`
	LipSync  *lipSyncInput  `json:"lip_sync,omitempty"`
}

type lipSyncInput struct {
	Enabled *bool   `json:"enabled,omitempty"`
	BaseURL *string `json:"base_url,omitempty"`
	APIKey  *string `json:"api_key,omitempty"`
}

type pipelineInput struct {
	Workers            *int     `json:"workers,omitempty"`
	WhisperMode        *string  `json:"whisper_mode,omitempty"`
	WhisperModel       *string  `json:"whisper_model,omitempty"`
	FFmpegBin          *string  `json:"ffmpeg_bin,omitempty"`
	DefaultTargetLang  *string  `json:"default_target_lang,omitempty"`
	TranslatePrompt    *string  `json:"translate_prompt,omitempty"`
	MaxSpeedFactor     *float64 `json:"max_speed_factor,omitempty"`
	TranslateChunkSize *int     `json:"translate_chunk_size,omitempty"`
	TTSWorkers         *int     `json:"tts_workers,omitempty"`
	CleanIntermediate  *bool    `json:"clean_intermediate,omitempty"`
	SubtitleOutput     *string  `json:"subtitle_output,omitempty"`
}

type llmInput struct {
	Provider  *string `json:"provider,omitempty"`
	BaseURL   *string `json:"base_url,omitempty"`
	APIKey    *string `json:"api_key,omitempty"`
	Model     *string `json:"model,omitempty"`
	DeepLXURL *string `json:"deeplx_url,omitempty"`
}

type ttsInput struct {
	Type    *string `json:"type,omitempty"`
	Voice   *string `json:"voice,omitempty"`
	BaseURL *string `json:"base_url,omitempty"`
	APIKey  *string `json:"api_key,omitempty"`
	Model   *string `json:"model,omitempty"`
}

func (uc *Usecase) updateConfig(_ *http.Request, in *updateConfigInput) (configOutput, error) {
	next := *uc.Conf
	c := &next
	if in.Pipeline != nil && in.Pipeline.Workers != nil {
		if err := conf.ValidateWorkers(*in.Pipeline.Workers); err != nil {
			return configOutput{}, reason.ErrBadRequest.Withf("%s", err)
		}
	}

	if p := in.Pipeline; p != nil {
		if p.Workers != nil {
			c.Pipeline.Workers = *p.Workers
		}
		if p.WhisperMode != nil {
			c.Pipeline.WhisperMode = "whisper-cpp"
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
		if p.TranslateChunkSize != nil {
			c.Pipeline.TranslateChunkSize = *p.TranslateChunkSize
		}
		if p.TTSWorkers != nil {
			c.Pipeline.TTSWorkers = *p.TTSWorkers
		}
		if p.CleanIntermediate != nil {
			c.Pipeline.CleanIntermediate = *p.CleanIntermediate
		}
		if p.SubtitleOutput != nil {
			c.Pipeline.SubtitleOutput = *p.SubtitleOutput
		}
	}

	if l := in.LLM; l != nil {
		if l.Provider != nil {
			c.LLM.Provider = *l.Provider
		}
		if l.BaseURL != nil {
			c.LLM.BaseURL = *l.BaseURL
		}
		if l.APIKey != nil {
			c.LLM.APIKey = *l.APIKey
		}
		if l.Model != nil {
			c.LLM.Model = *l.Model
		}
		if l.DeepLXURL != nil {
			c.LLM.DeepLXURL = *l.DeepLXURL
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

	if ls := in.LipSync; ls != nil {
		if ls.Enabled != nil {
			c.LipSync.Enabled = *ls.Enabled
		}
		if ls.BaseURL != nil {
			c.LipSync.BaseURL = *ls.BaseURL
		}
		if ls.APIKey != nil {
			c.LipSync.APIKey = *ls.APIKey
		}
	}

	if err := conf.WriteConfig(c, c.Runtime.ConfigPath); err != nil {
		return configOutput{}, reason.ErrServer.Withf("保存配置失败: %s", err)
	}

	*uc.Conf = next
	if uc.Scheduler != nil && in.Pipeline != nil && in.Pipeline.Workers != nil {
		if err := uc.Scheduler.SetWorkers(c.Pipeline.Workers); err != nil {
			return configOutput{}, err
		}
	}
	return configOutput{
		Pipeline: c.Pipeline,
		LLM: llmOutput{
			BaseURL: c.LLM.BaseURL,
			APIKey:  maskKey(c.LLM.APIKey),
			Model:   c.LLM.Model,
		},
		TTS: c.TTS,
		LipSync: lipSyncOutput{
			Enabled: c.LipSync.Enabled,
			BaseURL: c.LipSync.BaseURL,
			APIKey:  maskKey(c.LipSync.APIKey),
		},
	}, nil
}
