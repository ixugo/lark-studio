package asr

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ixugo/vdub/internal/adapter/whisper"
)

const (
	transcriptionTimeout = 30 * time.Minute
	maxTranscriptionBody = 32 << 20
	maxAudioBytes        = 25 << 20
)

// Config 描述一次 ASR 转录所需的引擎与模型参数。
type Config struct {
	Engine       string
	WhisperBin   string
	WhisperModel string
	BaseURL      string
	APIKey       string
	Model        string
}

// Router 让新任务即时使用已保存的 ASR 默认引擎。
type Router struct {
	config atomic.Pointer[Config]
	client *http.Client
}

// NewRouter 创建本地 Whisper 与 OpenAI 兼容 ASR 的路由器。
func NewRouter(config Config) *Router {
	router := &Router{client: &http.Client{Timeout: transcriptionTimeout}}
	router.SetConfig(config)
	return router
}

// SetConfig 原子发布新的设置，避免正在转录时读到半套配置。
func (r *Router) SetConfig(config Config) {
	snapshot := new(Config)
	*snapshot = config
	r.config.Store(snapshot)
}

// ValidateConfig 检查 OpenAI ASR 的地址与模型输入是否可用于请求。
func ValidateConfig(config Config) error {
	if config.Engine != "" && config.Engine != "whisper-cpp" && config.Engine != "openai" {
		return fmt.Errorf("不支持的 ASR 引擎: %s", config.Engine)
	}
	if config.Engine != "openai" {
		return nil
	}
	if len(config.BaseURL) > 2048 || len(config.Model) > 100 || len(config.APIKey) > 4096 {
		return fmt.Errorf("OpenAI ASR 配置长度无效")
	}
	parsed, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("OpenAI ASR Base URL 必须是有效的 HTTP 或 HTTPS 地址")
	}
	if strings.TrimSpace(config.Model) == "" {
		return fmt.Errorf("OpenAI ASR 模型名称不能为空")
	}
	if config.Model != strings.TrimSpace(config.Model) {
		return fmt.Errorf("OpenAI ASR 模型名称不能包含首尾空格")
	}
	return nil
}

// Transcribe 按已保存的引擎配置转录，OpenAI 响应直接写为 SRT。
func (r *Router) Transcribe(ctx context.Context, audioPath, outputSRT, lang string, onProgress func(int), onLog func(string)) error {
	config := r.config.Load()
	if config == nil {
		return fmt.Errorf("ASR 引擎尚未配置")
	}
	if config.Engine == "openai" {
		return r.transcribeOpenAI(ctx, *config, audioPath, outputSRT, lang, onProgress, onLog)
	}
	return whisper.NewRunner(config.WhisperBin, config.WhisperModel).Transcribe(ctx, audioPath, outputSRT, lang, onProgress, onLog)
}

// transcribeOpenAI 调用 OpenAI 兼容转录接口并保留服务端返回的时间轴。
func (r *Router) transcribeOpenAI(ctx context.Context, config Config, audioPath, outputSRT, lang string, onProgress func(int), onLog func(string)) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}
	request, err := newTranscriptionRequest(ctx, config, audioPath, lang)
	if err != nil {
		return err
	}
	if onLog != nil {
		onLog("正在调用 OpenAI 兼容语音识别接口")
	}
	if onProgress != nil {
		onProgress(5)
	}
	response, err := r.client.Do(request)
	if err != nil {
		return fmt.Errorf("OpenAI ASR 请求失败: %w", err)
	}
	defer response.Body.Close()
	return writeTranscriptionResponse(response, outputSRT, onProgress)
}

// newTranscriptionRequest 构造带音频文件、模型与语言参数的 multipart 请求。
func newTranscriptionRequest(ctx context.Context, config Config, audioPath, lang string) (*http.Request, error) {
	audio, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("打开待识别音频失败: %w", err)
	}
	defer audio.Close()
	info, err := audio.Stat()
	if err != nil {
		return nil, fmt.Errorf("读取待识别音频信息失败: %w", err)
	}
	if info.Size() > maxAudioBytes {
		return nil, fmt.Errorf("待识别音频超过 25 MiB，请先缩短音频")
	}
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	if err := addTranscriptionFields(writer, audio, audioPath, config.Model, lang); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("结束 ASR 请求体失败: %w", err)
	}
	endpoint := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if !strings.HasSuffix(endpoint, "/audio/transcriptions") {
		endpoint += "/audio/transcriptions"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("构造 OpenAI ASR 请求失败: %w", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if config.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+config.APIKey)
	}
	return request, nil
}

// addTranscriptionFields 添加 OpenAI 标准字段并流式复制音频内容。
func addTranscriptionFields(writer *multipart.Writer, audio io.Reader, audioPath, model, lang string) error {
	file, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return fmt.Errorf("添加 ASR 音频字段失败: %w", err)
	}
	if _, err := io.Copy(file, audio); err != nil {
		return fmt.Errorf("写入 ASR 音频字段失败: %w", err)
	}
	fields := map[string]string{"model": model, "response_format": "srt"}
	if lang != "" && lang != "auto" {
		fields["language"] = lang
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("添加 ASR 参数 %s 失败: %w", key, err)
		}
	}
	return nil
}

// writeTranscriptionResponse 限制响应体大小并保存服务端生成的 SRT。
func writeTranscriptionResponse(response *http.Response, outputSRT string, onProgress func(int)) error {
	data, err := io.ReadAll(io.LimitReader(response.Body, maxTranscriptionBody+1))
	if err != nil {
		return fmt.Errorf("读取 OpenAI ASR 响应失败: %w", err)
	}
	if len(data) > maxTranscriptionBody {
		return fmt.Errorf("OpenAI ASR 响应超过限制")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		detail := strings.TrimSpace(string(data))
		if len(detail) > 1024 {
			detail = detail[:1024]
		}
		return fmt.Errorf("OpenAI ASR 返回 HTTP %d: %s", response.StatusCode, detail)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return fmt.Errorf("OpenAI ASR 返回空字幕")
	}
	if err := os.MkdirAll(filepath.Dir(outputSRT), 0o755); err != nil {
		return fmt.Errorf("创建字幕目录失败: %w", err)
	}
	if err := os.WriteFile(outputSRT, data, 0o644); err != nil {
		return fmt.Errorf("保存 OpenAI ASR 字幕失败: %w", err)
	}
	if onProgress != nil {
		onProgress(100)
	}
	return nil
}
