package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

const defaultWorkerCount = 2

// Scheduler 任务调度器，管理多 worker 并行处理，支持按 task 粒度暂停
type Scheduler struct {
	core      *Core
	workerNum int
	jobs      chan Job
	cancel    context.CancelFunc
	ctx       context.Context
	wg        sync.WaitGroup

	onDone func(taskID string, err error)

	mu          sync.Mutex
	taskCancels map[string]context.CancelFunc
	pausedTasks map[string]bool
}

// NewScheduler 创建调度器
func NewScheduler(core *Core, onDone func(taskID string, err error)) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		core:        core,
		workerNum:   defaultWorkerCount,
		jobs:        make(chan Job, 100),
		ctx:         ctx,
		cancel:      cancel,
		onDone:      onDone,
		taskCancels: make(map[string]context.CancelFunc),
		pausedTasks: make(map[string]bool),
	}
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

// Pause 暂停正在运行的任务，返回 false 表示该任务不在运行中
func (s *Scheduler) Pause(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cancel, ok := s.taskCancels[taskID]
	if !ok {
		return false
	}
	s.pausedTasks[taskID] = true
	cancel()
	delete(s.taskCancels, taskID)
	return true
}

// WasPaused 检查任务是否为主动暂停导致的终止，调用后清除记录
func (s *Scheduler) WasPaused(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pausedTasks[taskID] {
		delete(s.pausedTasks, taskID)
		return true
	}
	return false
}

// IsRunning 检查指定任务是否正在执行中
func (s *Scheduler) IsRunning(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.taskCancels[taskID]
	return ok
}

// worker 工作协程，每个 job 使用独立的 context 以支持单任务暂停
func (s *Scheduler) worker(id int) {
	defer s.wg.Done()
	slog.Info("worker started", "id", id)

	for job := range s.jobs {
		if s.ctx.Err() != nil {
			break
		}

		taskCtx, cancel := context.WithCancel(s.ctx)
		s.trackTask(job.TaskID, cancel)

		slog.Info("worker processing", "worker", id, "task_id", job.TaskID)
		err := s.core.Run(taskCtx, job)

		s.untrackTask(job.TaskID)
		cancel()

		if s.onDone != nil {
			s.onDone(job.TaskID, err)
		}
	}

	slog.Info("worker stopped", "id", id)
}

func (s *Scheduler) trackTask(taskID string, cancel context.CancelFunc) {
	s.mu.Lock()
	s.taskCancels[taskID] = cancel
	s.mu.Unlock()
}

func (s *Scheduler) untrackTask(taskID string) {
	s.mu.Lock()
	delete(s.taskCancels, taskID)
	s.mu.Unlock()
}
