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
  Film,
} from 'lucide-react';
import { api } from '../lib/api';
import { RefreshButton } from '../components/RefreshButton';
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
          <span className="flex items-center space-x-1.5 text-blue-600 bg-blue-50 dark:bg-blue-500/15 dark:text-blue-400 px-2.5 py-0.5 rounded-full text-[11px] font-semibold border border-blue-200 dark:border-blue-500/30">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-600 dark:bg-blue-400 animate-pulse" />
            <span>处理中</span>
          </span>
        );
      case 2:
        return (
          <span className="flex items-center space-x-1.5 text-amber-600 bg-amber-50 dark:bg-amber-500/15 dark:text-amber-400 px-2.5 py-0.5 rounded-full text-[11px] font-semibold border border-amber-200 dark:border-amber-500/30">
            <Pause size={10} />
            <span>已暂停</span>
          </span>
        );
      case 3:
        return (
          <span className="flex items-center space-x-1.5 text-emerald-600 bg-emerald-50 dark:bg-emerald-500/15 dark:text-emerald-400 px-2.5 py-0.5 rounded-full text-[11px] font-semibold border border-emerald-200 dark:border-emerald-500/30">
            <CheckCircle2 size={11} />
            <span>已完成</span>
          </span>
        );
      case 4:
        return (
          <span className="flex items-center space-x-1.5 text-rose-600 bg-rose-50 dark:bg-rose-500/15 dark:text-rose-400 px-2.5 py-0.5 rounded-full text-[11px] font-semibold border border-rose-200 dark:border-rose-500/30">
            <AlertTriangle size={11} />
            <span>失败</span>
          </span>
        );
      default:
        return (
          <span className="flex items-center space-x-1.5 text-slate-500 bg-slate-100 dark:bg-white/10 px-2.5 py-0.5 rounded-full text-[11px] border border-slate-200 dark:border-white/10">
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
    { key: 'merge', label: '音视频混音' },
    { key: 'burn', label: '字幕压制' },
  ];

  const getStepStatus = (stepKey: string, task: Task) => {
    if (task.status === 3) return 'done';
    if (task.current_step === stepKey && task.status === 1) return 'current';
    const stepOrder = ['whisper', 'split', 'translate', 'tts', 'merge', 'burn'];
    const currentIndex = stepOrder.indexOf(task.current_step || '');
    const thisIndex = stepOrder.indexOf(stepKey);
    if (currentIndex > thisIndex) return 'done';
    return 'pending';
  };

  return (
    <div className="h-screen flex flex-col overflow-y-auto px-8 py-6 select-none">
      <div className="max-w-5xl w-full mx-auto space-y-6 pt-4 pb-12">
        {/* 顶部标题与操作 */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
              任务流转看板
            </h2>
          </div>
          <RefreshButton
            onRefresh={loadTasks}
            title="刷新列表"
            size={13}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-xl border border-slate-200 dark:border-white/10 hover:bg-slate-100 dark:hover:bg-white/5 text-xs text-slate-600 dark:text-slate-300 transition-colors"
          >
            <span>刷新</span>
          </RefreshButton>
        </div>

        {/* 状态统计卡片 */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {[
            { label: '全部任务', count: tasks.length, color: 'text-slate-800 dark:text-white' },
            { label: '处理中', count: tasks.filter((t) => t.status === 1).length, color: 'text-blue-600 dark:text-blue-400' },
            { label: '已完成', count: tasks.filter((t) => t.status === 3).length, color: 'text-emerald-600 dark:text-emerald-400' },
            { label: '异常/失败', count: tasks.filter((t) => t.status === 4).length, color: 'text-rose-600 dark:text-rose-400' },
          ].map((item, i) => (
            <div
              key={i}
              className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-4"
            >
              <span className="text-[11px] font-semibold text-slate-400 block">{item.label}</span>
              <span className={`text-2xl font-bold font-mono mt-1 block ${item.color}`}>
                {item.count}
              </span>
            </div>
          ))}
        </div>

        {/* 任务列表主体 */}
        {loading ? (
          <div className="p-16 text-center text-xs text-slate-400">正在载入流水线任务...</div>
        ) : tasks.length === 0 ? (
          <div className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-16 text-center space-y-3">
            <div className="w-12 h-12 rounded-2xl bg-slate-100 dark:bg-white/5 text-slate-400 flex items-center justify-center mx-auto">
              <Sparkles size={20} />
            </div>
            <p className="text-sm font-semibold text-slate-800 dark:text-white">
              暂无正在进行或历史任务
            </p>
            <p className="text-xs text-slate-400 max-w-sm mx-auto">
              前往「工作台」或「字幕合成」提交视频，即可在此处观测全流程流水线执行细节
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {tasks.map((task) => {
              const isExpanded = activeLogTaskId === task.id;
              const fileName = task.original_name || (task.input_path ? task.input_path.split(/[\\/]/).pop() : '未命名任务视频');

              return (
                <div
                  key={task.id}
                  className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-5 space-y-4 transition-all"
                >
                  {/* 头部：文件名、状态、操作按钮 */}
                  <div className="flex items-start justify-between gap-4">
                    <div className="space-y-1.5 min-w-0">
                      <div className="flex items-center space-x-2.5 truncate">
                        <div className="w-7 h-7 rounded-lg bg-blue-600/10 text-blue-600 dark:bg-blue-500/15 dark:text-blue-400 flex items-center justify-center shrink-0">
                          <Film size={15} />
                        </div>
                        <span className="text-sm font-bold text-slate-800 dark:text-white truncate">
                          {fileName}
                        </span>
                        {statusBadge(task.status)}
                      </div>
                      <p className="text-[11px] font-mono text-slate-400 truncate pl-9.5">
                        输出：{task.batch_dir || task.output_dir}
                      </p>
                    </div>

                    {/* 操作按钮组 */}
                    <div className="flex items-center space-x-2 shrink-0">
                      <button
                        onClick={() => handleOpenDir(task.batch_dir || task.output_dir)}
                        className="p-2 rounded-xl border border-slate-200 dark:border-white/10 hover:bg-slate-100 dark:hover:bg-white/5 text-slate-500 hover:text-slate-800 dark:hover:text-white text-xs transition-colors"
                        title="在系统访达/资源管理器中打开输出目录"
                      >
                        <FolderOpen size={14} />
                      </button>

                      {task.status === 1 && (
                        <button
                          onClick={() => handlePause(task.id)}
                          className="p-2 rounded-xl border border-amber-200 dark:border-amber-500/30 hover:bg-amber-50 dark:hover:bg-amber-500/10 text-amber-600 dark:text-amber-400 text-xs transition-colors"
                          title="暂停任务"
                        >
                          <Pause size={14} />
                        </button>
                      )}

                      {task.status === 2 && (
                        <button
                          onClick={() => handleResume(task.id)}
                          className="p-2 rounded-xl border border-blue-200 dark:border-blue-500/30 hover:bg-blue-50 dark:hover:bg-blue-500/10 text-blue-600 dark:text-blue-400 text-xs transition-colors"
                          title="继续执行任务"
                        >
                          <Play size={14} />
                        </button>
                      )}

                      <button
                        onClick={() => toggleLogs(task.id)}
                        className={`p-2 rounded-xl border text-xs flex items-center space-x-1 transition-colors ${
                          isExpanded
                            ? 'bg-blue-600 text-white border-blue-600'
                            : 'border-slate-200 dark:border-white/10 hover:bg-slate-100 dark:hover:bg-white/5 text-slate-600 dark:text-slate-300'
                        }`}
                        title="查看/折叠底层执行日志"
                      >
                        <Terminal size={14} />
                        {isExpanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                      </button>

                      <button
                        onClick={() => handleDelete(task.id)}
                        className="p-2 rounded-xl border border-rose-200 dark:border-rose-500/30 hover:bg-rose-50 dark:hover:bg-rose-500/10 text-rose-600 dark:text-rose-400 text-xs transition-colors"
                        title="删除任务"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>

                  {/* 进度条与当前步骤 */}
                  <div className="space-y-2 pt-1">
                    <div className="flex justify-between items-center text-xs">
                      <span className="text-slate-500 dark:text-slate-400">
                        当前步骤：
                        <strong className="text-slate-800 dark:text-slate-100 ml-1 font-bold">
                          {task.current_step ? task.current_step : '排队准备中'}
                          {task.current_detail && ` · ${task.current_detail}`}
                        </strong>
                      </span>
                      <span className="font-mono font-bold text-blue-600 dark:text-blue-400 text-xs">
                        {task.progress}%
                      </span>
                    </div>

                    <div className="w-full h-2 bg-slate-100 dark:bg-white/10 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all duration-300 ${
                          task.status === 4
                            ? 'bg-rose-500'
                            : task.status === 3
                            ? 'bg-emerald-500'
                            : 'bg-blue-600'
                        }`}
                        style={{ width: `${Math.min(100, Math.max(0, task.progress))}%` }}
                      />
                    </div>
                  </div>

                  {/* 步骤流转状态条 */}
                  <div className="grid grid-cols-6 gap-2 pt-3 border-t border-slate-100 dark:border-white/5">
                    {stepsList.map((step, idx) => {
                      const status = getStepStatus(step.key, task);
                      return (
                        <div
                          key={idx}
                          className={`text-center py-2 px-1 rounded-xl text-[11px] font-semibold border transition-all ${
                            status === 'current'
                              ? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300'
                              : status === 'done'
                              ? 'border-emerald-200 bg-emerald-50/60 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20'
                              : 'border-transparent text-slate-400 bg-slate-50 dark:bg-white/[0.03]'
                          }`}
                        >
                          {step.label}
                        </div>
                      );
                    })}
                  </div>

                  {/* 展开的日志终端卡片 */}
                  {isExpanded && (
                    <div className="mt-4 rounded-2xl bg-[#0D1117] text-slate-200 p-4 font-mono text-xs space-y-2 border border-slate-800 animate-in fade-in">
                      <div className="flex items-center justify-between pb-2 border-b border-slate-800 text-[11px] text-slate-400">
                        <div className="flex items-center space-x-2">
                          <span className="w-2.5 h-2.5 rounded-full bg-rose-500 inline-block" />
                          <span className="w-2.5 h-2.5 rounded-full bg-amber-500 inline-block" />
                          <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 inline-block" />
                          <span className="ml-2 font-sans font-bold text-slate-300">执行终端日志流</span>
                        </div>
                        <span>共 {logs.length} 行输出</span>
                      </div>

                      <div className="max-h-60 overflow-y-auto space-y-1 pr-2 select-text">
                        {logs.length === 0 ? (
                          <div className="text-slate-500 italic py-2">暂无流水线详细日志输出...</div>
                        ) : (
                          logs.map((log) => (
                            <div key={log.id} className="leading-relaxed flex items-start space-x-2">
                              <span className="text-slate-500 shrink-0 text-[10px]">
                                {new Date(log.created_at).toLocaleTimeString()}
                              </span>
                              <span
                                className={`uppercase text-[10px] font-bold px-1 rounded shrink-0 ${
                                  log.level === 'error'
                                    ? 'bg-rose-500/20 text-rose-400'
                                    : log.level === 'warn'
                                    ? 'bg-amber-500/20 text-amber-400'
                                    : log.level === 'success'
                                    ? 'bg-emerald-500/20 text-emerald-400'
                                    : 'bg-blue-500/20 text-blue-400'
                                }`}
                              >
                                {log.level}
                              </span>
                              {log.step && (
                                <span className="text-purple-400 shrink-0 text-[10px]">[{log.step}]</span>
                              )}
                              <span className="text-slate-300 break-all">{log.message}</span>
                            </div>
                          ))
                        )}
                        <div ref={logEndRef} />
                      </div>
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
