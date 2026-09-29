package pipeline

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// 上下文窗口大小：每次翻译时携带前后各 3 句供 LLM 参考语境
const contextWindow = 3

// ttsPair 翻译→TTS 的流水线数据单元
type ttsPair struct {
	index int
	text  string
}

// ttsPipelineState 汇总并发配音 worker 的进度和首个错误。
type ttsPipelineState struct {
	done   atomic.Int32
	logged atomic.Int32
	err    error
	once   sync.Once
	wg     sync.WaitGroup
}

// ttsWorkerConfig 收拢 worker 的共享输入，避免并发函数参数过多。
type ttsWorkerConfig struct {
	ctx      context.Context
	job      Job
	input    <-chan ttsPair
	audioDir string
	total    int
	state    *ttsPipelineState
}

// translationRange 描述当前翻译分块的半开区间。
type translationRange struct {
	start int
	end   int
}

// runTranslate 翻译步骤
// ModeDub 模式先检查译文和实际配音长度，再发布通过校验的文本。
func (c *Core) runTranslate(ctx context.Context, job Job) error {
	srcSRT := filepath.Join(job.OutputDir, "src.srt")
	data, err := os.ReadFile(srcSRT)
	if err != nil {
		return fmt.Errorf("读取字幕失败: %w", err)
	}

	entries := parseSRT(string(data))
	if len(entries) == 0 {
		return fmt.Errorf("字幕为空")
	}

	sentences := make([]string, len(entries))
	for i, e := range entries {
		sentences[i] = e.Text
	}

	if job.Mode == ModeDub {
		return c.translateAndTTS(ctx, job, entries, sentences)
	}
	return c.translateOnly(ctx, job, entries, sentences)
}

// translateOnly 仅翻译（ModeTranslate 用），不做 TTS
func (c *Core) translateOnly(ctx context.Context, job Job, entries []srtEntry, sentences []string) error {
	c.llmMu.Lock()
	defer c.llmMu.Unlock()

	c.logEvent(job.TaskID, "info", StepTranslate, "翻译开始：%d 句 → %s", len(sentences), job.TargetLang)

	translated, err := c.translateAllChunks(ctx, job, sentences, nil, entries)
	if err != nil {
		return err
	}

	if err := c.writeTranslationOutputs(entries, translated, job.OutputDir); err != nil {
		return &translationQualityError{err: err}
	}
	c.logEvent(job.TaskID, "success", StepTranslate, "翻译完成，时间轴 1:1 对齐")
	return nil
}

// translateAndTTS 在质量检查通过后发布最终配音，消费者通过文本指纹复用已测量音频。
func (c *Core) translateAndTTS(ctx context.Context, job Job, entries []srtEntry, sentences []string) error {
	prepared, err := c.prepareSpeechContext(ctx, job)
	if err != nil {
		return err
	}
	ctx = prepared
	ttsWorkers := normalizedTTSWorkers(c.cfg.TTSWorkers)
	ttsCh := make(chan ttsPair, 300)
	audioDir := filepath.Join(job.OutputDir, "audio_segs")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return fmt.Errorf("创建音频目录失败: %w", err)
	}

	total := len(sentences)
	c.logEvent(job.TaskID, "info", StepTranslate, "翻译开始：%d 句 → %s", total, job.TargetLang)
	c.logEvent(job.TaskID, "info", StepTTS, "配音准备：%d 句，并发 %d", total, ttsWorkers)
	state := new(ttsPipelineState)
	c.ttsMu.Lock()
	workerConfig := ttsWorkerConfig{
		ctx: ctx, job: job, input: ttsCh, audioDir: audioDir, total: total, state: state,
	}
	c.startTTSWorkers(ttsWorkers, workerConfig)

	c.llmMu.Lock()
	translated, translateErr := c.translateAllChunks(ctx, job, sentences, ttsCh, entries)
	c.llmMu.Unlock()
	close(ttsCh)
	state.wg.Wait()
	c.ttsMu.Unlock()

	if translateErr != nil {
		return translateErr
	}
	if state.err != nil {
		return &translationQualityError{err: state.err}
	}
	if err := c.writeTranslationOutputs(entries, translated, job.OutputDir); err != nil {
		return &translationQualityError{err: err}
	}
	c.logEvent(job.TaskID, "success", StepTranslate, "翻译完成，时间轴 1:1 对齐")
	c.logEvent(job.TaskID, "success", StepTTS, "配音完成：%d 段", state.done.Load())
	return nil
}

// normalizedTTSWorkers 将旧配置零值恢复为默认并限制并发上限。
func normalizedTTSWorkers(workers int) int {
	if workers <= 0 {
		return 2
	}
	return min(4, workers)
}

// startTTSWorkers 启动固定数量的配音消费者。
func (c *Core) startTTSWorkers(workers int, cfg ttsWorkerConfig) {
	for range workers {
		cfg.state.wg.Add(1)
		go c.consumeTTS(cfg)
	}
}

