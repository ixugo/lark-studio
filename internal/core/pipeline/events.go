package pipeline

import (
	"fmt"
	"time"
)

// detailedNotifier 承载结构化步骤和日志事件，旧测试通知器仍可只实现基础接口。
type detailedNotifier interface {
	OnStepStart(taskID, step string)
	OnStepFailed(taskID, step string, elapsed time.Duration, err error)
	OnDetailedLog(taskID, level, step, message string)
}

// logEvent 发送带级别和步骤的日志；基础通知器仍能收到纯文本。
func (c *Core) logEvent(taskID, level, step, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if notifier, ok := c.notifier.(detailedNotifier); ok {
		notifier.OnDetailedLog(taskID, level, step, message)
		return
	}
	c.notifier.OnLog(taskID, message)
}

// startStep 通知持久层创建或恢复步骤记录。
func (c *Core) startStep(taskID, step string) {
	if notifier, ok := c.notifier.(detailedNotifier); ok {
		notifier.OnStepStart(taskID, step)
	}
	c.notifier.OnProgress(taskID, step, 0)
}

// failStep 保存失败步骤和耗时，基础通知器仍走原有任务失败事件。
func (c *Core) failStep(taskID, step string, elapsed time.Duration, err error) {
	if notifier, ok := c.notifier.(detailedNotifier); ok {
		notifier.OnStepFailed(taskID, step, elapsed, err)
	}
	c.notifier.OnTaskFailed(taskID, err)
}

// stepTitle 返回全中文步骤名称，供任务日志和界面复用。
func stepTitle(step string) string {
	switch step {
	case StepWhisper:
		return "听写转录"
	case StepSplit:
		return "语义分句"
	case StepTranslate:
		return "智能翻译"
	case StepTTS:
		return "语音合成"
	case StepMerge:
		return "音视频混音"
	case StepLipSync:
		return "唇形同步"
	case StepBurn:
		return "压制合成"
	default:
		return step
	}
}

// modeTitle 返回任务模式的中文名称。
func modeTitle(mode int) string {
	switch mode {
	case ModeSubtitle:
		return "原文字幕"
	case ModeTranslate:
		return "双语字幕"
	case ModeDub:
		return "配音成片"
	case ModeDubOnly:
		return "文本朗读/配音"
	default:
		return "视频处理"
	}
}
