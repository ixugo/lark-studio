package lipsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// MuseTalkClient 通过 HTTP API 调用 MuseTalk 对口型服务
// 支持自部署 MuseTalk API 或阿里云兼容接口
type MuseTalkClient struct {
	baseURL string
	apiKey  string
}

// NewMuseTalkClient 创建 MuseTalk 客户端
// baseURL: API 地址（如 http://localhost:7860）
// apiKey:  鉴权密钥，部分服务需要
func NewMuseTalkClient(baseURL, apiKey string) *MuseTalkClient {
	return &MuseTalkClient{baseURL: baseURL, apiKey: apiKey}
}

// Generate 上传视频+音频到 MuseTalk API，下载唇形同步后的视频
func (c *MuseTalkClient) Generate(ctx context.Context, videoPath, audioPath, outputPath string) error {
	slog.Info("lipsync request", "video", videoPath, "audio", audioPath, "output", outputPath)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := addFileField(writer, "video", videoPath); err != nil {
		return fmt.Errorf("附加视频文件失败: %w", err)
	}
	if err := addFileField(writer, "audio", audioPath); err != nil {
		return fmt.Errorf("附加音频文件失败: %w", err)
	}
	writer.Close()

	url := c.baseURL + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 MuseTalk API 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("MuseTalk API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("写入结果视频失败: %w", err)
	}

	slog.Info("lipsync done", "output", outputPath, "size", written)
	return nil
}

// addFileField 向 multipart writer 添加文件字段
func addFileField(w *multipart.Writer, fieldName, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	part, err := w.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, f)
	return err
}
