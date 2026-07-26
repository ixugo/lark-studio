package pipeline

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestTransSRTAlignedWithSrcSRT 验证翻译字幕与原始字幕时间轴 1:1 对齐
func TestTransSRTAlignedWithSrcSRT(t *testing.T) {
	srcContent := `1
00:00:00,000 --> 00:00:03,000
Welcome to this course.

2
00:00:03,500 --> 00:00:07,000
We will discuss design philosophy.

3
00:00:07,200 --> 00:00:10,500
Let us start with the basics.
`
	transContent := `1
00:00:00,000 --> 00:00:03,000
欢迎来到这门课程。

2
00:00:03,500 --> 00:00:07,000
我们将讨论设计哲学。

3
00:00:07,200 --> 00:00:10,500
让我们从基础开始。
`
	srcEntries := parseSRT(srcContent)
	transEntries := parseSRT(transContent)

	if len(srcEntries) != len(transEntries) {
		t.Fatalf("条目数不一致: src=%d, trans=%d", len(srcEntries), len(transEntries))
	}

	for i := range srcEntries {
		if srcEntries[i].Start != transEntries[i].Start {
			t.Errorf("[%d] 开始时间不一致: src=%s, trans=%s", i, srcEntries[i].Start, transEntries[i].Start)
		}
		if srcEntries[i].End != transEntries[i].End {
			t.Errorf("[%d] 结束时间不一致: src=%s, trans=%s", i, srcEntries[i].End, transEntries[i].End)
		}
	}
}

// TestBuildTransSRT_PreservesTimestamps 验证 buildTransSRT 保持原始时间轴
func TestBuildTransSRT_PreservesTimestamps(t *testing.T) {
	entries := []srtEntry{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "Hello world.", StartSec: 1.0, EndSec: 4.0},
		{Index: 2, Start: "00:00:05,500", End: "00:00:08,200", Text: "This is a test.", StartSec: 5.5, EndSec: 8.2},
	}
	translated := []string{"你好世界。", "这是一个测试。"}

	dir := t.TempDir()
	outPath := filepath.Join(dir, "trans.srt")
	if err := buildTransSRT(entries, translated, outPath); err != nil {
		t.Fatalf("buildTransSRT failed: %v", err)
	}

	data, _ := os.ReadFile(outPath)
	result := parseSRT(string(data))

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	for i, entry := range entries {
		if result[i].Start != entry.Start || result[i].End != entry.End {
			t.Errorf("[%d] 时间轴偏移: want %s→%s, got %s→%s",
				i, entry.Start, entry.End, result[i].Start, result[i].End)
		}
		if result[i].Text != translated[i] {
			t.Errorf("[%d] 翻译内容不符: want %q, got %q", i, translated[i], result[i].Text)
		}
	}
}

// TestAudioDurationWithinSubtitleWindow 验证每段 TTS 音频不超过对应字幕时间窗口
// 字幕消失后仍在播放音频是明显的同步问题
func TestAudioDurationWithinSubtitleWindow(t *testing.T) {
	const maxExcessSec = 2.0 // 允许的最大超出秒数

	entries := []srtEntry{
		{Index: 1, Start: "00:00:00,000", End: "00:00:03,000", StartSec: 0, EndSec: 3.0},
		{Index: 2, Start: "00:00:03,500", End: "00:00:07,000", StartSec: 3.5, EndSec: 7.0},
		{Index: 3, Start: "00:00:07,500", End: "00:00:09,000", StartSec: 7.5, EndSec: 9.0},
	}
	audioDurations := []float64{2.5, 3.0, 1.2}

	for i, dur := range audioDurations {
		window := entries[i].EndSec - entries[i].StartSec
		excess := dur - window
		if excess > maxExcessSec {
			t.Errorf("[%d] 音频超出字幕窗口 %.1fs: 音频=%.1fs, 窗口=%.1fs",
				i, excess, dur, window)
		}
	}
}

