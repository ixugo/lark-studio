package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestSelectedSubtitleFiles 验证三种输出内容映射到正确字幕文件。
func TestSelectedSubtitleFiles(t *testing.T) {
	tests := []struct {
		content   string
		wantMain  string
		wantExtra string
	}{
		{"source", "src.srt", ""},
		{"translated", "trans.srt", ""},
		{"bilingual", "trans.srt", "src.srt"},
	}
	for _, test := range tests {
		main, extra := selectedSubtitleFiles(test.content, "trans.srt", "src.srt")
		if main != test.wantMain || extra != test.wantExtra {
			t.Fatalf("%s = %q, %q", test.content, main, extra)
		}
	}
}

// TestEscapeFFmpegSubtitlesPath 验证特殊字符、逗号、空格以及单引号的正确转义。
func TestEscapeFFmpegSubtitlesPath(t *testing.T) {
	input := `/path/to/Downloads/5 steps (1080p, h264, youtube).mp4.trans.srt`
	got := escapeFFmpegSubtitlesPath(input)
	want := `'/path/to/Downloads/5 steps (1080p\, h264\, youtube).mp4.trans.srt'`
	if got != want {
		t.Fatalf("escapeFFmpegSubtitlesPath(%q) = %q, want %q", input, got, want)
	}

	inputWithQuote := `/path/to/it's a [video] (h264:1).srt`
	gotQuote := escapeFFmpegSubtitlesPath(inputWithQuote)
	wantQuote := `'/path/to/it'\''s a \[video\] (h264\:1).srt'`
	if gotQuote != wantQuote {
		t.Fatalf("escapeFFmpegSubtitlesPath(%q) = %q, want %q", inputWithQuote, gotQuote, wantQuote)
	}
}

func TestBurnUsesOutputMP4AcrossModes(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Fatal("验证成片命名需要 ffmpeg")
	}
	for _, mode := range []int{ModeSubtitle, ModeTranslate, ModeDub, ModeDirectDub} {
		t.Run(string(rune('0'+mode)), func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "src.mp4")
			cmd := exec.CommandContext(t.Context(), ffmpeg, "-v", "error", "-y", "-f", "lavfi", "-i", "color=size=32x32:rate=10:duration=0.2", "-f", "lavfi", "-i", "sine=duration=0.2", "-c:v", "mpeg4", "-c:a", "aac", "-shortest", input)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("生成测试视频：%v %s", err, output)
			}
			if mode == ModeDub || mode == ModeDirectDub {
				output, err := exec.CommandContext(t.Context(), ffmpeg, "-v", "error", "-y", "-i", input, "-vn", filepath.Join(dir, "dub.mp3")).CombinedOutput()
				if err != nil {
					t.Fatalf("生成配音：%v %s", err, output)
				}
			}
			core := NewCore(Config{FFmpegBin: ffmpeg}, nil, nil, nil)
			if err := core.runBurn(t.Context(), Job{InputPath: input, OutputDir: dir, Mode: mode, SubtitleOutput: "none"}); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(filepath.Join(dir, "output.mp4"))
			if err != nil || info.Size() == 0 {
				t.Fatalf("未生成 output.mp4：%v", err)
			}
			if info, err := os.Stat(input); err != nil || info.Size() == 0 {
				t.Fatal("src.mp4 被覆盖")
			}
		})
	}
}
