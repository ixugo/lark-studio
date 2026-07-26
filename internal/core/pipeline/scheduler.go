package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

const defaultWorkerCount = 2

// Scheduler 任务调度器，管理多 worker 并行处理
type Scheduler struct {
	core      *Core
	workerNum int
	jobs      chan Job
	cancel    context.CancelFunc
	ctx       context.Context
	wg        sync.WaitGroup

	// 任务完成回调
	onDone func(taskID string, err error)
}

// NewScheduler 创建调度器
func NewScheduler(core *Core, onDone func(taskID string, err error)) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler{
		core:      core,
		workerNum: defaultWorkerCount,
		jobs:      make(chan Job, 100),
		ctx:       ctx,
		cancel:    cancel,
		onDone:    onDone,
	}
	return s
}

// Start 启动 worker 协程池
func (s *Scheduler) Start() {
	for i := range s.workerNum {
		s.wg.Add(1)
		go s.worker(i)
	}
	slog.Info("scheduler started", "workers", s.workerNum)
}

// Stop 停止调度器，等待所有 worker 完成当前任务
func (s *Scheduler) Stop() {
	s.cancel()
	close(s.jobs)
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

// Submit 提交任务到队列
func (s *Scheduler) Submit(job Job) error {
	select {
	case <-s.ctx.Done():
		return fmt.Errorf("调度器已停止")
	case s.jobs <- job:
		slog.Info("job submitted", "task_id", job.TaskID, "input", job.InputPath)
		return nil
	}
}

// worker 工作协程
func (s *Scheduler) worker(id int) {
	defer s.wg.Done()
	slog.Info("worker started", "id", id)

	for job := range s.jobs {
		if s.ctx.Err() != nil {
			break
		}

		slog.Info("worker processing", "worker", id, "task_id", job.TaskID)
		err := s.core.Run(s.ctx, job)
		if s.onDone != nil {
			s.onDone(job.TaskID, err)
		}
	}

	slog.Info("worker stopped", "id", id)
}
