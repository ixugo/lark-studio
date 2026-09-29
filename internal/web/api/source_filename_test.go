package api

import (
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"os"
	"path/filepath"
	"testing"
	"uuid"
)

func TestStagedSourceUsesStableName(t *testing.T) {
	for _, ext := range []string{".mp4", ".mov", ".srt", ".wav"} {
		t.Run(ext, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "含空格 原文件"+ext)
			content := []byte("source bytes")
			if err := os.WriteFile(input, content, 0600); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(dir, "out")
			if err := os.Mkdir(output, 0700); err != nil {
				t.Fatal(err)
			}
			in := task.CreateTaskInput{InputPath: input, OutputDir: output}
			if err := StageSourceFile(&in); err != nil {
				t.Fatal(err)
			}
			want := in.InputPath
			id := filepath.Base(in.OutputDir)
			parsed, err := uuid.Parse(id)
			if err != nil || parsed[6]>>4 != 4 || filepath.Dir(in.OutputDir) != output || filepath.Base(want) != id+ext {
				t.Fatalf("源文件未按 UUIDv4 隔离: input=%q work=%q", want, in.OutputDir)
			}
			for _, path := range []string{input, want} {
				got, err := os.ReadFile(path)
				if err != nil || string(got) != string(content) {
					t.Fatalf("源文件内容被改动：%q %v", path, err)
				}
			}
			if err := StageSourceFile(&in); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(want)
			if err != nil || string(got) != string(content) {
				t.Fatal("重复暂存损坏源文件")
			}
		})
	}
}

func TestStagingOwnSourceDoesNotTruncate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "src.mp4")
	if err := os.WriteFile(path, []byte("keep source"), 0600); err != nil {
		t.Fatal(err)
	}
	in := task.CreateTaskInput{InputPath: path, OutputDir: dir}
	if err := StageSourceFile(&in); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "keep source" || in.InputPath == path {
		t.Fatal("同目录源文件被覆盖或改名")
	}
}

func TestPreparedTaskUsesConfiguredOutputRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	input := filepath.Join(home, "input.txt")
	if err := os.WriteFile(input, []byte("text"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{"", filepath.Join(home, "Documents"), filepath.Join(home, ".lark-studio", "tasks"), filepath.Join(home, "custom")} {
		bc := conf.DefaultConfig()
		bc.Pipeline.DefaultOutputDir = root
		service := TaskAPI{conf: &bc}
		in := task.CreateTaskInput{InputPath: input, Mode: pipeline.ModeTextTranslate, SubtitleOutput: "file"}
		if err := service.prepareTaskInput(&in); err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(home, "Documents", "lark-studio")
		if root == filepath.Join(home, "custom") {
			want = root
		}
		if got := filepath.Dir(filepath.Dir(in.OutputDir)); got != want {
			t.Fatalf("任务输出根目录=%q，期望 %q", got, want)
		}
	}
}
