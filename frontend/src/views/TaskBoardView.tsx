import React, { useEffect, useState, useCallback, useRef } from 'react';
import {
  FolderOpen,
  Pause,
  Play,
  Trash2,
  Terminal,
  CheckCircle2,
  Clock,
  AlertTriangle,
  RotateCcw,
  Sparkles,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import { api } from '../lib/api';
import { Task, TaskLog, TaskStatus } from '../types';

export const TaskBoardView: React.FC = () => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeLogTaskId, setActiveLogTaskId] = useState<string | null>(null);
  const [logs, setLogs] = useState<TaskLog[]>([]);
  const logEndRef = useRef<HTMLDivElement>(null);

  const loadTasks = useCallback(async () => {
    try {
      const data = await api.listTasks();
      setTasks(data || []);
    } catch (err) {
      console.error('Failed to load tasks:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTasks();
    const timer = setInterval(loadTasks, 3000);

    // 订阅 Wails 3 事件总线
    const unbindProgress = api.onEvent('task_progress', (data: unknown) => {
      const p = data as { task_id: string; total_progress?: number; progress?: number; step?: string; detail?: string };
      setTasks((prev) =>
        prev.map((t) =>
          t.id === p.task_id
            ? {
                ...t,
                progress: p.total_progress ?? p.progress ?? t.progress,
                current_step: p.step || t.current_step,
                current_detail: p.detail || t.current_detail,
              }
            : t
        )
      );
    });

    const unbindStepStart = api.onEvent('task_step_started', (data: unknown) => {
      const s = data as { task_id: string; step: string; detail: string };
      setTasks((prev) =>
        prev.map((t) =>
          t.id === s.task_id
            ? { ...t, current_step: s.step, current_detail: s.detail, status: 1 }
            : t
        )
      );
    });

    const unbindPaused = api.onEvent('task_paused', (data: unknown) => {
      const p = data as { task_id: string };
      setTasks((prev) =>
        prev.map((t) => (t.id === p.task_id ? { ...t, status: 2 } : t))
      );
    });

    return () => {
      clearInterval(timer);
      unbindProgress();
      unbindStepStart();
      unbindPaused();
    };
  }, [loadTasks]);

  // 打开日志终端
  const toggleLogs = async (taskId: string) => {
    if (activeLogTaskId === taskId) {
      setActiveLogTaskId(null);
      return;
    }
    setActiveLogTaskId(taskId);
    try {
      const taskLogs = await api.listTaskLogs(taskId);
      setLogs(taskLogs || []);
    } catch (err) {
      console.error('Failed to load logs:', err);
    }
  };

  useEffect(() => {
    if (activeLogTaskId) {
      logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, activeLogTaskId]);

  const handlePause = async (id: string) => {
    await api.pauseTask(id);
    loadTasks();
  };

  const handleResume = async (id: string) => {
    await api.resumeTask(id);
    loadTasks();
  };

  const handleDelete = async (id: string) => {
    if (confirm('确定要删除此任务吗？')) {
      await api.deleteTask(id);
      loadTasks();
    }
  };

  const handleOpenDir = (dir: string) => {
    api.openInFileManager(dir);
  };

  const statusBadge = (status: TaskStatus) => {
    switch (status) {
      case 1:
        return (
          <span className="flex items-center space-x-1 text-apple-accent bg-apple-accent/10 px-2 py-0.5 rounded-full text-[11px] font-semibold">
            <span className="w-1.5 h-1.5 rounded-full bg-apple-accent animate-pulse" />
            <span>进行中</span>
          </span>
        );
      case 2:
        return (
          <span className="flex items-center space-x-1 text-amber-500 bg-amber-500/10 px-2 py-0.5 rounded-full text-[11px] font-semibold">
            <Pause size={10} />
            <span>已暂停</span>
          </span>
        );
      case 3:
        return (
          <span className="flex items-center space-x-1 text-emerald-500 bg-emerald-500/10 px-2 py-0.5 rounded-full text-[11px] font-semibold">
            <CheckCircle2 size={11} />
            <span>已完成</span>
          </span>
        );
      case 4:
        return (
          <span className="flex items-center space-x-1 text-red-500 bg-red-500/10 px-2 py-0.5 rounded-full text-[11px] font-semibold">
            <AlertTriangle size={11} />
            <span>执行失败</span>
          </span>
        );
      default:
        return (
          <span className="flex items-center space-x-1 text-apple-muted bg-black/5 dark:bg-white/5 px-2 py-0.5 rounded-full text-[11px]">
            <Clock size={11} />
            <span>等待中</span>
          </span>
        );
    }
  };

  const stepsList = [
    { key: 'whisper', label: '语音识别' },
    { key: 'split', label: '语义分句' },
    { key: 'translate', label: '智能翻译' },
    { key: 'tts', label: '语音合成' },
    { key: 'merge', label: '音视频合并' },
    { key: 'burn', label: '字幕压制' },
  ];

  return (
    <div className="h-screen flex flex-col overflow-y-auto px-8 py-6">
      <div className="max-w-5xl w-full mx-auto space-y-6">
        {/* 顶部标题与快速统计 */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold tracking-tight text-apple-text dark:text-apple-darkText">
              任务流转看板
            </h2>
            <p className="text-xs text-apple-muted mt-1">
              实时追踪任务状态、步骤分支进度与底层执行日志
            </p>
          </div>
          <button
            onClick={loadTasks}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg border border-apple-border dark:border-apple-darkBorder hover:bg-black/5 dark:hover:bg-white/5 text-xs text-apple-muted transition-colors"
          >
            <RotateCcw size={13} />
            <span>刷新</span>
          </button>
        </div>

        {/* 状态统计卡片 */}
        <div className="grid grid-cols-4 gap-3">
          {[
            { label: '全部任务', count: tasks.length, color: 'text-apple-text dark:text-apple-darkText' },
            { label: '处理中', count: tasks.filter((t) => t.status === 1).length, color: 'text-apple-accent' },
            { label: '已完成', count: tasks.filter((t) => t.status === 3).length, color: 'text-emerald-500' },
            { label: '异常/失败', count: tasks.filter((t) => t.status === 4).length, color: 'text-red-500' },
          ].map((item, i) => (
            <div
              key={i}
              className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-3.5 shadow-sm"
            >
              <span className="text-xs text-apple-muted block">{item.label}</span>
              <span className={`text-2xl font-bold font-mono mt-0.5 block ${item.color}`}>
                {item.count}
              </span>
            </div>
          ))}
        </div>

        {/* 任务列表 */}
        {loading ? (
          <div className="p-12 text-center text-xs text-apple-muted">正在加载任务...</div>
        ) : tasks.length === 0 ? (
          <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-2xl p-12 text-center space-y-3">
            <div className="w-12 h-12 rounded-full bg-black/5 dark:bg-white/5 text-apple-muted flex items-center justify-center mx-auto">
              <Sparkles size={20} />
            </div>
            <p className="text-sm font-semibold text-apple-text dark:text-apple-darkText">
              暂无正在进行或历史任务
            </p>
            <p className="text-xs text-apple-muted max-w-sm mx-auto">
              切换到「工作台」拖入视频并点击处理，即可在此处实时观测全流程进度
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {tasks.map((task) => {
              const isExpanded = activeLogTaskId === task.id;
              const fileName = task.input_path ? task.input_path.split(/[\\/]/).pop() : '未命名视频';

              return (
                <div
                  key={task.id}
                  className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-5 shadow-sm space-y-4 transition-all"
                >
                  {/* 头部：文件名、状态、操作按钮 */}
                  <div className="flex items-start justify-between">
                    <div className="space-y-1 max-w-xl">
                      <div className="flex items-center space-x-2">
                        <span className="text-sm font-bold text-apple-text dark:text-apple-darkText truncate">
                          {fileName}
                        </span>
                        {statusBadge(task.status)}
                      </div>
                      <p className="text-[11px] font-mono text-apple-muted truncate">
                        输出：{task.output_dir}
                      </p>
                    </div>

                    {/* 操作按钮组 */}
                    <div className="flex items-center space-x-2">
                      <button
                        onClick={() => handleOpenDir(task.output_dir)}
                        className="p-1.5 rounded-lg border border-apple-border dark:border-apple-darkBorder hover:bg-black/5 dark:hover:bg-white/5 text-apple-muted hover:text-apple-text dark:hover:text-apple-darkText text-xs flex items-center space-x-1"
                        title="在系统文件管理器中打开"
                      >
                        <FolderOpen size={14} />
                      </button>

                      {task.status === 1 && (
                        <button
                          onClick={() => handlePause(task.id)}
                          className="p-1.5 rounded-lg border border-apple-border dark:border-apple-darkBorder hover:bg-amber-500/10 text-amber-500 text-xs"
                          title="暂停"
                        >
                          <Pause size={14} />
                        </button>
                      )}

                      {task.status === 2 && (
                        <button
                          onClick={() => handleResume(task.id)}
                          className="p-1.5 rounded-lg border border-apple-border dark:border-apple-darkBorder hover:bg-apple-accent/10 text-apple-accent text-xs"
                          title="继续"
                        >
                          <Play size={14} />
                        </button>
                      )}

                      <button
                        onClick={() => toggleLogs(task.id)}
                        className={`p-1.5 rounded-lg border text-xs flex items-center space-x-1 transition-colors ${
                          isExpanded
                            ? 'bg-apple-accent text-white border-apple-accent'
                            : 'border-apple-border dark:border-apple-darkBorder hover:bg-black/5 dark:hover:bg-white/5 text-apple-muted'
                        }`}
                        title="查看日志"
                      >
                        <Terminal size={14} />
                        {isExpanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                      </button>

                      <button
                        onClick={() => handleDelete(task.id)}
                        className="p-1.5 rounded-lg border border-apple-border dark:border-apple-darkBorder hover:bg-red-500/10 text-red-500 text-xs"
                        title="删除任务"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>

                  {/* 进度条与当前步骤 */}
                  <div className="space-y-1.5">
                    <div className="flex justify-between items-center text-xs">
                      <span className="text-apple-muted">
                        当前步骤：
                        <strong className="text-apple-text dark:text-apple-darkText ml-1 font-medium">
                          {task.current_step ? task.current_step : '排队准备中'}
                          {task.current_detail && ` · ${task.current_detail}`}
                        </strong>
                      </span>
                      <span className="font-mono font-bold text-apple-accent">{task.progress}%</span>
                    </div>

                    <div className="w-full h-2 bg-black/5 dark:bg-white/5 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all duration-300 ${
                          task.status === 4
                            ? 'bg-red-500'
                            : task.status === 3
                            ? 'bg-emerald-500'
                            : 'bg-apple-accent'
                        }`}
                        style={{ width: `${Math.min(100, Math.max(0, task.progress))}%` }}
                      />
                    </div>
                  </div>

                  {/* 步骤条目状态 */}
                  <div className="grid grid-cols-6 gap-2 pt-2 border-t border-apple-border dark:border-apple-darkBorder">
                    {stepsList.map((step, idx) => {
                      const isCurrent = task.current_step === step.key && task.status === 1;
                      const isDone = task.status === 3;
                      return (
                        <div
                          key={idx}
                          className={`text-center py-1.5 px-1 rounded-lg text-[10px] font-medium border ${
                            isCurrent
                              ? 'border-apple-accent bg-apple-accent/10 text-apple-accent'
                              : isDone
                              ? 'border-emerald-500/30 text-emerald-500 bg-emerald-500/5'
                              : 'border-transparent text-apple-muted bg-black/5 dark:bg-white/5'
                          }`}
                        >
                          {step.label}
                        </div>
                      );
                    })}
                  </div>

                  {/* 展开的日志终端卡片 */}
                  {isExpanded && (
                    <div className="mt-3 bg-black/90 dark:bg-black/95 text-emerald-400 font-mono text-[11px] p-3.5 rounded-xl max-h-56 overflow-y-auto space-y-1 border border-white/10 shadow-inner">
                      <div className="text-white/40 pb-1 border-b border-white/10 flex justify-between text-[10px]">
                        <span>终端输出 (任务 ID: {task.id})</span>
                        <span>{logs.length} 条记录</span>
                      </div>
                      {logs.length === 0 ? (
                        <div className="text-white/30 py-2">暂无详细日志或正在初始化...</div>
                      ) : (
                        logs.map((log, i) => (
                          <div key={i} className="leading-relaxed flex items-start space-x-2">
                            <span className="text-white/30 text-[10px] shrink-0">
                              {log.created_at ? log.created_at.split('T')[1]?.slice(0, 8) : ''}
                            </span>
                            <span
                              className={
                                log.level === 'error'
                                  ? 'text-red-400'
                                  : log.level === 'warn'
                                  ? 'text-amber-400'
                                  : log.level === 'success'
                                  ? 'text-emerald-400'
                                  : 'text-white/80'
                              }
                            >
                              {log.message}
                            </span>
                          </div>
                        ))
                      )}
                      <div ref={logEndRef} />
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
