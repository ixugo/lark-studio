package pipeline

import "context"

// translateWithJob 使用任务快照选择翻译引擎，旧适配器保持兼容。
func (c *Core) translateWithJob(
	ctx context.Context,
	job Job,
	sentences []string,
	prompt string,
	contextBefore []string,
	contextAfter []string,
) ([]string, error) {
	if routed, ok := c.llm.(RoutedLLMClient); ok && job.Translator != "" {
		return routed.TranslateWithProvider(
			ctx,
			sentences,
			job.TargetLang,
			prompt,
			contextBefore,
			contextAfter,
			job.Translator,
		)
	}
	return c.llm.Translate(
		ctx,
		sentences,
		job.TargetLang,
		prompt,
		contextBefore,
		contextAfter,
	)
}

// synthesizeWithJob 使用任务快照选择配音引擎、音色和语速。
func (c *Core) synthesizeWithJob(
	ctx context.Context,
	job Job,
	text string,
	outputPath string,
) error {
	if routed, ok := c.tts.(RoutedTTSClient); ok {
		return routed.SynthesizeWithOptions(
			ctx,
			text,
			outputPath,
			job.TTSEngine,
			job.TTSVoice,
			job.SpeechRate,
		)
	}
	return c.tts.Synthesize(ctx, text, outputPath, job.TTSVoice)
}
