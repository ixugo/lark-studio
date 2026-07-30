//go:build integration

package pipeline_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ixugo/vdub/internal/adapter/llm"
	"github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/adapter/whisper"
	"github.com/ixugo/vdub/internal/core/pipeline"
)

const testVideoPath = "/path/to/integration-video.mp4"

// TestIntegration_TranslatePipeline 使用 stub whisper 的翻译流水线测试
//
// 运行: go test -tags=integration -run TestIntegration_TranslatePipeline -v -timeout 10m ./internal/core/pipeline/
//
// 环境变量:
//
//	VDUB_LLM_BASE_URL  LLM API 地址（必填）
//	VDUB_LLM_API_KEY   LLM API 密钥
//	VDUB_LLM_MODEL     LLM 模型名（默认 opus）
func TestIntegration_TranslatePipeline(t *testing.T) {
	requireCmd(t, "ffmpeg")
	requireFile(t, testVideoPath)
	llmClient, _ := requireLLM(t)

	workDir := makeWorkDir(t)
	clipped := clipTestVideo(t, workDir)

	mockWhisper := &stubWhisper{srt: generateTestSRT()}
	edgeTTS := tts.NewEdgeTTS("zh-CN-YunjianNeural")
	cfg := pipeline.Config{FFmpegBin: "ffmpeg"}
	pipeCore := pipeline.NewCore(cfg, mockWhisper, llmClient, edgeTTS)

	job := pipeline.Job{
		TaskID:     "test_translate",
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

	assertOutputFiles(t, workDir, "src.srt", "trans.txt", "trans.srt")
	assertSRTQuality(t, filepath.Join(workDir, "trans.srt"))
}

// TestIntegration_E2E_Whisper 真实 whisper.cpp → split → translate → burn
// 端到端验证：裁剪视频 10~30s，用真实语音识别 + LLM 翻译 + 字幕烧录
//
// 运行: VDUB_LLM_BASE_URL=... VDUB_LLM_API_KEY=... VDUB_WHISPER_MODEL=/path/to/model.bin \
//
//	go test -tags=integration -run TestIntegration_E2E_Whisper -v -timeout 15m ./internal/core/pipeline/
func TestIntegration_E2E_Whisper(t *testing.T) {
	requireCmd(t, "ffmpeg")
	requireFile(t, testVideoPath)
	llmClient, _ := requireLLM(t)
	whisperModel := requireWhisperModel(t)

	workDir := makeWorkDir(t)
	clipped := clipTestVideo(t, workDir)

	wr := whisper.NewRunner("whisper-cli", whisperModel)
	edgeTTS := tts.NewEdgeTTS("zh-CN-YunjianNeural")
	cfg := pipeline.Config{FFmpegBin: "ffmpeg", MaxSpeedFactor: 1.2}
	pipeCore := pipeline.NewCore(cfg, wr, llmClient, edgeTTS)

	job := pipeline.Job{
		TaskID:     "test_e2e",
		InputPath:  clipped,
		OutputDir:  workDir,
		Mode:       pipeline.ModeTranslate,
		TargetLang: "zh-CN",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	if err := pipeCore.Run(ctx, job); err != nil {
		t.Fatalf("E2E 流水线执行失败: %v", err)
	}

	assertOutputFiles(t, workDir, "src.srt", "trans.txt", "trans.srt")
	assertSRTQuality(t, filepath.Join(workDir, "trans.srt"))

	// 验证烧录输出视频
	baseName := strings.TrimSuffix(filepath.Base(clipped), filepath.Ext(clipped))
	outputVideo := filepath.Join(workDir, baseName+".trans.mp4")
	assertOutputFiles(t, workDir, baseName+".trans.mp4")
	t.Logf("输出视频: %s", outputVideo)
}

// TestIntegration_E2E_Dub 配音端到端测试：whisper → translate + TTS → merge → burn
//
// 运行: VDUB_LLM_BASE_URL=... VDUB_LLM_API_KEY=... VDUB_WHISPER_MODEL=/path/to/model.bin \
//
//	go test -tags=integration -run TestIntegration_E2E_Dub -v -timeout 15m ./internal/core/pipeline/
func TestIntegration_E2E_Dub(t *testing.T) {
	requireCmd(t, "ffmpeg")
	requireCmd(t, "edge-tts")
	requireFile(t, testVideoPath)
	llmClient, _ := requireLLM(t)
	whisperModel := requireWhisperModel(t)

	workDir := makeWorkDir(t)
	clipped := clipTestVideo(t, workDir)

	wr := whisper.NewRunner("whisper-cli", whisperModel)
	edgeTTS := tts.NewEdgeTTS("zh-CN-YunjianNeural")
	cfg := pipeline.Config{
		FFmpegBin:      "ffmpeg",
		MaxSpeedFactor: 1.2,
		TTSWorkers:     2,
	}
	pipeCore := pipeline.NewCore(cfg, wr, llmClient, edgeTTS)

	job := pipeline.Job{
		TaskID:     "test_dub",
		InputPath:  clipped,
		OutputDir:  workDir,
		Mode:       pipeline.ModeDub,
		TargetLang: "zh-CN",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	if err := pipeCore.Run(ctx, job); err != nil {
		t.Fatalf("配音 E2E 流水线执行失败: %v", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(clipped), filepath.Ext(clipped))
	assertOutputFiles(t, workDir, "src.srt", "trans.txt", "trans.srt", "dub.mp3", baseName+".final.mp4")
}

// --- helpers ---

func requireLLM(t *testing.T) (*llm.Client, string) {
	t.Helper()
	baseURL := os.Getenv("VDUB_LLM_BASE_URL")
	if baseURL == "" {
		t.Skip("VDUB_LLM_BASE_URL not set")
	}
	apiKey := os.Getenv("VDUB_LLM_API_KEY")
	model := os.Getenv("VDUB_LLM_MODEL")
	if model == "" {
		model = "opus"
	}
	return llm.NewClient(baseURL, apiKey, model), model
}

func requireWhisperModel(t *testing.T) string {
	t.Helper()
	model := os.Getenv("VDUB_WHISPER_MODEL")
	if model == "" {
		t.Skip("VDUB_WHISPER_MODEL not set")
	}
	if _, err := os.Stat(model); err != nil {
		t.Skipf("whisper model not found: %s", model)
	}
	return model
}

func makeWorkDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("vdub_test_%d", time.Now().Unix()))
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建工作目录失败: %v", err)
	}
	return dir
}

func clipTestVideo(t *testing.T, workDir string) string {
	t.Helper()
	clipped := filepath.Join(workDir, "clipped.mp4")
	cmd := exec.Command("ffmpeg",
		"-y", "-i", testVideoPath,
		"-ss", "10", "-to", "30",
		"-c", "copy", clipped,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg 裁剪失败: %v\n%s", err, output)
	}
	t.Logf("裁剪完成: %s", clipped)
	return clipped
}

func assertOutputFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, f := range names {
		path := filepath.Join(dir, f)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("产物缺失: %s", f)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("产物为空: %s", f)
		}
		t.Logf("  %s (%d bytes)", f, info.Size())
	}
}

// assertSRTQuality 检查翻译 SRT 基本质量：含中文字符、条目数 > 0
func assertSRTQuality(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("读取 %s 失败: %v", path, err)
		return
	}
	content := string(data)
	if !strings.Contains(content, "-->") {
		t.Errorf("trans.srt 缺少时间轴标记")
	}
	hasChinese := false
	for _, r := range content {
		if r >= 0x4E00 && r <= 0x9FFF {
			hasChinese = true
			break
		}
	}
	if !hasChinese {
		t.Errorf("trans.srt 不含中文翻译内容")
	}
	t.Logf("  翻译 SRT 质量检查通过 (%d bytes)", len(data))
}

// stubWhisper 模拟 whisper，直接写入预制 SRT 文件
type stubWhisper struct {
	srt string
}

func (s *stubWhisper) Transcribe(
	_ context.Context,
	_ string,
	outputSRT string,
	_ string,
	onProgress func(int),
	onLog func(string),
) error {
	if onLog != nil {
		onLog("测试听写完成")
	}
	if onProgress != nil {
		onProgress(100)
	}
	return os.WriteFile(outputSRT, []byte(s.srt), 0o644)
}

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
