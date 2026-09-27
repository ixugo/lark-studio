//go:build integration

package pipeline

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixugo/vdub/internal/adapter/llm"
	"github.com/ixugo/vdub/internal/adapter/tts"
)

// translationRealMetric 保存实际字幕和原速音频，便于独立核对前后效果。
type translationRealMetric struct {
	SourceIndex   int     `json:"source_index"`
	Source        string  `json:"source"`
	Before        string  `json:"before"`
	After         string  `json:"after"`
	WindowSeconds float64 `json:"window_seconds"`
	BeforeSeconds float64 `json:"before_seconds"`
	AfterSeconds  float64 `json:"after_seconds"`
	BeforeOver20  bool    `json:"before_over_20"`
	AfterOver20   bool    `json:"after_over_20"`
}

// TestTranslationQualityRealSamples 仅运行真实翻译和语音校验，保留原时间轴且不合成视频。
func TestTranslationQualityRealSamples(t *testing.T) {
	config, source, output := os.Getenv("VDUB_TRANSLATION_CONFIG"), os.Getenv("VDUB_TRANSLATION_SOURCE"), os.Getenv("VDUB_TRANSLATION_OUTPUT")
	if config == "" || source == "" || output == "" {
		t.Skip("需要 VDUB_TRANSLATION_CONFIG、VDUB_TRANSLATION_SOURCE、VDUB_TRANSLATION_OUTPUT")
	}
	cfg := readSpeechSyncConfig(t, config)
	sourceEntries := readSpeechSyncSRT(t, source, "src.srt")
	beforeEntries := readTranslationRealText(t, source)
	if len(sourceEntries) != len(beforeEntries) || len(sourceEntries) < 442 {
		t.Fatal("源字幕和历史译文不满足样本范围")
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bounds := range [][2]int{{223, 228}, {436, 442}, {36, 40}, {118, 122}, {384, 388}} {
		t.Run(fmt.Sprintf("%d-%d", bounds[0], bounds[1]), func(t *testing.T) {
			original, before := sourceEntries[bounds[0]-1:bounds[1]], beforeEntries[bounds[0]-1:bounds[1]]
			dir := prepareTranslationRealSample(t, output, original, before)
			recorder := &speechSyncRecorder{dir: dir}
			engine := tts.NewEdgeTTS("zh-CN-YunjianNeural")
			core := NewCore(Config{FFmpegBin: "ffmpeg", TranslateChunkSize: cfg.Pipeline.TranslateChunkSize, TTSWorkers: 2}, nil,
				llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model), engine, WithNotifier(recorder))
			job := Job{TaskID: t.Name(), OutputDir: dir, Mode: ModeDub, TargetLang: "zh-CN", SourceLang: "en", Translator: "openai", TTSEngine: "edge", TTSVoice: "zh-CN-YunjianNeural", SpeechRate: 1}
			if err := core.runTranslate(t.Context(), job); err != nil {
				t.Fatal(err)
			}
			if recorder.err != nil {
				t.Fatal(recorder.err)
			}
			collectTranslationRealMetrics(t, dir, original, before, engine)
		})
	}
}

