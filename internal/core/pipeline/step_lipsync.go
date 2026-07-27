package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
)

// runLipSync 调用 LipSyncClient 对配音视频做唇形同步
// 输入: 原视频 + merge 阶段产出的 dub.mp3
// 输出: {outputDir}/lipsync.mp4，供后续 burn 步骤使用
func (c *Core) runLipSync(ctx context.Context, job Job) error {
	if c.lipSync == nil {
		return fmt.Errorf("对口型客户端未配置")
	}

	dubAudio := filepath.Join(job.OutputDir, "dub.mp3")
	outputVideo := filepath.Join(job.OutputDir, "lipsync.mp4")

	c.notifier.OnLog(job.TaskID, "开始对口型处理（MuseTalk）")
	c.notifier.OnProgress(job.TaskID, StepLipSync, 10)

	if err := c.lipSync.Generate(ctx, job.InputPath, dubAudio, outputVideo); err != nil {
		return fmt.Errorf("对口型处理失败: %w", err)
	}

	c.notifier.OnProgress(job.TaskID, StepLipSync, 100)
	return nil
}
