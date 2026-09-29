package tts

// SpeechOptions 保存服务确认的协议、配音语言与情绪参数。
type SpeechOptions struct {
	Protocol     string
	Language     string
	Instructions string
}

// WithSpeechOptions 返回不可变客户端副本，避免并发任务互相覆盖参数。
func (t *OpenAITTS) WithSpeechOptions(options SpeechOptions) *OpenAITTS {
	copy := *t
	copy.options = options
	return &copy
}
