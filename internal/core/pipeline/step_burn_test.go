package pipeline

import "testing"

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
