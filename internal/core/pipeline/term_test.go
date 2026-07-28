package pipeline

import (
	"context"
	"testing"
)

func TestMatchTermMappings(t *testing.T) {
	tests := []struct {
		name      string
		mappings  []TermMapping
		sentences []string
		wantLen   int
		wantTexts []string
	}{
		{
			name: "exact match case-insensitive",
			mappings: []TermMapping{
				{Text: "Golang", Translation: "Golang"},
				{Text: "Java", Translation: "Java"},
				{Text: "OK", Translation: "OK"},
			},
			sentences: []string{"We use golang for this project", "Is everything ok?"},
			wantLen:   2,
			wantTexts: []string{"Golang", "OK"},
		},
		{
			name: "custom translation",
			mappings: []TermMapping{
				{Text: "AI", Translation: "人工智能"},
				{Text: "ML", Translation: "机器学习"},
			},
			sentences: []string{"AI and ML are important"},
			wantLen:   2,
			wantTexts: []string{"AI", "ML"},
		},
		{
			name:      "no match",
			mappings:  []TermMapping{{Text: "Rust", Translation: "Rust"}},
			sentences: []string{"We use Golang here"},
			wantLen:   0,
		},
		{
			name:      "empty mappings",
			mappings:  nil,
			sentences: []string{"Hello world"},
			wantLen:   0,
		},
		{
			name: "duplicate terms deduplicated",
			mappings: []TermMapping{
				{Text: "Go", Translation: "Go"},
				{Text: "go", Translation: "go"},
			},
			sentences: []string{"Let's go"},
			wantLen:   1,
		},
		{
			name: "multi-word term",
			mappings: []TermMapping{
				{Text: "Visual Studio Code", Translation: "VSCode"},
			},
			sentences: []string{"Open Visual Studio Code to edit"},
			wantLen:   1,
			wantTexts: []string{"Visual Studio Code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchTermMappings(tt.mappings, tt.sentences)
			if len(got) != tt.wantLen {
				t.Fatalf("matchTermMappings() returned %d, want %d", len(got), tt.wantLen)
			}
			for i, text := range tt.wantTexts {
				if i >= len(got) {
					break
				}
				if got[i].Text != text {
					t.Errorf("[%d] Text = %q, want %q", i, got[i].Text, text)
				}
			}
		})
	}
}

// mockTermLister 测试用 mock，实现 TermLister 接口
type mockTermLister struct {
	mappings []TermMapping
	err      error
}

func (m *mockTermLister) ListMappings(_ context.Context) ([]TermMapping, error) {
	return m.mappings, m.err
}

func TestInjectTermsIntoPrompt(t *testing.T) {
	ctx := context.Background()
	core := &Core{}

	t.Run("nil termLister returns base prompt unchanged", func(t *testing.T) {
		got := core.injectTermsIntoPrompt(ctx, "base prompt", []string{"hello Golang"})
		if got != "base prompt" {
			t.Fatalf("got %q, want %q", got, "base prompt")
		}
	})

	t.Run("no matching terms returns base prompt", func(t *testing.T) {
		core.termLister = &mockTermLister{mappings: []TermMapping{{Text: "Rust", Translation: "Rust"}}}
		got := core.injectTermsIntoPrompt(ctx, "base prompt", []string{"hello world"})
		if got != "base prompt" {
			t.Fatalf("got %q, want %q", got, "base prompt")
		}
	})

	t.Run("keep-as-is terms", func(t *testing.T) {
		core.termLister = &mockTermLister{mappings: []TermMapping{
			{Text: "Golang", Translation: "Golang"},
		}}
		got := core.injectTermsIntoPrompt(ctx, "base", []string{"Use golang"})
		if got == "base" {
			t.Fatal("prompt should be augmented")
		}
		if !contains(got, "keep as-is") {
			t.Fatalf("expected 'keep as-is' in prompt, got: %s", got)
		}
	})

	t.Run("custom translation terms", func(t *testing.T) {
		core.termLister = &mockTermLister{mappings: []TermMapping{
			{Text: "AI", Translation: "人工智能"},
		}}
		got := core.injectTermsIntoPrompt(ctx, "", []string{"AI is great"})
		if !contains(got, `"人工智能"`) {
			t.Fatalf("expected '人工智能' in prompt, got: %s", got)
		}
		if !contains(got, "Translate") || !contains(got, "{{target_lang}}") {
			t.Fatalf("术语提示词丢失翻译指令: %s", got)
		}
	})
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
