package pipeline

import (
	"context"
	"fmt"
	ttsadapter "github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/conf"
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
	changed   chan struct{}
	startOnce sync.Once

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
		changed:       make(chan struct{}, 1),
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

// SetWorkers 即时调整上限，不中断已运行的任务。
func (s *Scheduler) SetWorkers(n int) error {
	if err := conf.ValidateWorkers(n); err != nil {
		return err
	}
	s.mu.Lock()
	s.workerNum = n
	s.mu.Unlock()
	s.wake()
	return nil
}

func (s *Scheduler) wake() {
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// Start 启动单一派发器，按入队顺序分配可用任务名额。
func (s *Scheduler) Start() {
	s.startOnce.Do(func() { s.wg.Go(s.dispatch) })
}

// Stop 取消运行与排队任务，等待完成回调退出；可重复调用。
func (s *Scheduler) Stop() {
	s.cancel()
	s.wake()
	s.wg.Wait()
	slog.Info("scheduler stopped")
}

func (s *Scheduler) dispatch() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case job := <-s.jobs:
			if !s.dispatchJob(job) {
				return
			}
		}
	}
}

func (s *Scheduler) dispatchJob(job Job) bool {
	for {
		s.mu.Lock()
		if s.ctx.Err() != nil {
			s.mu.Unlock()
			return false
		}
		if s.deletingTasks[job.TaskID] || s.taskDone[job.TaskID] != nil {
			s.mu.Unlock()
			return true
		}
		if len(s.taskDone) < s.workerNum {
			ctx, cancel := context.WithCancel(s.ctx)
			s.taskCancels[job.TaskID] = cancel
			s.taskDone[job.TaskID] = make(chan struct{})
			s.mu.Unlock()
			s.wg.Go(func() { s.runJob(ctx, cancel, job) })
			return true
		}
		s.mu.Unlock()
		select {
		case <-s.ctx.Done():
			return false
		case <-s.changed:
		}
	}
}

// Submit 提交任务到队列
func (s *Scheduler) Submit(job Job) error {
	s.mu.Lock()
	deleting := s.deletingTasks[job.TaskID]
	s.mu.Unlock()
	if s.ctx.Err() != nil {
		return fmt.Errorf("调度器已停止")
	}
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
	s.wake()
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
func (s *Scheduler) SetTTSConfig(engine, voice, baseURL, apiKey, model string, options ...ttsadapter.SpeechOptions) {
	s.core.SetTTSConfig(engine, voice, baseURL, apiKey, model, options...)
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

// runJob 将完成回调纳入运行名额及删除等待范围。
func (s *Scheduler) runJob(ctx context.Context, cancel context.CancelFunc, job Job) {
	defer s.untrackTask(job.TaskID)
	defer cancel()
	err := s.core.Run(ctx, job)
	if s.onDone != nil {
		s.onDone(job.TaskID, err)
	}
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
	s.wake()
}
