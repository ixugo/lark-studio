package pipeline

import "context"

// WhisperRunner 语音识别接口
type WhisperRunner interface {
	// Transcribe 转写音频文件为 SRT 字幕
	// audioPath: 输入音频路径
	// outputSRT: 输出 SRT 文件路径
	// lang: 源语言（如 "en", "auto"）
	Transcribe(ctx context.Context, audioPath, outputSRT, lang string) error
}

// LLMClient 大模型翻译/分句接口
type LLMClient interface {
	// SplitSentences 用 LLM 做语义分句
	// 输入一段文本，返回按语义切分后的句子列表
	SplitSentences(ctx context.Context, text string, lang string) ([]string, error)

	// Translate 翻译文本
	// sentences: 待翻译句子列表
	// targetLang: 目标语言
	// systemPrompt: 自定义系统提示词，空串则使用内置默认
	// contextBefore/contextAfter: 上下文句子（仅供 LLM 参考，不翻译）
	// 返回翻译后的句子列表（与 sentences 一一对应）
	Translate(ctx context.Context, sentences []string, targetLang, systemPrompt string, contextBefore, contextAfter []string) ([]string, error)
}

// TTSClient 文本转语音接口
type TTSClient interface {
	// Synthesize 合成单句语音
	// text: 待合成文本
	// outputPath: 输出音频文件路径
	// voice: 语音名称/ID
	Synthesize(ctx context.Context, text, outputPath, voice string) error
}

// TermMapping 术语映射条目
type TermMapping struct {
	Text        string
	Translation string
}

// TermLister 术语列表查询接口，翻译时向 LLM 注入术语表
type TermLister interface {
	ListMappings(ctx context.Context) ([]TermMapping, error)
}
