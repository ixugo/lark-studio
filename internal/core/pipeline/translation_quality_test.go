package pipeline

import (
	"context"
	"fmt"
	"slices"
	"testing"
)

// qualityCall 保存调用参数，便于验证重译携带了完整上下文。
type qualityCall struct {
	sentences, before, after []string
	prompt                   string
}

// qualityLLM 用固定回复检验检查器，避免依赖在线模型的随机结果。
type qualityLLM struct {
	replies [][]string
	calls   []qualityCall
}

// SplitSentences 禁止质量检查意外改动原始字幕分句。
func (m *qualityLLM) SplitSentences(context.Context, string, string) ([]string, error) {
	return nil, fmt.Errorf("质量检查不应重新分句")
}

// Translate 记录请求并按顺序回复，使额外重试直接暴露为错误。
func (m *qualityLLM) Translate(ctx context.Context, sentences []string, lang, prompt string, before, after []string) ([]string, error) {
	m.calls = append(m.calls, qualityCall{slices.Clone(sentences), slices.Clone(before), slices.Clone(after), prompt})
	if len(m.calls) > len(m.replies) {
		return nil, fmt.Errorf("意外的额外翻译调用")
	}
	return slices.Clone(m.replies[len(m.calls)-1]), nil
}

// newQualityReview 为每个用例隔离字幕和重试状态。
func newQualityReview(source, translated []string, replies ...[]string) (*translationReview, *qualityLLM) {
	llm := &qualityLLM{replies: replies}
	review := &translationReview{
		core:   NewCore(Config{SemanticSplitReady: true}, nil, llm, nil),
		job:    Job{Translator: "openai", TargetLang: "zh-CN"},
		source: slices.Clone(source), translated: slices.Clone(translated), retried: make([]bool, len(source)),
	}
	return review, llm
}

// TestTranslationQualityDuplicateRegression 防止 440 条提前搬用 441 条译文的问题流入配音。
func TestTranslationQualityDuplicateRegression(t *testing.T) {
	source := []string{"before one", "before two", "before three", "now.", "And he's been studying decision making and how do you make yourself happy?", "after one", "after two", "after three"}
	translated := []string{"前一", "前二", "前三", "他一直研究如何做决定并让自己快乐？", "他一直研究如何做决定并让自己快乐？", "后一", "后二", "后三"}
	review, llm := newQualityReview(source, translated, []string{"至今。", "他一直研究如何做决定、让自己快乐。"})
	if err := review.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 1 {
		t.Fatalf("调用 %d 次，期望一次", len(llm.calls))
	}
	call := llm.calls[0]
	assertSliceEqual(t, "重译原文", source[3:5], call.sentences)
	assertSliceEqual(t, "前文", source[:3], call.before)
	assertSliceEqual(t, "后文", source[5:], call.after)
	if review.translated[3] != "至今。" {
		t.Fatalf("错误前条未纠正：%v", review.translated)
	}
	if call.prompt == "" {
		t.Fatal("重译必须提供纠错提示词")
	}
}

// TestTranslationQualityDuplicateRules 区分真实重复与排版差异，保留标点和大小写的意义。
func TestTranslationQualityDuplicateRules(t *testing.T) {
	tests := []struct {
		name               string
		source, translated []string
		calls              int
	}{
		{"原文真实重复", []string{"Yes.", "Yes."}, []string{"是。", "是。"}, 0},
		{"原文仅空白不同", []string{"Yes. ", "Yes.\n"}, []string{"是。", "是。"}, 0},
		{"译文排版空白", []string{"first", "second"}, []string{"同 一句", "同一句\n"}, 1},
		{"译文标点不同", []string{"first", "second"}, []string{"是。", "是？"}, 0},
		{"译文大小写不同", []string{"first", "second"}, []string{"YES", "yes"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			review, llm := newQualityReview(tt.source, tt.translated, []string{"第一", "第二"})
			if err := review.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			if len(llm.calls) != tt.calls {
				t.Fatalf("调用 %d 次，期望 %d", len(llm.calls), tt.calls)
			}
		})
	}
}

