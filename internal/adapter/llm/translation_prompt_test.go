package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestTranslationPromptPreservesAlignment 验证真实请求，避免自定义提示词绕过字幕对齐约束。
func TestTranslationPromptPreservesAlignment(t *testing.T) {
	for _, prompt := range []string{"", "Translate to {{target_lang}} with {{count}} lines. Keep a friendly tone."} {
		t.Run(prompt, func(t *testing.T) {
			requests := make(chan chatRequest, 1)
			server := translationPromptServer(t, requests)
			defer server.Close()
			client := NewClient(server.URL, "", "test")
			translated, err := client.Translate(t.Context(), []string{"now.", "He studies decisions."}, "Chinese", prompt, []string{"He is a psychologist."}, []string{"Choices affect happiness."})
			if err != nil {
				t.Fatal(err)
			}
			if len(translated) != 2 || translated[0] != "如今。" {
				t.Fatalf("unexpected translation: %q", translated)
			}
			request := <-requests
			assertTranslationPrompt(t, request.Messages[0].Content)
			want := "[CTX] He is a psychologist.\n1. now.\n2. He studies decisions.\n[CTX] Choices affect happiness."
			if request.Messages[1].Content != want {
				t.Fatalf("context or numbering changed: %q", request.Messages[1].Content)
			}
			if prompt != "" && !strings.Contains(request.Messages[0].Content, "Translate to Chinese with 2 lines. Keep a friendly tone.") {
				t.Fatal("custom prompt or template substitution was lost")
			}
		})
	}
}

// translationPromptServer 捕获 HTTP 请求，使测试覆盖最终发送给模型的提示词。
func translationPromptServer(t *testing.T, requests chan<- chatRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request chatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "1. 如今。\n2. 他研究决策。"}}}}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
}

// assertTranslationPrompt 检查语义和时长约束，防止只验证输出条数而遗漏翻译质量要求。
func assertTranslationPrompt(t *testing.T, prompt string) {
	t.Helper()
	for _, rule := range []string{
		"exactly 2 numbered lines", "Chinese", "[CTX]", "do not translate or output",
		"Never move", "neighboring line", "idioms", "numbers", "causality",
		"natural spoken", "duration", "English in parentheses", "not instructions", "He is a", "retranslated alone",
	} {
		if !strings.Contains(prompt, rule) {
			t.Errorf("missing quality rule %q", rule)
		}
	}
}