// consumeTTS 合成队列中的字幕，并只保存第一个错误。
func (c *Core) consumeTTS(cfg ttsWorkerConfig) {
	defer cfg.state.wg.Done()
	for pair := range cfg.input {
		if cfg.ctx.Err() != nil {
			cfg.state.once.Do(func() { cfg.state.err = cfg.ctx.Err() })
			continue
		}
		if err := c.synthesizePair(cfg, pair); err != nil {
			cfg.state.once.Do(func() {
				cfg.state.err = err
			})
			continue
		}
		c.reportTTSProgress(cfg.job, int(cfg.state.done.Add(1)), cfg.total, &cfg.state.logged)
	}
}

// synthesizePair 记录实际译文，并只复用译文指纹一致的音频。
func (c *Core) synthesizePair(cfg ttsWorkerConfig, pair ttsPair) error {
	outputPath := filepath.Join(cfg.audioDir, fmt.Sprintf("%d.wav", pair.index))
	displayText := strings.Join(strings.Fields(pair.text), " ")
	c.logEvent(
		cfg.job.TaskID,
		"info",
		StepTTS,
		"配音第 %d/%d 句：%s",
		pair.index+1,
		cfg.total,
		displayText,
	)
	if audioSegmentMatchesText(outputPath, pair.text) {
		c.logEvent(cfg.job.TaskID, "info", StepTTS, "复用第 %d 句配音", pair.index+1)
		return nil
	}
	if err := removeStaleAudio(outputPath); err != nil {
		return fmt.Errorf("清理第 %d 句旧配音失败: %w", pair.index+1, err)
	}
	if err := c.synthesizeWithJob(cfg.ctx, cfg.job, pair.text, outputPath); err != nil {
		return fmt.Errorf("TTS 第 %d 句失败: %w", pair.index+1, err)
	}
	if err := writeAudioTextFingerprint(outputPath, pair.text); err != nil {
		return fmt.Errorf("保存第 %d 句配音指纹失败: %w", pair.index+1, err)
	}
	return nil
}

// audioSegmentMatchesText 仅在音频存在且译文指纹一致时允许断点复用。
func audioSegmentMatchesText(audioPath, text string) bool {
	info, err := os.Stat(audioPath)
	if err != nil || info.Size() == 0 {
		return false
	}
	data, err := os.ReadFile(audioPath + ".text.sha256")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == audioTextFingerprint(text)
}

// writeAudioTextFingerprint 保存配音实际输入的稳定指纹。
func writeAudioTextFingerprint(audioPath, text string) error {
	return os.WriteFile(
		audioPath+".text.sha256",
		[]byte(audioTextFingerprint(text)),
		0o644,
	)
}

// audioTextFingerprint 生成译文内容的 SHA-256 指纹。
func audioTextFingerprint(text string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
}

// removeStaleAudio 删除旧音频与指纹，避免译文变化后仍沿用旧配音。
func removeStaleAudio(audioPath string) error {
	for _, path := range []string{audioPath, audioPath + ".text.sha256"} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// translateAllChunks 先校验整批字幕再发布配音，保证跨分块重复不会提前被朗读。
func (c *Core) translateAllChunks(
	ctx context.Context, job Job, sentences []string, streamTo chan<- ttsPair, timing ...[]srtEntry,
) ([]string, error) {
	prompt := c.cfg.TranslatePrompt
	if c.semanticSplitReady(job.Translator) {
		prompt = c.injectTermsIntoPrompt(ctx, prompt, sentences)
	}
	review := &translationReview{core: c, job: job, source: sentences, prompt: prompt}
	if len(timing) > 0 {
		review.entries = timing[0]
	}
	if err := review.translateInitial(ctx); err != nil {
		return nil, err
	}
	if err := review.run(ctx); err != nil {
		return nil, &translationQualityError{err: err}
	}
	streamTranslationChunk(streamTo, 0, review.translated)
	c.notifier.OnProgress(job.TaskID, StepTranslate, 100)
	return review.translated, nil
}

// translateInitial 按分块预算请求首版译文，条数不符时拒绝位置漂移。
func (r *translationReview) translateInitial(ctx context.Context) error {
	chunkSize := r.core.cfg.TranslateChunkSize
	if chunkSize <= 0 {
		chunkSize = 10
	}
	for start := 0; start < len(r.source); start += chunkSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		span := translationRange{start, min(start+chunkSize, len(r.source))}
		lines, err := r.core.translateChunk(ctx, r.job, r.source, r.budgetPrompt(span), span)
		if err != nil {
			return fmt.Errorf("翻译第 %d-%d 条失败: %w", start+1, span.end, err)
		}
		if err := validateTranslations(lines, span.end-span.start); err != nil {
			return err
		}
		r.translated = append(r.translated, lines...)
		for index, line := range lines {
			i := start + index
			r.core.logEvent(r.job.TaskID, "info", StepTranslate, "[翻译 %d/%d]\n原文: %s\n译文: %s", i+1, len(r.source), r.source[i], line)
		}
		r.core.notifier.OnProgress(r.job.TaskID, StepTranslate, span.end*translationInitialProgress/len(r.source))
	}
	return nil
}

