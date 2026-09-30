package wails

import (
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateConfigWorkersValidation(t *testing.T) {
	for _, value := range []any{float64(0), float64(-1), float64(9), float64(1.5), "2", nil} {
		bc := conf.DefaultConfig()
		bc.Runtime.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
		svc := &AppService{bc: &bc}
		if err := svc.UpdateConfig(map[string]any{"pipeline": map[string]any{"workers": value}}); err == nil {
			t.Fatalf("接受无效并发数 %v", value)
		}
		if bc.Pipeline.Workers != 2 {
			t.Fatal("无效输入改变内存配置")
		}
		if _, err := os.Stat(bc.Runtime.ConfigPath); !os.IsNotExist(err) {
			t.Fatal("无效输入写入配置")
		}
	}
}

func TestSavingWorkersUpdatesRunningScheduler(t *testing.T) {
	bc := conf.DefaultConfig()
	bc.Runtime.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	entered := make(chan string, 2)
	release := make(chan struct{}, 2)
	scheduler := pipeline.NewScheduler(pipeline.NewCore(pipeline.Config{}, nil, nil, nil), func(id string, err error) { entered <- id; <-release })
	if err := scheduler.SetWorkers(1); err != nil {
		t.Fatal(err)
	}
	scheduler.Start()
	t.Cleanup(func() { release <- struct{}{}; release <- struct{}{}; scheduler.Stop() })
	svc := &AppService{bc: &bc, scheduler: scheduler}
	for _, id := range []string{"a", "b"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "src.srt"), []byte("1\n00:00:00,000 --> 00:00:01,000\nhello\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := scheduler.Submit(pipeline.Job{TaskID: id, OutputDir: dir, Mode: pipeline.ModeSubtitle, SubtitleOutput: "none"}); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("首个任务未运行")
	}
	select {
	case <-entered:
		t.Fatal("保存前超出上限")
	case <-time.After(50 * time.Millisecond):
	}
	if err := svc.UpdateConfig(map[string]any{"pipeline": map[string]any{"workers": float64(2)}}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("保存并发数后排队任务未启动")
	}
	var loaded conf.Bootstrap
	if err := conf.SetupConfig(&loaded, bc.Runtime.ConfigPath); err != nil {
		t.Fatal(err)
	}
	if loaded.Pipeline.Workers != 2 {
		t.Fatal("并发数未保存")
	}
}
