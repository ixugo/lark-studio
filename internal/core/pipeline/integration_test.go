//go:build integration

package pipeline_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ixugo/vdub/internal/adapter/llm"
	"github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/core/pipeline"
)

const testVideoPath = "/path/to/integration-video.mp4"

// TestIntegration_TranslatePipeline 端到端集成测试
// 裁剪视频 10~30s → whisper → split → translate → burn
//
// 运行: go test -tags=integration -run TestIntegration_TranslatePipeline -v -timeout 10m ./internal/core/pipeline/
//
// 环境变量:
//
//	VDUB_LLM_BASE_URL  LLM API 地址（必填）
//	VDUB_LLM_API_KEY   LLM API 密钥
//	VDUB_LLM_MODEL     LLM 模型名（默认 qwen3-235b-a22b）
func TestIntegration_TranslatePipeline(t *testing.T) {
	requireCmd(t, "ffmpeg")
	requireCmd(t, "edge-tts")
	requireFile(t, testVideoPath)

	baseURL := os.Getenv("VDUB_LLM_BASE_URL")
	if baseURL == "" {
		t.Skip("VDUB_LLM_BASE_URL not set, skipping integration test")
	}
	apiKey := os.Getenv("VDUB_LLM_API_KEY")
	model := os.Getenv("VDUB_LLM_MODEL")
	if model == "" {
		model = "qwen3-235b-a22b"
	}

	// 工作目录
	workDir := filepath.Join(os.TempDir(), fmt.Sprintf("vdub_test_%d", time.Now().Unix()))
	t.Cleanup(func() { os.RemoveAll(workDir) })
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("创建工作目录失败: %v", err)
	}

	// 裁剪 10~30s
	clipped := filepath.Join(workDir, "clipped.mp4")
	clipCmd := exec.Command("ffmpeg",
		"-y", "-i", testVideoPath,
		"-ss", "10", "-to", "30",
		"-c", "copy", clipped,
	)
	if output, err := clipCmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg 裁剪失败: %v\n%s", err, output)
	}

	// 组装适配器
	llmClient := llm.NewClient(baseURL, apiKey, model)
	edgeTTS := tts.NewEdgeTTS("zh-CN-YunjianNeural")

	// whisper 跳过：手动提供一个简单 SRT 做后续测试
	mockWhisper := &stubWhisper{srt: generateTestSRT()}

	cfg := pipeline.Config{FFmpegBin: "ffmpeg"}
	pipeCore := pipeline.NewCore(cfg, mockWhisper, llmClient, edgeTTS)

	job := pipeline.Job{
		TaskID:     "integration_test",
		InputPath:  clipped,
		OutputDir:  workDir,
		Mode:       pipeline.ModeTranslate,
		TargetLang: "zh-CN",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := pipeCore.Run(ctx, job); err != nil {
		t.Fatalf("流水线执行失败: %v", err)
	}

	// 验证产物（split 步骤已从流水线摘除，翻译直接读取 src.srt）
	expectFiles := []string{"src.srt", "trans.txt", "trans.srt"}
	for _, f := range expectFiles {
		path := filepath.Join(workDir, f)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("产物缺失: %s", f)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("产物为空: %s", f)
		}
		t.Logf("✓ %s (%d bytes)", f, info.Size())
	}
}

// stubWhisper 模拟 whisper，直接写入预制 SRT 文件
type stubWhisper struct {
	srt string
}

func (s *stubWhisper) Transcribe(_ context.Context, _, outputSRT, _ string) error {
	return os.WriteFile(outputSRT, []byte(s.srt), 0o644)
}

// generateTestSRT 生成一段模拟 SRT（模拟 20 秒英语演讲）
func generateTestSRT() string {
	return `1
00:00:00,000 --> 00:00:04,500
Welcome to this course on software design with Go.

2
00:00:04,800 --> 00:00:09,200
We're going to talk about design philosophy and guidelines.

3
00:00:09,500 --> 00:00:14,000
The goal is to help you write better, more maintainable code.

4
00:00:14,300 --> 00:00:19,800
Let's start by understanding what good software design actually means.
`
}

func requireCmd(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not found in PATH, skipping", name)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("test file not found: %s", path)
	}
}
