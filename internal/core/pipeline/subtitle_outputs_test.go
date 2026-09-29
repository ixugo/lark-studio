package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSubtitleArtifactsAreIndependent 固定保留原文，翻译后另存译文字幕。
func TestSubtitleArtifactsAreIndependent(t *testing.T) {
	for _, content := range []string{"source", "translated", "bilingual"} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			source := "1\n00:00:00,000 --> 00:00:02,000\nHello world\n\n"
			input := filepath.Join(dir, "input.mp4")
			if err := os.WriteFile(filepath.Join(dir, "input.srt"), []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			llm := &qualityLLM{replies: [][]string{{"你好世界"}}}
			core := NewCore(Config{}, nil, llm, nil)
			job := Job{InputPath: input, OutputDir: dir, Mode: ModeTranslate, Translator: "openai", TargetLang: "zh-CN", OutputContent: content}
			if err := core.runWhisper(t.Context(), job); err != nil {
				t.Fatal(err)
			}
			if err := core.runTranslate(t.Context(), job); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "src.srt"))
			if err != nil || string(got) != source {
				t.Fatalf("原文字幕丢失或被译文覆盖：%q %v", got, err)
			}
			translated, err := os.ReadFile(filepath.Join(dir, "trans.srt"))
			if err != nil || !strings.Contains(string(translated), "你好世界") || !strings.Contains(string(translated), "00:00:00,000 --> 00:00:02,000") {
				t.Fatalf("翻译字幕或时间轴缺失：%q %v", translated, err)
			}
		})
	}
}
