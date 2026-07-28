package llm

import "testing"

// TestRequireTranslationCount 验证翻译缺行时返回错误而非补入英文原文。
func TestRequireTranslationCount(t *testing.T) {
	_, err := requireTranslationCount([]string{"第一句"}, 2)
	if err == nil {
		t.Fatal("翻译缺行时必须失败")
	}

	got, err := requireTranslationCount([]string{"第一句", "第二句"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("译文数量 = %d，期望 2", len(got))
	}
}
