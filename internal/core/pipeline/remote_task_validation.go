package pipeline

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ixugo/vdub/internal/adapter/remoteapi"
	ttsadapter "github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/conf"
)

// ValidateRemoteTask 在写任务、暂存输入或清理重跑产物前复核实际需要的远程模型和声音。
func ValidateRemoteTask(ctx context.Context, job Job, bc *conf.Bootstrap) error {
	if bc == nil {
		return fmt.Errorf("任务配置不可用")
	}
	if strings.TrimSpace(job.InputPath) == "" {
		return fmt.Errorf("输入文件路径不能为空")
	}
	if job.Mode < ModeSubtitle || job.Mode > ModeTextTranslate {
		return fmt.Errorf("任务配方无效")
	}
	if err := ValidateResourceMode(job.InputPath, job.Mode); err != nil {
		return err
	}
	job.Translator = normalizedRemoteEngine(cmp.Or(job.Translator, bc.LLM.Provider))
	job.TTSEngine = normalizedRemoteEngine(cmp.Or(job.TTSEngine, bc.TTS.Type))
	steps := remoteValidationSteps(job, bc)
	needed := steps[resumeStepIndex(steps, job.ResumeFrom):]
	if slices.Contains(needed, StepWhisper) && !hasTaskRecognitionSubtitle(job) && normalizedRemoteEngine(bc.Pipeline.WhisperMode) == "openai" {
		if err := remoteapi.ValidateSelection(ctx, bc.Pipeline.ASRBaseURL, bc.Pipeline.ASRAPIKey, bc.Pipeline.ASRModel); err != nil {
			return fmt.Errorf("语音识别模型校验失败: %w", err)
		}
	}
	if slices.Contains(needed, StepTranslate) && job.Translator == "openai" {
		if err := remoteapi.ValidateSelection(ctx, bc.LLM.BaseURL, bc.LLM.APIKey, bc.LLM.Model); err != nil {
			return fmt.Errorf("翻译模型校验失败: %w", err)
		}
	}
	if slices.Contains(needed, StepTTS) && job.TTSEngine == "openai" {
		return validateRemoteTaskSpeech(ctx, job, bc.TTS)
	}
	return nil
}

func normalizedRemoteEngine(engine string) string {
	engine = strings.ToLower(strings.TrimSpace(engine))
	if engine == "local" {
		return "openai"
	}
	return engine
}

func remoteValidationSteps(job Job, bc *conf.Bootstrap) []string {
	core := Core{cfg: Config{SubtitleOutput: bc.Pipeline.SubtitleOutput}}
	core.semanticSplitConfigured.Store(strings.EqualFold(bc.LLM.Provider, "openai") && strings.TrimSpace(bc.LLM.BaseURL) != "" && strings.TrimSpace(bc.LLM.Model) != "")
	steps := core.buildSteps(job)
	// 节点表不创建真实客户端；开启的对口型节点仍应参与断点判断。
	if job.Mode == ModeDub && bc.LipSync.Enabled {
		if index := slices.Index(steps, StepBurn); index >= 0 {
			steps = slices.Insert(steps, index, StepLipSync)
		}
	}
	return steps
}

func hasTaskRecognitionSubtitle(job Job) bool {
	paths := []string{strings.TrimSuffix(job.InputPath, filepath.Ext(job.InputPath)) + ".srt"}
	if job.OutputDir != "" {
		paths = append(paths, filepath.Join(job.OutputDir, "src.srt"))
	}
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Size() > 0 {
			return true
		}
	}
	return false
}

func validateRemoteTaskSpeech(ctx context.Context, job Job, config conf.TTS) error {
	voice := cmp.Or(job.TTSVoice, conf.VoiceForEngine(config, "openai"))
	options := ttsadapter.SpeechOptions{Protocol: config.Protocol, Language: config.Language, Instructions: config.Instructions}
	if err := ttsadapter.ValidateSpeechSelection(ctx, config.BaseURL, config.APIKey, config.Model, voice, options); err != nil {
		return fmt.Errorf("语音合成模型或音色校验失败: %w", err)
	}
	return nil
}
