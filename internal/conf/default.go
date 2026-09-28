package conf

import (
	"time"

	"github.com/ixugo/goddd/pkg/orm"
)

func DefaultConfig() Bootstrap {
	return Bootstrap{
		Server: Server{
			HTTP: ServerHTTP{
				Port:      9523,
				Timeout:   Duration(120 * time.Second),
				JwtSecret: orm.GenerateRandomString(32),
				PProf: ServerPPROF{
					Enabled:   true,
					AccessIps: []string{"::1", "127.0.0.1"},
				},
			},
		},
		Data: Data{
			Database: Database{
				Dsn:             "vdub.db",
				MaxIdleConns:    1,
				MaxOpenConns:    1,
				ConnMaxLifetime: Duration(6 * time.Hour),
				SlowThreshold:   Duration(200 * time.Millisecond),
			},
		},
		Log: Log{
			Dir:          "./logs",
			Level:        "debug",
			MaxAge:       7,
			RotationTime: Duration(8 * time.Hour),
			MaxSize:      50,
			Compress:     false,
			MaxBackups:   0,
		},
		Pipeline: Pipeline{
			Workers:            2,
			WhisperMode:        "whisper-cpp",
			WhisperBin:         "whisper-cli",
			ASRBaseURL:         "https://api.openai.com/v1",
			ASRModel:           "whisper-1",
			DefaultOutputDir:   TasksDir(),
			DefaultTargetLang:  "zh-CN",
			MaxSpeedFactor:     1.2,
			TranslateChunkSize: 10,
			TTSWorkers:         2,
			SubtitleOutput:     "soft",
		},
		LLM: LLM{
			Provider: "bing",
			BaseURL:  "http://localhost:11434/v1",
			Model:    "qwen2.5:7b",
		},
		TTS: TTS{
			Type:  "edge",
			Voice: "zh-CN-XiaoxiaoNeural",
			Model: "tts-1",
		},
	}
}
