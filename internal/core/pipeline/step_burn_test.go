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
