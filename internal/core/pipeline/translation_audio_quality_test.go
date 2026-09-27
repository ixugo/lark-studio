package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestTranslationQualityFailedReviewDoesNotStream 防止不合格译文在失败返回前进入配音。
func TestTranslationQualityFailedReviewDoesNotStream(t *testing.T) {
	llm := &qualityLLM{replies: [][]string{{"重复"}, {"重复"}, {"仍重复", "仍重复"}}}
	core := NewCore(Config{SemanticSplitReady: true, TranslateChunkSize: 1}, nil, llm, nil)
	stream := make(chan ttsPair, 4)
	result, err := core.translateAllChunks(t.Context(), Job{Translator: "openai", TargetLang: "zh-CN"}, []string{"first", "second"}, stream)
	if err == nil {
		t.Fatal("重复未消除必须失败")
	}
	if len(stream) != 0 || len(result) != 0 {
		t.Fatalf("失败却输出了译文：stream=%d result=%v", len(stream), result)
	}
	if len(llm.calls) != 3 {
		t.Fatalf("失败后不能继续重译：%d", len(llm.calls))
	}
}

// TestTranslationQualityCancelledDoesNotStream 防止已取消任务继续发起翻译或发送配音。
func TestTranslationQualityCancelledDoesNotStream(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	llm := &qualityLLM{}
	core := NewCore(Config{SemanticSplitReady: true}, nil, llm, nil)
	stream := make(chan ttsPair, 2)
	result, err := core.translateAllChunks(ctx, Job{Translator: "openai", TargetLang: "zh-CN"}, []string{"source"}, stream)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("应返回取消错误，实际 %v", err)
	}
	if len(stream) != 0 || len(result) != 0 || len(llm.calls) != 0 {
		t.Fatal("已取消任务不得继续翻译或流出字幕")
	}
}

// TestTranslationQualityFailureStopsOuterRetry 确保质量重试预算不会被整步重试清零。
func TestTranslationQualityFailureStopsOuterRetry(t *testing.T) {
	llm := &qualityLLM{replies: [][]string{{"重复", "重复"}, {"仍重复", "仍重复"}}}
	core := NewCore(Config{SemanticSplitReady: true}, nil, llm, nil)
	job := Job{Translator: "openai", TargetLang: "zh-CN"}
	result, qualityErr := core.translateAllChunks(t.Context(), job, []string{"first", "second"}, nil)
	if qualityErr == nil || len(result) != 0 {
		t.Fatal("构造质量失败未成功")
	}
	if _, ok := errors.AsType[*translationQualityError](qualityErr); !ok {
		t.Fatalf("质量失败必须保留不可重试类型：%T %v", qualityErr, qualityErr)
	}
	calls := 0
	err := core.runStepWithRetry(t.Context(), job, StepTranslate, func(context.Context, Job) error {
		calls++
		if calls > 1 {
			t.Fatal("外层重试突破了单次质量重译预算")
		}
		return fmt.Errorf("保留调用链: %w", qualityErr)
	})
	if !errors.Is(err, qualityErr) || calls != 1 {
		t.Fatalf("质量失败应立即返回：calls=%d err=%v", calls, err)
	}
}

// TestTranslationQualityOutputFailureStopsOuterRetry 避免质量检查完成后因落盘失败重复调用翻译服务。
func TestTranslationQualityOutputFailureStopsOuterRetry(t *testing.T) {
	outputDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(outputDir, "trans.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "1\n00:00:00,000 --> 00:00:05,000\nhello\n\n"
	if err := os.WriteFile(filepath.Join(outputDir, "src.srt"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	llm := &qualityLLM{replies: [][]string{{"你好"}}}
	core := NewCore(Config{SemanticSplitReady: true}, nil, llm, nil)
	job := Job{Translator: "openai", TargetLang: "zh-CN", Mode: ModeTranslate, OutputDir: outputDir}
	calls := 0
	err := core.runStepWithRetry(t.Context(), job, StepTranslate, func(ctx context.Context, current Job) error {
		calls++
		if calls > 1 {
			t.Fatal("字幕落盘失败导致整步重跑，重置了质量重译预算")
		}
		return core.runTranslate(ctx, current)
	})
	if err == nil {
		t.Fatal("trans.txt 是目录，写入必须失败")
	}
	if _, ok := errors.AsType[*translationQualityError](err); !ok {
		t.Fatalf("质量检查后的落盘错误必须停止整步重试：%T %v", err, err)
	}
	if calls != 1 || len(llm.calls) != 1 {
		t.Fatalf("落盘失败不能再次调用翻译：step=%d translation=%d", calls, len(llm.calls))
	}
}

// qualityRecordingTTS 只记录配音输入，使测试能精确识别检查阶段的额外调用。
type qualityRecordingTTS struct {
	calls []string
}

// Synthesize 写入可复用的测试文件，不把调用次数验证伪装成音质验证。
func (m *qualityRecordingTTS) Synthesize(ctx context.Context, text, outputPath, voice string) error {
	m.calls = append(m.calls, text)
	return os.WriteFile(outputPath, []byte("test audio"), 0o644)
}

// TestTranslationReviewDoesNotRetryLongText 确保短时间窗不会触发第二次翻译或预先试配音。
func TestTranslationReviewDoesNotRetryLongText(t *testing.T) {
	for _, mode := range []int{ModeTranslate, ModeDub} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			text := "这是一条明显放不进短时间窗的完整译文，保留原来的事实。"
			review, llm := newQualityReview([]string{"a long source sentence"}, []string{text})
			tts := new(qualityRecordingTTS)
			review.core.tts = tts
			review.job.Mode, review.job.OutputDir = mode, t.TempDir()
			review.entries = []srtEntry{{EndSec: 0.1}}
			if err := review.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			if len(llm.calls) != 0 || len(tts.calls) != 0 || review.translated[0] != text {
				t.Fatalf("超长检查产生额外调用或改稿：翻译=%d 配音=%d 译文=%q", len(llm.calls), len(tts.calls), review.translated)
			}
		})
	}
}

// TestTranslationPipelineOnlySynthesizesFinalText 保证重复修正后只配最终稿，不为长句生成候选语音。
func TestTranslationPipelineOnlySynthesizesFinalText(t *testing.T) {
	dir := t.TempDir()
	source := "1\n00:00:00,000 --> 00:00:00,100\nfirst\n\n2\n00:00:00,100 --> 00:00:00,200\nsecond\n\n"
	if err := os.WriteFile(filepath.Join(dir, "src.srt"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	final := []string{"第一条完整而且很长的正确译文。", "第二条同样很长但是内容不同的译文。"}
	llm := &qualityLLM{replies: [][]string{{"重复", "重复"}, final}}
	tts := new(qualityRecordingTTS)
	core := NewCore(Config{SemanticSplitReady: true, TTSWorkers: 1}, nil, llm, tts)
	job := Job{Translator: "openai", TargetLang: "zh-CN", Mode: ModeDub, OutputDir: dir}
	if err := core.runTranslate(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 2 || len(tts.calls) != 2 {
		t.Fatalf("只应首译一次、纠重复一次，每条配音一次：翻译=%d 配音=%d", len(llm.calls), len(tts.calls))
	}
	assertSliceEqual(t, "最终配音", final, tts.calls)
	for i, text := range final {
		if !audioSegmentMatchesText(filepath.Join(dir, "audio_segs", fmt.Sprintf("%d.wav", i)), text) {
			t.Fatal("最终配音与译文指纹不一致")
		}
	}
}
