package conf

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigUsesXiaoxiaoVoice(t *testing.T) {
	t.Parallel()

	if got := DefaultConfig().TTS.Voice; got != "zh-CN-XiaoxiaoNeural" {
		t.Fatalf("默认音色应为晓晓，实际为 %q", got)
	}
}

func TestDefaultOutputUsesDocumentsLarkStudio(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, "Documents", "lark-studio")
	if got := DefaultConfig().Pipeline.DefaultOutputDir; got != want {
		t.Fatalf("默认输出目录 = %q，期望 %q", got, want)
	}
}

func TestTaskOutputDirKeepsCustomDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, "Documents", "lark-studio")
	for _, input := range []string{"", filepath.Join(home, "Documents"), "~/Documents"} {
		if got := TaskOutputDir(input); got != root {
			t.Fatalf("旧默认目录 %q 未更新：%q", input, got)
		}
	}
	custom := filepath.Join(home, "custom")
	if got := TaskOutputDir(custom); got != custom {
		t.Fatalf("自定义目录被改动：%q", got)
	}
}
