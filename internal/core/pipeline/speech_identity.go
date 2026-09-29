package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type taskSpeechClient interface {
	SnapshotTask(context.Context, string, string, string) (context.Context, error)
}

// prepareSpeechContext 固定整部视频的客户端、语言与音色，并拒绝混用旧声音缓存。
func (c *Core) prepareSpeechContext(ctx context.Context, job Job) (context.Context, error) {
	client, ok := c.tts.(taskSpeechClient)
	if !ok {
		return ctx, nil
	}
	c.ttsMu.Lock()
	defer c.ttsMu.Unlock()
	if err := ctx.Err(); err != nil {
		return ctx, err
	}
	language := job.SourceLang
	if job.Mode == ModeDub {
		language = job.TargetLang
	}
	taskCtx, err := client.SnapshotTask(ctx, job.TTSEngine, language, job.TTSVoice)
	if err != nil {
		return ctx, err
	}
	if err := c.ensureSpeechIdentity(taskCtx, job); err != nil {
		return ctx, err
	}
	return taskCtx, nil
}

// ensureSpeechIdentity 避免恢复任务时把旧音色缓存和新音色混在同一视频中。
func (c *Core) ensureSpeechIdentity(ctx context.Context, job Job) error {
	client, ok := c.tts.(interface {
		SpeechIdentity(context.Context, string, float64) string
	})
	if !ok {
		return nil
	}
	identity := client.SpeechIdentity(ctx, job.TTSVoice, job.SpeechRate)
	if identity == "" {
		return fmt.Errorf("无法确定配音配置身份")
	}
	path := filepath.Join(job.OutputDir, "speech_identity.sha256")
	previous, err := os.ReadFile(path)
	if err == nil && strings.TrimSpace(string(previous)) == identity {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(job.OutputDir, "audio_segs")); err != nil {
		return fmt.Errorf("清理旧音色缓存失败: %w", err)
	}
	return os.WriteFile(path, []byte(identity), 0600)
}