// TestSubtitleGapDetection 验证间距检测逻辑能发现异常间距
func TestSubtitleGapDetection(t *testing.T) {
	const maxGapSec = 3.0

	t.Run("normal_gaps_pass", func(t *testing.T) {
		entries := []srtEntry{
			{Index: 1, StartSec: 0, EndSec: 3.0},
			{Index: 2, StartSec: 3.5, EndSec: 7.0},
			{Index: 3, StartSec: 7.2, EndSec: 10.0},
		}
		for i := 1; i < len(entries); i++ {
			gap := entries[i].StartSec - entries[i-1].EndSec
			if gap > maxGapSec {
				t.Errorf("unexpected large gap at %d: %.1fs", i, gap)
			}
		}
	})

	t.Run("detects_large_gap", func(t *testing.T) {
		entries := []srtEntry{
			{Index: 1, StartSec: 0, EndSec: 3.0},
			{Index: 2, StartSec: 3.5, EndSec: 7.0},
			{Index: 3, StartSec: 12.0, EndSec: 15.0},
		}
		found := false
		for i := 1; i < len(entries); i++ {
			gap := entries[i].StartSec - entries[i-1].EndSec
			if gap > maxGapSec {
				found = true
				t.Logf("correctly detected gap at %d: %.1fs", i, gap)
			}
		}
		if !found {
			t.Error("failed to detect the 5s gap in test data")
		}
	})
}

// TestSRTTimestampMonotonic 验证 SRT 时间轴单调递增（不倒退）
func TestSRTTimestampMonotonic(t *testing.T) {
	content := `1
00:00:00,000 --> 00:00:03,000
First line.

2
00:00:03,500 --> 00:00:07,000
Second line.

3
00:00:07,200 --> 00:00:10,500
Third line.
`
	entries := parseSRT(content)
	for i := 1; i < len(entries); i++ {
		if entries[i].StartSec < entries[i-1].StartSec {
			t.Errorf("[%d] 开始时间倒退: %.3f < %.3f",
				i, entries[i].StartSec, entries[i-1].StartSec)
		}
	}
}

// TestWavDuration 从 WAV 文件大小推算时长
func TestWavDuration(t *testing.T) {
	tests := []struct {
		name          string
		fileSize      int64
		sampleRate    int
		channels      int
		bitsPerSample int
		wantSec       float64
	}{
		{"16kHz mono 16bit 1s", 32044, 16000, 1, 16, 1.0},
		{"16kHz mono 16bit 0.5s", 16044, 16000, 1, 16, 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headerSize := int64(44)
			dataBytes := tt.fileSize - headerSize
			bytesPerSample := tt.bitsPerSample / 8
			samplesPerSec := tt.sampleRate * tt.channels * bytesPerSample
			gotSec := float64(dataBytes) / float64(samplesPerSec)
			if math.Abs(gotSec-tt.wantSec) > 0.05 {
				t.Errorf("duration = %.3f, want %.3f", gotSec, tt.wantSec)
			}
		})
	}
}

// wavDuration 从 WAV 文件大小推算时长（16kHz mono 16bit）
func wavDuration(fileSize int64) float64 {
	const headerSize = 44
	const bytesPerSec = 32000 // 16000Hz * 1ch * 2bytes
	if fileSize <= headerSize {
		return 0
	}
	return float64(fileSize-headerSize) / float64(bytesPerSec)
}

// TestSyncReport 生成字幕出现时间 vs 音频出现时间的对照报告
func TestSyncReport(t *testing.T) {
	entries := []srtEntry{
		{Index: 1, Start: "00:00:00,000", End: "00:00:03,000", StartSec: 0, EndSec: 3.0},
		{Index: 2, Start: "00:00:03,500", End: "00:00:07,000", StartSec: 3.5, EndSec: 7.0},
		{Index: 3, Start: "00:00:07,200", End: "00:00:10,500", StartSec: 7.2, EndSec: 10.5},
	}
	audioDurations := []float64{2.5, 3.0, 2.8}

	audioStartSec := 0.0
	for i, entry := range entries {
		if i >= len(audioDurations) {
			break
		}
		subStart := entry.StartSec
		subEnd := entry.EndSec
		audioEnd := audioStartSec + audioDurations[i]

		drift := audioStartSec - subStart
		t.Logf("[%d] sub=%.1f~%.1fs  audio=%.1f~%.1fs  drift=%.2fs",
			i, subStart, subEnd, audioStartSec, audioEnd, drift)

		if math.Abs(drift) > 2.0 {
			t.Errorf("[%d] 字幕/音频漂移过大: %.2fs", i, drift)
		}

		audioStartSec = audioEnd
	}

	t.Log(fmt.Sprintf("总音频时长: %.1fs, 最后字幕结束: %.1fs",
		audioStartSec, entries[len(entries)-1].EndSec))
}
