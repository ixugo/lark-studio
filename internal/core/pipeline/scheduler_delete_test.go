package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDeleteKeepsOriginalRunTracking 防止重复入队覆盖退出信号，导致先删文件后仍有后台写入。
func TestDeleteKeepsOriginalRunTracking(t *testing.T) {
	scheduler := NewScheduler(NewCore(Config{}, nil, nil, nil), nil)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	if !scheduler.trackTask("same", cancel) {
		t.Fatal("首次任务应能启动")
	}
	if scheduler.trackTask("same", func() {}) {
		t.Fatal("重复任务不应覆盖现有执行状态")
	}
	waitCtx, stop := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer stop()
	if err := scheduler.CancelAndWait(waitCtx, "same"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("应等待原执行退出：%v", err)
	}
	if ctx.Err() == nil {
		t.Fatal("未取消原执行")
	}
	scheduler.untrackTask("same")
	if err := scheduler.CancelAndWait(t.Context(), "same"); err != nil {
		t.Fatal(err)
	}
}

// deletionJob 用已有字幕完成轻量真实流水线，让测试独立于外部语音服务。
func deletionJob(t *testing.T, id string) Job {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "src.srt"), []byte("1\n00:00:00,000 --> 00:00:01,000\nHello\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return Job{TaskID: id, OutputDir: dir, Mode: ModeSubtitle, SubtitleOutput: "none"}
}

// TestDeleteWaitsForCompletionCallback 验证删除等待写文件的收尾回调，而不只是发送取消信号。
func TestDeleteWaitsForCompletionCallback(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	scheduler := NewScheduler(NewCore(Config{}, nil, nil, nil), func(id string, err error) { close(entered); <-release })
	scheduler.Start()
	t.Cleanup(func() { once.Do(func() { close(release) }); scheduler.Stop() })
	job := deletionJob(t, "running")
	if err := scheduler.Submit(job); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("任务未进入收尾回调")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if err := scheduler.CancelAndWait(ctx, job.TaskID); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("回调未结束时删除不应继续：%v", err)
	}
	once.Do(func() { close(release) })
	if err := scheduler.CancelAndWait(t.Context(), job.TaskID); err != nil {
		t.Fatal(err)
	}
	if scheduler.IsRunning(job.TaskID) {
		t.Fatal("删除返回后任务仍在运行")
	}
}

// TestDeletePreventsQueuedTaskFromStarting 用真实队列检验已删除任务不会再次生成文件。
func TestDeletePreventsQueuedTaskFromStarting(t *testing.T) {
	var deletedRuns atomic.Int32
	controlDone := make(chan struct{})
	scheduler := NewScheduler(NewCore(Config{}, nil, nil, nil), func(id string, err error) {
		if id == "deleted" {
			deletedRuns.Add(1)
		} else {
			close(controlDone)
		}
	})
	scheduler.workerNum = 1
	job := deletionJob(t, "deleted")
	if err := scheduler.Submit(job); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.CancelAndWait(t.Context(), job.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Submit(deletionJob(t, "control")); err != nil {
		t.Fatal(err)
	}
	scheduler.Start()
	t.Cleanup(scheduler.Stop)
	select {
	case <-controlDone:
	case <-time.After(3 * time.Second):
		t.Fatal("队列未继续处理后续任务")
	}
	if deletedRuns.Load() != 0 {
		t.Fatal("已删除任务仍然被执行")
	}
	if err := scheduler.Submit(job); err == nil {
		t.Fatal("已删除任务不应重新入队")
	}
	if _, err := os.Stat(filepath.Join(job.OutputDir, "task.log")); !os.IsNotExist(err) {
		t.Fatalf("已删除任务写入了产物：%v", err)
	}
}
