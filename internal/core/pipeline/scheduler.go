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

	mu            sync.Mutex
	taskCancels   map[string]context.CancelFunc
	pausedTasks   map[string]bool
	deletingTasks map[string]bool
	taskDone      map[string]chan struct{}
}

// NewScheduler 创建调度器
func NewScheduler(core *Core, onDone func(taskID string, err error)) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		core:          core,
		workerNum:     defaultWorkerCount,
		jobs:          make(chan Job, 100),
		ctx:           ctx,
		cancel:        cancel,
		onDone:        onDone,
		taskCancels:   make(map[string]context.CancelFunc),
		pausedTasks:   make(map[string]bool),
		deletingTasks: make(map[string]bool),
		taskDone:      make(map[string]chan struct{}),
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
	s.mu.Lock()
	deleting := s.deletingTasks[job.TaskID]
	s.mu.Unlock()
	if deleting {
		return fmt.Errorf("任务正在删除或已删除")
	}
	select {
	case <-s.ctx.Done():
		return fmt.Errorf("调度器已停止")
	case s.jobs <- job:
		slog.Info("job submitted", "task_id", job.TaskID, "input", job.InputPath)
		return nil
	}
}

// CancelAndWait 阻止排队任务启动，并等待运行任务及完成回调退出后再允许清理文件。
func (s *Scheduler) CancelAndWait(ctx context.Context, taskID string) error {
	s.mu.Lock()
	s.deletingTasks[taskID] = true
	done := s.taskDone[taskID]
	if cancel := s.taskCancels[taskID]; cancel != nil {
		s.pausedTasks[taskID] = true
		cancel()
	}
	s.mu.Unlock()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Notifier 获取调度器核心绑定的通知器
func (s *Scheduler) Notifier() Notifier {
	if s.core != nil {
		return s.core.Notifier()
	}
	return nil
}

// SetTTSConfig 将保存的配音端点用于后续合成请求。
func (s *Scheduler) SetTTSConfig(engine, voice, baseURL, apiKey, model string) {
	s.core.SetTTSConfig(engine, voice, baseURL, apiKey, model)
}

// SetTranslationClient 切换新任务使用的翻译客户端与语义分句状态。
func (s *Scheduler) SetTranslationClient(client LLMClient, splitReady bool) {
	s.core.SetTranslationClient(client, splitReady)
	if notifier, ok := s.core.Notifier().(interface{ SetSemanticSplitReady(bool) }); ok {
		notifier.SetSemanticSplitReady(splitReady)
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
		if !s.trackTask(job.TaskID, cancel) {
			cancel()
			continue
		}

		slog.Info("worker processing", "worker", id, "task_id", job.TaskID)
		err := s.core.Run(taskCtx, job)

		cancel()

		if s.onDone != nil {
			s.onDone(job.TaskID, err)
		}
		s.untrackTask(job.TaskID)
	}

	slog.Info("worker stopped", "id", id)
}

// trackTask 在同一把锁内检查删除标记与注册执行状态，消除出队和删除之间的竞态。
func (s *Scheduler) trackTask(taskID string, cancel context.CancelFunc) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deletingTasks[taskID] || s.taskDone[taskID] != nil {
		return false
	}
	s.taskCancels[taskID] = cancel
	s.taskDone[taskID] = make(chan struct{})
	return true
}

// untrackTask 在所有写文件和完成回调结束后通知删除操作继续。
func (s *Scheduler) untrackTask(taskID string) {
	s.mu.Lock()
	delete(s.taskCancels, taskID)
	if done := s.taskDone[taskID]; done != nil {
		close(done)
	}
	delete(s.taskDone, taskID)
	s.mu.Unlock()
}
