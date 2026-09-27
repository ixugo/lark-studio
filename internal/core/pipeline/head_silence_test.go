package pipeline

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestConcatWithTimeline_HeadSilence 验证当首句字幕非 0 秒时，生成的最终音频必须在前段保持纯静音
func TestConcatWithTimeline_HeadSilence(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("缺少 ffmpeg，跳过本测试")
	}

	dir := t.TempDir()
	c := &Core{cfg: Config{FFmpegBin: ffmpeg}}

	// 1. 生成一段 1 秒的测试语音音频 seg0.wav (24000Hz, 1s 440Hz 蜂鸣)
	seg0 := filepath.Join(dir, "0.wav")
	cmd := exec.Command(ffmpeg, "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1.0", "-c:a", "pcm_s16le", "-ar", "24000", "-ac", "1", seg0)
	if err := cmd.Run(); err != nil {
		t.Fatalf("生成测试音频失败: %v", err)
	}

	// 2. 构造字幕：首句在 3.0 秒才开始
	entries := []srtEntry{
		{
			Index:    1,
			Start:    "00:00:03,000",
			End:      "00:00:04,000",
			StartSec: 3.0,
			EndSec:   4.0,
			Text:     "First line",
		},
	}

	outDub := filepath.Join(dir, "dub.mp3")
	if err := c.concatWithTimeline(context.Background(), []string{seg0}, entries, outDub); err != nil {
		t.Fatalf("concatWithTimeline failed: %v", err)
	}

	// 3. 验证输出 dub.mp3 总时长必须 >= 3.8 秒 (3s 静音 + 1s 音频)
	dur := probeMediaDuration(ffmpeg, outDub)
	if dur < 3.8 {
		t.Fatalf("预期音频时长至少 3.8 秒，实际仅: %.2f 秒，说明片头静音未生效或被跳过！", dur)
	}
}
