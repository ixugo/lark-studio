package pipeline

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// fixtureDurationTTS 用静音 WAV 隔离时长选择规则，不把夹具当作真实语音效果。
type fixtureDurationTTS struct {
	ffmpeg  string
	seconds map[string]float64
	calls   []string
}

// Synthesize 按文本映射生成指定时长，使字数与声音时长可以分别控制。
func (m *fixtureDurationTTS) Synthesize(ctx context.Context, text, outputPath, voice string) error {
	m.calls = append(m.calls, text)
	duration, ok := m.seconds[text]
	if !ok {
		return fmt.Errorf("未定义静音夹具：%q", text)
	}
	output, err := exec.CommandContext(ctx, m.ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "anullsrc=r=16000:cl=mono", "-t", fmt.Sprintf("%.3f", duration), "-c:a", "pcm_s16le", "-y", outputPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("生成静音 WAV 失败: %w: %s", err, output)
	}
	return nil
}

// TestTranslationAudioQualityChoosesMeasuredDuration 防止用字符数替代实际配音时长做选择。
func TestTranslationAudioQualityChoosesMeasuredDuration(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("真实媒体夹具需要本机 ffmpeg")
	}
	tests := []struct {
		name, original, candidate, want string
		first, second, wantSeconds      float64
	}{
		{"字更少但音频更长保留首版", "这句文字比较长", "短句", "这句文字比较长", 2, 3, 2},
		{"字更多但音频更短采用重译", "短句", "这句文字反而比较长", "这句文字反而比较长", 2, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			review, llm := newQualityReview([]string{"source"}, []string{tt.original}, []string{tt.candidate})
			tts := &fixtureDurationTTS{ffmpeg: ffmpeg, seconds: map[string]float64{tt.original: tt.first, tt.candidate: tt.second}}
			review.core.tts, review.core.cfg.FFmpegBin = tts, ffmpeg
			review.job.Mode, review.job.OutputDir = ModeDub, t.TempDir()
			review.entries = []srtEntry{{EndSec: 1}}
			if err := review.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			if review.translated[0] != tt.want {
				t.Fatalf("所选译文 %q，期望 %q", review.translated[0], tt.want)
			}
			output := filepath.Join(review.job.OutputDir, "audio_segs", "0.wav")
			assertQualityAudio(t, review, tt.want, tt.wantSeconds)
			if len(llm.calls) != 1 || len(tts.calls) != 2 {
				t.Fatalf("调用超出单次预算：翻译=%d 配音=%d", len(llm.calls), len(tts.calls))
			}
			cfg := ttsWorkerConfig{ctx: t.Context(), job: review.job, audioDir: filepath.Dir(output), total: 1}
			if err := review.core.synthesizePair(cfg, ttsPair{index: 0, text: tt.want}); err != nil {
				t.Fatal(err)
			}
			if len(tts.calls) != 2 {
				t.Fatal("最终配音应复用已经测量且指纹相符的音频")
			}
		})
	}
}

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

// assertQualityAudio 同时检查媒体时长和文本指纹，避免只验证内存中的译文。
func assertQualityAudio(t *testing.T, review *translationReview, want string, seconds float64) {
	t.Helper()
	output := filepath.Join(review.job.OutputDir, "audio_segs", "0.wav")
	if got := probeMediaDuration(review.core.cfg.FFmpegBin, output); math.Abs(got-seconds) > 0.01 {
		t.Fatalf("选中音频时长 %.3f，期望 %.3f", got, seconds)
	}
	if !audioSegmentMatchesText(output, want) {
		t.Fatal("正式配音指纹与所选译文不一致")
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
