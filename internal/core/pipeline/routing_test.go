package pipeline

import (
	"context"
	"fmt"
	"testing"
)

// TestBingTranslationAvoidsLLMPreparation 验证必应任务不调用语义分句、术语提示词和普通 LLM 翻译。
func TestBingTranslationAvoidsLLMPreparation(t *testing.T) {
	llm := newRoutingTestLLM()
	terms := &countingTermLister{}
	core := NewCore(
		Config{TranslateChunkSize: 1},
		nil,
		llm,
		nil,
		WithTermLister(terms),
	)

	translated, err := core.translateAllChunks(
		context.Background(),
		Job{Translator: "bing", TargetLang: "zh-CN"},
		[]string{"hello", "world"},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprint(translated), "[bing:hello bing:world]"; got != want {
		t.Fatalf("译文 = %s，期望 %s", got, want)
	}
	if llm.standardCalls != 0 || llm.splitCalls != 0 || terms.calls != 0 {
		t.Fatalf("必应不应触发 LLM 预处理：translate=%d split=%d terms=%d", llm.standardCalls, llm.splitCalls, terms.calls)
	}
	if llm.routedCalls != 2 || llm.contextCalls != 0 || llm.promptCalls != 0 {
		t.Fatalf("必应路由参数错误：routed=%d context=%d prompt=%d", llm.routedCalls, llm.contextCalls, llm.promptCalls)
	}
}

// routingTestLLM 记录流水线对翻译适配器的调用路径。
type routingTestLLM struct {
	standardCalls int
	splitCalls    int
	routedCalls   int
	contextCalls  int
	promptCalls   int
}

// newRoutingTestLLM 创建独立的调用计数器。
func newRoutingTestLLM() *routingTestLLM {
	return &routingTestLLM{}
}

// SplitSentences 记录错误的必应语义分句调用。
func (m *routingTestLLM) SplitSentences(context.Context, string, string) ([]string, error) {
	m.splitCalls++
	return nil, nil
}

// Translate 记录错误的普通 LLM 翻译调用。
func (m *routingTestLLM) Translate(context.Context, []string, string, string, []string, []string) ([]string, error) {
	m.standardCalls++
	return nil, nil
}

// TranslateWithProvider 记录必应专用适配器调用并生成确定性译文。
func (m *routingTestLLM) TranslateWithProvider(
	_ context.Context,
	sentences []string,
	_ string,
	prompt string,
	before []string,
	after []string,
	provider string,
) ([]string, error) {
	m.routedCalls++
	if len(before) != 0 || len(after) != 0 {
		m.contextCalls++
	}
	if prompt != "" {
		m.promptCalls++
	}
	result := make([]string, len(sentences))
	for i, sentence := range sentences {
		result[i] = provider + ":" + sentence
	}
	return result, nil
}

// countingTermLister 记录术语表读取次数。
type countingTermLister struct {
	calls int
}

// ListMappings 仅返回空术语表，供断言调用次数使用。
func (m *countingTermLister) ListMappings(context.Context) ([]TermMapping, error) {
	m.calls++
	return nil, nil
}