// translateChunk 翻译一个分块，并携带前后文供模型理解语境。
func (c *Core) translateChunk(
	ctx context.Context,
	job Job,
	sentences []string,
	prompt string,
	chunkRange translationRange,
) ([]string, error) {
	start, end := chunkRange.start, chunkRange.end
	if job.Translator == "google" || job.Translator == "bing" || job.Translator == "deeplx" {
		return c.translateWithJob(ctx, job, sentences[start:end], "", nil, nil)
	}
	contextStart := max(0, start-contextWindow)
	contextEnd := min(len(sentences), end+contextWindow)
	return c.translateWithJob(
		ctx,
		job,
		sentences[start:end],
		prompt,
		sentences[contextStart:start],
		sentences[end:contextEnd],
	)
}

// streamTranslationChunk 将刚翻译的分块立即送入配音队列。
func streamTranslationChunk(output chan<- ttsPair, start int, translated []string) {
	if output == nil {
		return
	}
	for index, text := range translated {
		output <- ttsPair{index: start + index, text: text}
	}
}

// reportTTSProgress 推送配音百分比，并确保同一百分比只写一行日志。
func (c *Core) reportTTSProgress(job Job, done, total int, logged *atomic.Int32) {
	progress := done * 100 / total
	c.notifier.OnProgress(job.TaskID, StepTTS, progress)
	for {
		previous := logged.Load()
		if int32(progress) <= previous {
			return
		}
		if logged.CompareAndSwap(previous, int32(progress)) {
			c.logEvent(
				job.TaskID,
				"info",
				StepTTS,
				"配音进度 %d%%（%d/%d）",
				progress,
				done,
				total,
			)
			return
		}
	}
}

// injectTermsIntoPrompt 查询术语表，将与当前字幕匹配的术语映射注入翻译 prompt
// 不区分大小写匹配：仅当术语源词完整出现在某句字幕中才注入
// 返回增强后的 prompt（空串交由 LLM 适配器使用内置默认）
func (c *Core) injectTermsIntoPrompt(ctx context.Context, basePrompt string, sentences []string) string {
	if c.termLister == nil {
		return basePrompt
	}
	allMappings, err := c.termLister.ListMappings(ctx)
	if err != nil || len(allMappings) == 0 {
		return basePrompt
	}

	matched := matchTermMappings(allMappings, sentences)
	if len(matched) == 0 {
		return basePrompt
	}

	var parts []string
	for _, m := range matched {
		if strings.EqualFold(m.Text, m.Translation) {
			parts = append(parts, fmt.Sprintf("%q → keep as-is", m.Text))
		} else {
			parts = append(parts, fmt.Sprintf("%q → %q", m.Text, m.Translation))
		}
	}

	termLine := fmt.Sprintf(
		"\n- MANDATORY terminology: for each term below, when you encounter it (case-insensitive), you MUST translate it EXACTLY as specified: %s",
		strings.Join(parts, "; "),
	)

	if basePrompt != "" {
		return basePrompt + termLine
	}
	return `Translate the numbered subtitle sentences to {{target_lang}}.
Output exactly {{count}} numbered translated lines.
Keep each translation concise, natural, and faithful to the original.
Do not repeat the source text unless it is already written in the target language.` + termLine
}

// matchTermMappings 从术语映射中筛出在 sentences 中出现的条目（不区分大小写）
func matchTermMappings(mappings []TermMapping, sentences []string) []TermMapping {
	corpus := strings.ToLower(strings.Join(sentences, " "))
	var matched []TermMapping
	seen := make(map[string]bool, len(mappings))
	for _, m := range mappings {
		lower := strings.ToLower(m.Text)
		if seen[lower] {
			continue
		}
		if strings.Contains(corpus, lower) {
			matched = append(matched, m)
			seen[lower] = true
		}
	}
	return matched
}

// writeTranslationOutputs 写入翻译文本和翻译字幕文件
func (c *Core) writeTranslationOutputs(entries []srtEntry, translated []string, outputDir string) error {
	transFile := filepath.Join(outputDir, "trans.txt")
	if err := os.WriteFile(transFile, []byte(strings.Join(translated, "\n")), 0o644); err != nil {
		return fmt.Errorf("写入翻译结果失败: %w", err)
	}

	transSRT := filepath.Join(outputDir, "trans.srt")
	if err := buildTransSRT(entries, translated, transSRT); err != nil {
		return fmt.Errorf("生成翻译字幕失败: %w", err)
	}
	return nil
}

// buildTransSRT 将翻译文本按原始时间轴写入 SRT，自动对长行做语义断行
func buildTransSRT(entries []srtEntry, translated []string, outputPath string) error {
	var sb strings.Builder
	for i, entry := range entries {
		text := entry.Text
		if i < len(translated) {
			text = semanticBreak(translated[i])
		}
		sb.WriteString(fmt.Sprintf("%d\n%s --> %s\n%s\n\n", i+1, entry.Start, entry.End, text))
	}
	return os.WriteFile(outputPath, []byte(sb.String()), 0o644)
}
