package pipeline

import (
	"encoding/json/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"uuid"

	"github.com/ixugo/vdub/internal/taskfile"
)

func layoutWork(t *testing.T, root, extension, resultName string) (string, string, string) {
	t.Helper()
	id := uuid.NewV4().String()
	work := filepath.Join(root, id)
	if err := os.Mkdir(work, 0755); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(work, id+extension)
	if resultName == "" {
		resultName = id + ".mp4"
	}
	meta := taskfile.Metadata{LayoutVersion: 1, OriginalName: "原视频" + extension, OriginalPath: filepath.Join(root, "original"+extension), StagedName: filepath.Base(input), StagedPath: input, ResultName: resultName}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "source_meta.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return work, input, filepath.Join(root, resultName)
}

func TestBurnWritesSiblingVideosAndRerunKeepsOtherTask(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Fatal("需要本机 ffmpeg 验证成片")
	}
	root := t.TempDir()
	var other string
	for range 2 {
		work, input, result := layoutWork(t, root, ".mp4", "")
		output, err := exec.CommandContext(t.Context(), ffmpeg, "-v", "error", "-y", "-f", "lavfi", "-i", "color=size=32x32:rate=10:duration=0.2", "-c:v", "mpeg4", input).CombinedOutput()
		if err != nil {
			t.Fatalf("生成测试视频: %v %s", err, output)
		}
		core := NewCore(Config{FFmpegBin: ffmpeg}, nil, nil, nil)
		if err := core.runBurn(t.Context(), Job{InputPath: input, OutputDir: work, Mode: ModeSubtitle, SubtitleOutput: "none"}); err != nil {
			t.Fatal(err)
		}
		result, err = taskfile.ResultVideoPath(work)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(result) != "原视频.mp4" && filepath.Base(result) != "原视频_1.mp4" {
			t.Fatal(result)
		}
		if info, err := os.Stat(result); err != nil || info.Size() == 0 {
			t.Fatalf("成片未输出到批次目录: %v", err)
		}
		if output, err := exec.CommandContext(t.Context(), ffmpeg, "-v", "error", "-i", result, "-f", "null", "-").CombinedOutput(); err != nil {
			t.Fatalf("成片解码失败: %v %s", err, output)
		}
		if other == "" {
			other = result
			continue
		}
		CleanStepAndSubsequent(work, StepWhisper)
		if _, err := os.Stat(result); !os.IsNotExist(err) {
			t.Fatal("重跑未删除本任务成片")
		}
		for _, keep := range []string{input, other, filepath.Join(work, "source_meta.json")} {
			if _, err := os.Stat(keep); err != nil {
				t.Fatalf("重跑误删前置输入或其他任务: %q %v", keep, err)
			}
		}
	}
}

func TestCleanupKeepsUUIDTextAudioAndSubtitleInputs(t *testing.T) {
	for _, ext := range []string{".txt", ".srt", ".wav", ".mov"} {
		t.Run(ext, func(t *testing.T) {
			work, input, _ := layoutWork(t, t.TempDir(), ext, "output.mp4")
			if err := os.WriteFile(input, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(work, "raw.mp3"), []byte("intermediate"), 0600); err != nil {
				t.Fatal(err)
			}
			cleanIntermediate(work)
			if data, err := os.ReadFile(input); err != nil || string(data) != "original" {
				t.Fatalf("清理删除 UUID 输入: %v", err)
			}
		})
	}
}
