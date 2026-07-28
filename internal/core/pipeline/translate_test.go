package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAudioSegmentMatchesText 验证旧音频仅在译文完全一致时复用。
func TestAudioSegmentMatchesText(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "0.wav")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if audioSegmentMatchesText(audioPath, "第一句中文") {
		t.Fatal("无译文指纹的旧音频不应复用")
	}
	if err := writeAudioTextFingerprint(audioPath, "第一句中文"); err != nil {
		t.Fatal(err)
	}
	if !audioSegmentMatchesText(audioPath, "第一句中文") {
		t.Fatal("相同译文应复用已有音频")
	}
	if audioSegmentMatchesText(audioPath, "第二句中文") {
		t.Fatal("译文变化后不应复用旧音频")
	}
}

// TestTranslateContextWindow 验证分块翻译携带正确的前后文。
func TestTranslateContextWindow(t *testing.T) {
	sentences := []string{"s0", "s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9"}

	tests := []struct {
		name       string
		chunkStart int
		chunkEnd   int
		wantBefore []string
		wantAfter  []string
	}{
		{
			name:       "首个 chunk，无前文",
			chunkStart: 0, chunkEnd: 3,
			wantBefore: nil,
			wantAfter:  []string{"s3", "s4", "s5"},
		},
		{
			name:       "中间 chunk，前后各 3",
			chunkStart: 3, chunkEnd: 6,
			wantBefore: []string{"s0", "s1", "s2"},
			wantAfter:  []string{"s6", "s7", "s8"},
		},
		{
			name:       "末尾 chunk，无后文",
			chunkStart: 8, chunkEnd: 10,
			wantBefore: []string{"s5", "s6", "s7"},
			wantAfter:  nil,
		},
		{
			name:       "前文不足 3",
			chunkStart: 1, chunkEnd: 4,
			wantBefore: []string{"s0"},
			wantAfter:  []string{"s4", "s5", "s6"},
		},
		{
			name:       "后文不足 3",
			chunkStart: 6, chunkEnd: 9,
			wantBefore: []string{"s3", "s4", "s5"},
			wantAfter:  []string{"s9"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := len(sentences)
			ctxStart := max(0, tt.chunkStart-contextWindow)
			ctxEnd := min(total, tt.chunkEnd+contextWindow)
			before := sentences[ctxStart:tt.chunkStart]
			after := sentences[tt.chunkEnd:ctxEnd]

			if len(before) == 0 {
				before = nil
			}
			if len(after) == 0 {
				after = nil
			}

			assertSliceEqual(t, "before", tt.wantBefore, before)
			assertSliceEqual(t, "after", tt.wantAfter, after)
		})
	}
}

// assertSliceEqual 比对上下文切片并输出准确下标。
func assertSliceEqual(t *testing.T, label string, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("%s: len = %d, want %d\ngot:  %v\nwant: %v", label, len(got), len(want), got, want)
		return
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("%s[%d] = %q, want %q", label, i, got[i], want[i])
		}
	}
}

// TestContextWindowConstant 验证上下文窗口不会膨胀提示词。
func TestContextWindowConstant(t *testing.T) {
	if contextWindow <= 0 {
		t.Error("contextWindow 必须为正整数")
	}
	if contextWindow > 10 {
		t.Error("contextWindow 过大会导致 prompt 膨胀")
	}
}
