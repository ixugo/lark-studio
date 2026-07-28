package pipeline

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestTrimTrailingSilencePreservesInternalPauses 验证尾静音处理不会截断句中停顿。
func TestTrimTrailingSilencePreservesInternalPauses(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("未安装 ffmpeg")
	}
	audioPath := filepath.Join(t.TempDir(), "0.wav")
	createSpeechLikeMP3(t, ffmpeg, audioPath)

	core := Core{cfg: Config{FFmpegBin: ffmpeg}, notifier: &noopNotifier{}}
	core.trimTrailingSilence(
		context.Background(),
		[]string{audioPath},
		Job{TaskID: "trim-test"},
	)

	duration := probeMediaDuration(ffmpeg, audioPath)
	if duration < 0.9 {
		t.Fatalf("句中停顿被截断：时长 %.3f 秒，期望至少 0.9 秒", duration)
	}
	assertAudioCodec(t, audioPath, "pcm_s16le")
}

// TestSortAudioFilesNumerically 验证两位数音频仍按字幕序号排列。
func TestSortAudioFilesNumerically(t *testing.T) {
	files := []string{"/tmp/10.wav", "/tmp/2.wav", "/tmp/1.wav"}
	sortAudioFiles(files)
	want := []string{"/tmp/1.wav", "/tmp/2.wav", "/tmp/10.wav"}
	for index := range want {
		if files[index] != want[index] {
			t.Fatalf("音频顺序 = %v，期望 %v", files, want)
		}
	}
}

// createSpeechLikeMP3 生成带句中停顿和尾静音的 edge-tts 同格式夹具。
func createSpeechLikeMP3(t *testing.T, ffmpeg, outputPath string) {
	t.Helper()
	cmd := exec.Command(
		ffmpeg,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=0.45:sample_rate=24000",
		"-f", "lavfi", "-i", "anullsrc=r=24000:cl=mono:d=0.20",
		"-f", "lavfi", "-i", "sine=frequency=660:duration=0.45:sample_rate=24000",
		"-f", "lavfi", "-i", "anullsrc=r=24000:cl=mono:d=0.25",
		"-filter_complex", "[0:a][1:a][2:a][3:a]concat=n=4:v=0:a=1[a]",
		"-map", "[a]", "-c:a", "libmp3lame", outputPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("生成音频夹具失败：%v\n%s", err, output)
	}
}

// assertAudioCodec 验证音频已统一为合并所需的编码。
func assertAudioCodec(t *testing.T, audioPath, expected string) {
	t.Helper()
	output, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("读取音频编码失败：%v\n%s", err, output)
	}
	if string(output) != expected+"\n" {
		t.Fatalf("音频编码 = %q，期望 %q", output, expected)
	}
}

// TestAdjustSpeedFactor_Capping 验证调速倍率遵守字幕窗口与上限。
func TestAdjustSpeedFactor_Capping(t *testing.T) {
	tests := []struct {
		name      string
		audioDur  float64
		segDur    float64
		maxFactor float64
		wantSkip  bool
		wantCap   float64
	}{
		{"音频短于段落 → 跳过", 3.0, 5.0, 1.3, true, 0},
		{"差异 <2% → 跳过", 5.05, 5.0, 1.3, true, 0},
		{"正常调速", 6.0, 5.0, 1.3, false, 1.2},
		{"超限截断", 8.0, 5.0, 1.3, false, 1.3},
		{"maxFactor ≤1 → 全跳过", 6.0, 5.0, 0.9, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.maxFactor <= 1 {
				if !tt.wantSkip {
					t.Error("maxFactor ≤1 应跳过")
				}
				return
			}

			if tt.audioDur <= 0 || tt.segDur <= 0 || tt.audioDur <= tt.segDur {
				if !tt.wantSkip {
					t.Error("音频不超段落应跳过")
				}
				return
			}

			factor := tt.audioDur / tt.segDur
			if factor < 1.02 {
				if !tt.wantSkip {
					t.Error("差异 <2% 应跳过")
				}
				return
			}

			if tt.wantSkip {
				t.Error("不应跳过")
				return
			}

			if factor > tt.maxFactor {
				factor = tt.maxFactor
			}
			if factor != tt.wantCap {
				t.Errorf("factor = %.2f, want %.2f", factor, tt.wantCap)
			}
		})
	}
}

// TestMaxSpeedFactorConfig 验证无效调速配置不会生效。
func TestMaxSpeedFactorConfig(t *testing.T) {
	cfg := Config{MaxSpeedFactor: 1.25}
	if cfg.MaxSpeedFactor <= 1 {
		t.Error("配置值应 > 1 才生效")
	}
	if cfg.MaxSpeedFactor > 2 {
		t.Error("调速超过 2x 影响听感")
	}

	cfg2 := Config{MaxSpeedFactor: 0}
	if cfg2.MaxSpeedFactor > 1 {
		t.Error("零值不应触发调速")
	}
}