// prepareTranslationRealSample 拒绝复用产物目录，防止重复实验覆盖先前证据。
func prepareTranslationRealSample(t *testing.T, output string, original, before []srtEntry) string {
	t.Helper()
	_, name, found := strings.CutLast(t.Name(), "/")
	if !found {
		name = t.Name()
	}
	dir := filepath.Join(output, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTranslationRealSample(t, dir, original, before)
	return dir
}

// writeTranslationRealSample 复制真实片段而不触碰用户原件，以便逐条还原对照。
func writeTranslationRealSample(t *testing.T, dir string, source, before []srtEntry) {
	t.Helper()
	for name, entries := range map[string][]srtEntry{"src.srt": source, "before.srt": before} {
		var data strings.Builder
		for i, entry := range entries {
			fmt.Fprintf(&data, "%d\n%s --> %s\n%s\n\n", i+1, entry.Start, entry.End, entry.Text)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "baseline_audio"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// collectTranslationRealMetrics 用同一音色和原速重合成历史译文，避免旧音频变速污染对照。
func collectTranslationRealMetrics(t *testing.T, dir string, original, before []srtEntry, engine TTSClient) {
	t.Helper()
	after := readTranslationRealText(t, dir)
	if len(after) != len(original) {
		t.Fatal("译文条数改变")
	}
	metrics := make([]translationRealMetric, len(after))
	for i, entry := range after {
		if entry.Start != original[i].Start || entry.End != original[i].End {
			t.Fatal("原时间轴改变")
		}
		baseline := filepath.Join(dir, "baseline_audio", fmt.Sprintf("%d.wav", i))
		if err := engine.Synthesize(t.Context(), before[i].Text, baseline, "zh-CN-YunjianNeural"); err != nil {
			t.Fatal(err)
		}
		first, final := probeMediaDuration("ffmpeg", baseline), probeMediaDuration("ffmpeg", filepath.Join(dir, "audio_segs", fmt.Sprintf("%d.wav", i)))
		if first <= 0 || final <= 0 {
			t.Fatal("无法测量真实语音长度")
		}
		window := original[i].EndSec - original[i].StartSec
		metrics[i] = translationRealMetric{original[i].Index, original[i].Text, before[i].Text, entry.Text, window, first, final, first > window*1.2, final > window*1.2}
		t.Logf("原第%d条：窗口 %.2fs；旧译 %.2fs；新译 %.2fs；%s", original[i].Index, window, first, final, entry.Text)
	}
	data, err := json.Marshal(metrics, jsontext.WithIndent("  "))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "metrics.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// readTranslationRealText 使用真实配音文本，避免 SRT 的排版换行制造额外停顿。
func readTranslationRealText(t *testing.T, dir string) []srtEntry {
	t.Helper()
	entries := readSpeechSyncSRT(t, dir, "trans.srt")
	data, err := os.ReadFile(filepath.Join(dir, "trans.txt"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(entries) {
		t.Fatal("配音文本和字幕条数不同")
	}
	for i := range entries {
		entries[i].Text = strings.TrimSpace(lines[i])
	}
	return entries
}

// TestTranslationQualityRealMetrics 仅重测历史语音基线，保持已生成的新译文和音频不变。
func TestTranslationQualityRealMetrics(t *testing.T) {
	source, output := os.Getenv("VDUB_TRANSLATION_SOURCE"), os.Getenv("VDUB_TRANSLATION_OUTPUT")
	if source == "" || output == "" {
		t.Skip("需要真实字幕来源和已有输出目录")
	}
	original := readSpeechSyncSRT(t, source, "src.srt")
	before := readTranslationRealText(t, source)
	for _, bounds := range [][2]int{{223, 228}, {436, 442}, {36, 40}, {118, 122}, {384, 388}} {
		t.Run(fmt.Sprintf("%d-%d", bounds[0], bounds[1]), func(t *testing.T) {
			dir := filepath.Join(output, fmt.Sprintf("%d-%d", bounds[0], bounds[1]))
			collectTranslationRealMetrics(t, dir, original[bounds[0]-1:bounds[1]], before[bounds[0]-1:bounds[1]], tts.NewEdgeTTS("zh-CN-YunjianNeural"))
		})
	}
}

// TestTranslationQualityRealDuplicateReplay 把真实错误旧译送入检查入口，验证重复修复确实发生而非仅靠初译碰巧正确。
func TestTranslationQualityRealDuplicateReplay(t *testing.T) {
	config, source, output := os.Getenv("VDUB_TRANSLATION_CONFIG"), os.Getenv("VDUB_TRANSLATION_SOURCE"), os.Getenv("VDUB_TRANSLATION_OUTPUT")
	if config == "" || source == "" || output == "" {
		t.Skip("需要真实服务配置、字幕来源和新输出目录")
	}
	cfg := readSpeechSyncConfig(t, config)
	original := readSpeechSyncSRT(t, source, "src.srt")[435:442]
	before := readTranslationRealText(t, source)[435:442]
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := prepareTranslationRealSample(t, output, original, before)
	recorder := &speechSyncRecorder{dir: dir}
	engine := tts.NewEdgeTTS("zh-CN-YunjianNeural")
	core := NewCore(Config{FFmpegBin: "ffmpeg", TranslateChunkSize: cfg.Pipeline.TranslateChunkSize}, nil, llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model), engine, WithNotifier(recorder))
	review := translationReview{core: core, job: Job{TaskID: t.Name(), OutputDir: dir, Mode: ModeDub, TargetLang: "zh-CN", SourceLang: "en", Translator: "openai", TTSEngine: "edge", TTSVoice: "zh-CN-YunjianNeural", SpeechRate: 1}, entries: original}
	for i := range original {
		review.source = append(review.source, original[i].Text)
		review.translated = append(review.translated, before[i].Text)
	}
	runTranslationRealReplay(t, &review, recorder)
}

// runTranslationRealReplay 只验证真实重复修复并保存字幕，避免检查回放产生试配音。
func runTranslationRealReplay(t *testing.T, review *translationReview, recorder *speechSyncRecorder) {
	t.Helper()
	if !review.duplicateAt(5) {
		t.Fatal("历史样本未包含预期重复")
	}
	if err := review.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if recorder.err != nil {
		t.Fatal(recorder.err)
	}
	if review.duplicateAt(5) {
		t.Fatal("原第440和441条重复未消除")
	}
	if err := review.core.writeTranslationOutputs(review.entries, review.translated, review.job.OutputDir); err != nil {
		t.Fatal(err)
	}
	assertTranslationRealReplay(t, review.job.OutputDir)
}

// assertTranslationRealReplay 核查真实服务日志，保证重复组只触发一次质量重译。
func assertTranslationRealReplay(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "engines.log"))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(data), "质量重译第 5-6 条（仅一次）"); count != 1 {
		t.Fatalf("重复组重译次数 %d，期望 1", count)
	}
}