// TestTranslationQualityPersistentDuplicateStops 确认失败重译会显式报错而非无限调用。
func TestTranslationQualityPersistentDuplicateStops(t *testing.T) {
	review, llm := newQualityReview([]string{"first", "second", "third"}, []string{"同句", "同句", "同句"}, []string{"仍同句", "仍同句", "仍同句"})
	if err := review.run(t.Context()); err == nil {
		t.Fatal("重译仍重复必须报错")
	}
	if len(llm.calls) != 1 || len(llm.calls[0].sentences) != 3 {
		t.Fatalf("连续重复组应整体仅重译一次：%+v", llm.calls)
	}
}

// TestTranslationQualityDuplicateConsumesBudget 防止同一条先纠重复再压缩导致重复计费。
func TestTranslationQualityDuplicateConsumesBudget(t *testing.T) {
	review, llm := newQualityReview([]string{"first", "second"}, []string{"同句", "同句"}, []string{"第一条仍然很长很长", "第二条仍然很长很长"})
	review.entries = []srtEntry{{EndSec: 0.1}, {EndSec: 0.1}}
	if err := review.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 1 {
		t.Fatalf("重复修正后不得再次压缩：%d", len(llm.calls))
	}
}

// TestTranslationQualityRejectsMalformedRetry 防止重译缺条或空白导致字幕内容丢失。
func TestTranslationQualityRejectsMalformedRetry(t *testing.T) {
	for _, reply := range [][]string{{"一条"}, {"", "第二条"}, {"第一条", " \n"}, {"第一", "第二", "第三"}} {
		t.Run(fmt.Sprint(reply), func(t *testing.T) {
			review, llm := newQualityReview([]string{"first", "second"}, []string{"重复", "重复"}, reply)
			if err := review.run(t.Context()); err == nil {
				t.Fatal("无效重译必须报错")
			}
			if len(llm.calls) != 1 {
				t.Fatalf("无效重译不得继续调用：%d", len(llm.calls))
			}
		})
	}
}

// TestTranslationQualityCrossChunkBeforeStream 确保批次边界重复在任何字幕流入配音前修正。
func TestTranslationQualityCrossChunkBeforeStream(t *testing.T) {
	llm := &qualityLLM{replies: [][]string{{"重复"}, {"重复"}, {"第一", "第二"}}}
	core := NewCore(Config{SemanticSplitReady: true, TranslateChunkSize: 1}, nil, llm, nil)
	stream := make(chan ttsPair, 4)
	translated, err := core.translateAllChunks(t.Context(), Job{Translator: "openai", TargetLang: "zh-CN"}, []string{"first", "second"}, stream)
	if err != nil {
		t.Fatal(err)
	}
	close(stream)
	assertSliceEqual(t, "结果", []string{"第一", "第二"}, translated)
	var spoken []string
	for pair := range stream {
		spoken = append(spoken, pair.text)
	}
	assertSliceEqual(t, "送入配音", []string{"第一", "第二"}, spoken)
	if len(llm.calls) != 3 {
		t.Fatalf("总调用次数 %d，期望 3", len(llm.calls))
	}
}

// TestTranslationQualityDuplicateGroupIncludesRepeatedSource 确保合法重复前缀也纳入完整的同文重译组。
func TestTranslationQualityDuplicateGroupIncludesRepeatedSource(t *testing.T) {
	source := []string{"A", "A", "B"}
	review, llm := newQualityReview(source, []string{"同文", "同文", "同文"}, []string{"甲", "甲", "乙"})
	if err := review.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 1 {
		t.Fatalf("完整重复组应只调用一次：%d", len(llm.calls))
	}
	assertSliceEqual(t, "完整重译组", source, llm.calls[0].sentences)
	assertSliceEqual(t, "保留原文合法重复", []string{"甲", "甲", "乙"}, review.translated)
}
