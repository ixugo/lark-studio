import React, { useEffect, useState, useCallback, useRef } from 'react';
import {
  FolderOpen,
  Pause,
  Play,
  Trash2,
  Terminal,
  Clock,
  AlertTriangle,
  AlertCircle,
  X,
  RotateCcw,
  Sparkles,
  ChevronDown,
  ChevronUp,
  Film,
  Sliders,
} from 'lucide-react';
import { api } from '../lib/api';
import { RefreshButton } from '../components/RefreshButton';
import { Task, TaskLog, TaskStatus, RerunTaskOptions, RecipeItem } from '../types';
import { useTranslation } from '../i18n';
import { detectResourceType, recipeUnavailableReason, resolveWorkflowMode, subtitleContent, currentTaskRecipe, rerunRecipeCatalog } from '../lib/workflowRecipes';

import { taskBoardStepStatus, taskBoardActiveSteps, resetTaskStepStates, TASK_STEP_ORDER, type StepStates } from '../lib/taskBoardSteps';

// ─── 调整配方并重跑弹窗 ─────────────────────────────────────────
interface RerunRecipeModalProps {
  task: Task;
  onClose: () => void;
  onSubmit: (opts: RerunTaskOptions) => Promise<void>;
}

const RerunRecipeModal: React.FC<RerunRecipeModalProps> = ({ task, onClose, onSubmit }) => {
  const { t } = useTranslation();
  const currentRecipe = currentTaskRecipe(task, t);
  const [recipes, setRecipes] = useState<RecipeItem[]>(() => rerunRecipeCatalog(task, [], t));
  const [selectedRecipeId, setSelectedRecipeId] = useState(() => rerunRecipeCatalog(task, [], t).find(recipe => recipe.title === currentRecipe.title)?.id || currentRecipe.id);
  const resource = detectResourceType(task.input_path);
  const [recipesLoaded, setRecipesLoaded] = useState(false);
  const [recipeLoadFailed, setRecipeLoadFailed] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const selectedRecipe = recipes.find((recipe) => recipe.id === selectedRecipeId);

  useEffect(() => {
    api.listRecipes().then((items) => {
      const catalog = rerunRecipeCatalog(task, items, t);
      setRecipes(catalog);
      setSelectedRecipeId(previous => previous === 'task-recipe' ? catalog.find(recipe => recipe.title === currentTaskRecipe(task, t).title)?.id || previous : previous);
      setRecipesLoaded(true);
    }).catch((error) => {
      console.error('加载重跑配方失败:', error);
      setRecipeLoadFailed(true);
      setRecipesLoaded(true);
    });
  }, [task.id, t]);

  const handleRerun = async (stepKey: string) => {
    if (!selectedRecipe || recipeUnavailableReason(selectedRecipe, resource)) return;
    setSubmitting(true);
    try {
      const mode = resolveWorkflowMode(selectedRecipe);
      await onSubmit({
        from_step: stepKey,
        mode,
        source_lang: selectedRecipe.source_lang || task.source_lang,
        target_lang: selectedRecipe.target_lang || task.target_lang,
        translator: selectedRecipe.translate_service === 'local' ? 'openai' : selectedRecipe.translate_service || task.translator,
        output_content: subtitleContent(selectedRecipe.do_translate),
        tts_engine: selectedRecipe.tts_engine || task.tts_engine,
        tts_voice: selectedRecipe.tts_voice || task.tts_voice,
        speech_rate: selectedRecipe.speech_rate || task.speech_rate,
        subtitle_output: selectedRecipe.do_video ? selectedRecipe.subtitle_output || task.subtitle_output : 'file',
        recipe_name: selectedRecipe.title,
      });
      onClose();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-in fade-in">
      <div className="bg-white dark:bg-[#1E1E20] border border-slate-200 dark:border-[#2C2C2E] rounded-2xl shadow-2xl max-w-lg w-full p-6 space-y-5 animate-in zoom-in-95">
        <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-white/10">
          <div className="flex items-center space-x-2">
            <Sliders className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <h3 className="text-base font-bold text-slate-800 dark:text-white">
              {t('rerunModal.title', '调整配方与重跑任务')}
            </h3>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-white/5"
          >
            <X size={16} />
          </button>
        </div>

        <p className="text-xs text-slate-500 dark:text-slate-400">
          {t('rerunModal.subtitle', '选择系统或用户配方，按配方配置重跑任务并指定起始节点。')}
        </p>

        <div className="space-y-2 text-xs">
          <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400">
            {t('rerunModal.recipeLabel', '执行配方')}
          </label>
          <select
            value={selectedRecipeId}
            onChange={(event) => setSelectedRecipeId(event.target.value)}
            disabled={submitting}
            className="w-full h-9 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-slate-800 dark:text-white"
          >
            {recipes.map((recipe) => {
              const unavailable = recipeUnavailableReason(recipe, resource);
              return <option key={recipe.id} value={recipe.id} disabled={!!unavailable}>{recipe.title}{unavailable ? ` — ${t('rerunModal.incompatible', '不适用当前资源')}` : ''}</option>;
            })}
          </select>
          <p className="text-slate-500 dark:text-slate-400">
            {selectedRecipe?.subtitle}
            {!recipesLoaded && ` ${t('rerunModal.loadingRecipes', '正在加载用户配方…')}`}
            {recipeLoadFailed && ` ${t('rerunModal.recipeLoadFailed', '用户配方读取失败，仍可选择系统配方。')}`}
            {selectedRecipe && recipeUnavailableReason(selectedRecipe, resource)}
          </p>
        </div>

        {/* 起始节点选择按钮组 */}
        <div className="pt-2 border-t border-slate-100 dark:border-white/10 space-y-2">
          <div className="text-[11px] font-bold text-slate-500 uppercase tracking-wider">
            {t('rerunModal.selectStepTitle', '选择重跑起始节点 (将自动清理该节点及后续旧产物)')}
          </div>
          <div className="grid grid-cols-1 gap-2">
            {selectedRecipe?.do_sub && (
              <button
                disabled={submitting || !!(selectedRecipe && recipeUnavailableReason(selectedRecipe, resource))}
                onClick={() => handleRerun('whisper')}
                className="w-full px-3 py-2 text-left rounded-xl border border-slate-200 dark:border-white/10 hover:border-blue-500 hover:bg-blue-50/50 dark:hover:bg-blue-500/10 text-xs font-semibold text-slate-800 dark:text-slate-100 flex items-center justify-between transition-colors"
              >
                <span>{t('rerunModal.stepWhisperBtn', '从「语音识别」全量重跑')}</span>
                <span className="text-[10px] text-slate-400 font-normal">{t('taskBoard.rerunTips.whisper', '全量从头跑')}</span>
              </button>
            )}
            {selectedRecipe && selectedRecipe.do_translate && (
              <button
                disabled={submitting || !!(selectedRecipe && recipeUnavailableReason(selectedRecipe, resource))}
                onClick={() => handleRerun('translate')}
                className="w-full px-3 py-2 text-left rounded-xl border border-slate-200 dark:border-white/10 hover:border-blue-500 hover:bg-blue-50/50 dark:hover:bg-blue-500/10 text-xs font-semibold text-slate-800 dark:text-slate-100 flex items-center justify-between transition-colors"
              >
                <span>{t('rerunModal.stepTranslateBtn', '从「智能翻译」重跑 (重新翻译与生成声音)')}</span>
                <span className="text-[10px] text-slate-400 font-normal">{t('taskBoard.rerunTips.translate', '重合成声音')}</span>
              </button>
            )}
            {selectedRecipe?.do_dub && !selectedRecipe.do_translate && (
              <button
                disabled={submitting || !!recipeUnavailableReason(selectedRecipe, resource)}
                onClick={() => handleRerun('tts')}
                className="w-full px-3 py-2 text-left rounded-xl border border-slate-200 dark:border-white/10 hover:border-blue-500 hover:bg-blue-50/50 dark:hover:bg-blue-500/10 text-xs font-semibold text-slate-800 dark:text-slate-100"
              >
                {t('rerunModal.stepTTSBtn', '从「AI 配音」重跑（重新生成声音）')}
              </button>
            )}
            {selectedRecipe?.do_dub && (
              <button
                disabled={submitting || !!(selectedRecipe && recipeUnavailableReason(selectedRecipe, resource))}
                onClick={() => handleRerun('merge')}
                className="w-full px-3 py-2 text-left rounded-xl border border-slate-200 dark:border-white/10 hover:border-blue-500 hover:bg-blue-50/50 dark:hover:bg-blue-500/10 text-xs font-semibold text-slate-800 dark:text-slate-100 flex items-center justify-between transition-colors"
              >
                <span>{t('rerunModal.stepMergeBtn', '从「音视频混音」重跑 (复用声音，重算时间轴)')}</span>
                <span className="text-[10px] text-slate-400 font-normal">{t('taskBoard.rerunTips.merge', '重算时间轴')}</span>
              </button>
            )}
            {selectedRecipe?.do_video && (
              <button
                disabled={submitting || !selectedRecipe}
                onClick={() => handleRerun('burn')}
                className="w-full px-3 py-2 text-left rounded-xl border border-slate-200 dark:border-white/10 hover:border-blue-500 hover:bg-blue-50/50 dark:hover:bg-blue-500/10 text-xs font-semibold text-slate-800 dark:text-slate-100 flex items-center justify-between transition-colors"
              >
                <span>{t('rerunModal.stepBurnBtn', '从「压制合成」重跑 (复用声音，直接成片)')}</span>
                <span className="text-[10px] text-slate-400 font-normal">{t('taskBoard.rerunTips.burn', '直接压成片')}</span>
              </button>
            )}
          </div>
        </div>

        <div className="flex justify-end pt-2">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-white/5 transition-colors"
          >
            {t('rerunModal.cancel', '取消')}
          </button>
        </div>
      </div>
    </div>
  );
};

export const TaskBoardView: React.FC = () => {
  const { t } = useTranslation();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [taskStepStates, setTaskStepStates] = useState<Record<string, StepStates>>({});
  const [taskStepsMap, setTaskStepsMap] = useState<Record<string, Record<string, number>>>({});
  const [loading, setLoading] = useState(true);
  const [activeLogTaskId, setActiveLogTaskId] = useState<string | null>(null);
  const activeLogTaskIdRef = useRef<string | null>(null);
  activeLogTaskIdRef.current = activeLogTaskId;
  const [logs, setLogs] = useState<TaskLog[]>([]);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [rerunMenuTaskId, setRerunMenuTaskId] = useState<string | null>(null);
  const [recipeModalTask, setRecipeModalTask] = useState<Task | null>(null);
  const [toasts, setToasts] = useState<Array<{ id: string; type: 'error' | 'success'; text: string }>>([]);
  const logContainerRef = useRef<HTMLDivElement>(null);
  const isAutoScrollRef = useRef(true);

  const showToast = useCallback((text: string, type: 'error' | 'success' = 'error') => {
    const id = `${Date.now()}_${Math.random().toString(36).substring(2, 7)}`;
    setToasts((prev) => {
      // 允许往下显示，最多保留最新 3 个
      const next = [...prev, { id, type, text }];
      return next.slice(-3);
    });

    setTimeout(() => {
      setToasts((prev) => prev.filter((item) => item.id !== id));
    }, 4500);
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((item) => item.id !== id));
  }, []);

  const loadTasks = useCallback(async () => {
    try {
      const data = await api.listTasks();
      const newTasks = data || [];
      setTasks((prev) => {
        if (prev.length === 0 && newTasks.length === 0) return prev;
        // 如果长度不同，直接更新
        if (prev.length !== newTasks.length) return newTasks;
        // 比对核心字段，未变化则维持旧引用避免全量重新挂载导致抖动
        const hasChange = newTasks.some((nt, idx) => {
          const pt = prev[idx];
          return (
            !pt ||
            pt.id !== nt.id ||
            pt.status !== nt.status ||
            pt.progress !== nt.progress ||
            pt.current_step !== nt.current_step ||
            pt.current_detail !== nt.current_detail ||
            pt.error !== nt.error ||
 pt.original_name !== nt.original_name ||
 pt.batch_dir !== nt.batch_dir ||
 pt.result_path !== nt.result_path
          );
        });
        return hasChange ? newTasks : prev;
      });

      // 并发异步查询各任务步骤耗时记录
      for (const t of newTasks) {
        api.listTaskSteps(t.id).then((stepList) => {
          setTaskStepStates(previous => ({
            ...previous,
            [t.id]: Object.fromEntries((stepList || []).map(step => [step.name, step.status])),
          }));
          if (!stepList || stepList.length === 0) return;
          const durations: Record<string, number> = {};
          for (const s of stepList) {
            // 严格要求：只有步骤真正完成（有 ended_at）时才呈现耗时，执行中绝不提前显示 1s
            if (s.started_at && s.ended_at) {
              const start = new Date(s.started_at).getTime();
              const end = new Date(s.ended_at).getTime();
              const rawSec = Math.round((end - start) / 1000);
              // 边界防御：过滤历史脏数据或异常跨越时间
              if (rawSec > 0 && rawSec < 7200) {
                durations[s.name] = rawSec;
              }
            }
          }
          if (Object.keys(durations).length > 0) {
            setTaskStepsMap((prev) => ({
              ...prev,
              [t.id]: { ...(prev[t.id] || {}), ...durations },
            }));
          }
        }).catch(() => {});
      }
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
      if (p.task_id && p.step) {
        setTaskStepStates(previous => ({ ...previous, [p.task_id]: { ...previous[p.task_id], [p.step!]: 1 } }));
      }
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
      if (s.task_id && s.step) {
        setTaskStepStates(previous => ({ ...previous, [s.task_id]: { ...previous[s.task_id], [s.step]: 1 } }));
      }
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

    const unbindDone = api.onEvent('task_done', (data: unknown) => {
      const p = data as { task_id: string };
      setTasks((prev) =>
        prev.map((t) => (t.id === p.task_id ? { ...t, status: 3, progress: 100 } : t))
      );
    });

    const unbindFailed = api.onEvent('task_failed', (data: unknown) => {
      const f = data as { task_id: string; error?: string };
      setTasks((prev) =>
        prev.map((t) =>
          t.id === f.task_id
            ? { ...t, status: 4, error: f.error || t.error }
            : t
        )
      );
      if (f.task_id) {
        setActiveLogTaskId(f.task_id);
        api.listTaskLogs(f.task_id).then((l) => setLogs(l || [])).catch(() => {});
      }
    });

    const unbindTaskLog = api.onEvent('task_log', (data: unknown) => {
      const log = data as TaskLog;
      if (log && log.task_id && activeLogTaskIdRef.current === log.task_id) {
        setLogs((prev) => {
          if (prev.some((item) => item.id === log.id)) return prev;
          return [...prev, log];
        });
      }
    });

    const unbindStepDone = api.onEvent('task_step_done', (data: unknown) => {
      const d = data as { task_id: string; step: string; elapsed_ms?: number };
      if (d?.task_id && d?.step) {
        setTaskStepStates(previous => ({ ...previous, [d.task_id]: { ...previous[d.task_id], [d.step]: 2 } }));
      }
      if (d?.task_id && d?.step && typeof d.elapsed_ms === 'number') {
        const sec = Math.max(1, Math.round(d.elapsed_ms / 1000));
        setTaskStepsMap((prev) => ({
          ...prev,
          [d.task_id]: { ...(prev[d.task_id] || {}), [d.step]: sec },
        }));
      }
    });

    return () => {
      clearInterval(timer);
      unbindProgress();
      unbindStepStart();
      unbindStepDone();
      unbindPaused();
      unbindDone();
      unbindFailed();
      unbindTaskLog();
    };
  }, [loadTasks]);

  // 打开日志终端
  const toggleLogs = async (taskId: string) => {
    if (activeLogTaskId === taskId) {
      setActiveLogTaskId(null);
      return;
    }
    setActiveLogTaskId(taskId);
    isAutoScrollRef.current = true;
    try {
      const taskLogs = await api.listTaskLogs(taskId);
      setLogs(taskLogs || []);
    } catch (err) {
      console.error('Failed to load logs:', err);
    }
  };

  // 用户手动上下滚动日志容器检测
  const handleLogScroll = () => {
    const el = logContainerRef.current;
    if (!el) return;
    // 当距离底部不足 40px 时，恢复跟随自动滚动；否则保持用户当前位置
    const isCloseToBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= 40;
    isAutoScrollRef.current = isCloseToBottom;
  };

  useEffect(() => {
    if (!activeLogTaskId) return;
    const el = logContainerRef.current;
    if (!el) return;
    if (isAutoScrollRef.current) {
      el.scrollTo({
        top: el.scrollHeight,
        behavior: 'smooth',
      });
    }
  }, [logs, activeLogTaskId]);

  const handlePause = async (id: string) => {
    setTasks((prev) => prev.map((t) => (t.id === id ? { ...t, status: 2 } : t)));
    await api.pauseTask(id);
    loadTasks();
  };

  const handleResume = async (id: string) => {
    setTasks((prev) => prev.map((t) => (t.id === id ? { ...t, status: 1 } : t)));
    await api.resumeTask(id);
    loadTasks();
  };

  const handleRerunFromStep = async (id: string, stepKey: string, stepLabel: string) => {
    setRerunMenuTaskId(null);
    try {
      showToast(`正在从「${stepLabel}」重跑任务...`, 'success');
      const basePct =
        stepKey === 'whisper' ? 0 : stepKey === 'translate' ? 30 : stepKey === 'merge' ? 70 : 85;
      setTasks((prev) =>
        prev.map((t) =>
          t.id === id
            ? { ...t, status: 1, progress: basePct, current_step: stepKey, current_detail: `重跑「${stepLabel}」`, error: '' }
            : t
        )
      );
      // 清空当前重跑节点及后续步骤的旧耗时记录
      const stepOrder: readonly string[] = TASK_STEP_ORDER;
      const startIndex = stepOrder.indexOf(stepKey);
      setTaskStepStates(previous => ({ ...previous, [id]: resetTaskStepStates(previous[id] || {}, stepKey) }));
      if (startIndex >= 0) {
        setTaskStepsMap((prev) => {
          const current = { ...(prev[id] || {}) };
          for (let i = startIndex; i < stepOrder.length; i++) {
            delete current[stepOrder[i]];
          }
          return { ...prev, [id]: current };
        });
      }
      await api.rerunTaskFromStep(id, stepKey);
      loadTasks();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      showToast(`重跑失败: ${msg}`, 'error');
    }
  };

  const handleRetryTask = async (task: Task) => {
    // 若原先卡在或失败在压制合成(burn)，由于配音时间轴是在混音(merge)生成的，优先回退从音视频混音重跑
    let retryStep = task.current_step || 'whisper';
    if (retryStep === 'burn' && task.mode === 3) {
      retryStep = 'merge';
    }
    const stepLabel = formatStepName(retryStep);
    await handleRerunFromStep(task.id, retryStep, stepLabel);
  };

  const handleRerunWithRecipe = async (opts: RerunTaskOptions) => {
    if (!recipeModalTask) return;
    const id = recipeModalTask.id;
    try {
      showToast(`正在应用新配方并从「${opts.from_step}」重跑...`, 'success');
      const basePct =
        opts.from_step === 'whisper'
          ? 0
          : opts.from_step === 'translate'
          ? 30
          : opts.from_step === 'merge'
          ? 70
          : 85;
      setTasks((prev) =>
        prev.map((t) =>
          t.id === id
            ? {
                ...t,
                status: 1,
                progress: basePct,
                current_step: opts.from_step,
                current_detail: `修改配方重跑「${opts.from_step}」`,
                error: '',
                ...(opts.mode ? { mode: opts.mode } : {}),
                ...(opts.source_lang ? { source_lang: opts.source_lang } : {}),
                ...(opts.speech_rate ? { speech_rate: opts.speech_rate } : {}),
                ...(opts.tts_voice ? { tts_voice: opts.tts_voice } : {}),
                ...(opts.tts_engine ? { tts_engine: opts.tts_engine } : {}),
                ...(opts.subtitle_output ? { subtitle_output: opts.subtitle_output } : {}),
                ...(opts.target_lang ? { target_lang: opts.target_lang } : {}),
                ...(opts.translator ? { translator: opts.translator } : {}),
                ...(opts.recipe_name ? { recipe_name: opts.recipe_name } : {}),
              }
            : t
        )
      );
      // 清空当前重跑节点及后续步骤的旧耗时记录
      const stepOrder: readonly string[] = TASK_STEP_ORDER;
      const startIndex = stepOrder.indexOf(opts.from_step);
      setTaskStepStates(previous => ({ ...previous, [id]: resetTaskStepStates(previous[id] || {}, opts.from_step) }));
      if (startIndex >= 0) {
        setTaskStepsMap((prev) => {
          const current = { ...(prev[id] || {}) };
          for (let i = startIndex; i < stepOrder.length; i++) {
            delete current[stepOrder[i]];
          }
          return { ...prev, [id]: current };
        });
      }
      await api.rerunTaskWithRecipe(id, opts);
      loadTasks();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      showToast(`重跑失败: ${msg}`, 'error');
    }
  };

  const handleDelete = async (id: string) => {
    // 乐观立即移除，绝不依赖易被拦截的 window.confirm
    setTasks((prev) => prev.filter((t) => t.id !== id));
    setDeletingId(null);
    if (activeLogTaskId === id) {
      setActiveLogTaskId(null);
    }
    try {
      await api.deleteTask(id);
    } catch (err) {
      showToast(err instanceof Error ? err.message : '删除任务及产物失败', 'error');
    } finally {
      loadTasks();
    }
  };

  const handleOpenDir = async (dir: string) => {
    if (!dir || !dir.trim()) {
      showToast('目录不存在', 'error');
      return;
    }
    try {
      await api.openInFileManager(dir);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      if (msg.includes('不存在') || msg.includes('no such file')) {
        showToast('目录不存在', 'error');
      } else {
        showToast(`打开失败: ${msg}`, 'error');
      }
    }
  };

  const statusBadge = (status: TaskStatus) => {
    switch (status) {
      case 1:
        return null;
      case 2:
        return (
          <span className="flex items-center space-x-1.5 text-amber-600 bg-amber-50 dark:bg-amber-500/15 dark:text-amber-400 px-2.5 py-0.5 rounded-full text-[11px] font-semibold border border-amber-200 dark:border-amber-500/30">
            <Pause size={10} />
            <span>{t('taskBoard.statusPaused', '已暂停')}</span>
          </span>
        );
      case 3:
        return null;
      case 4:
        return (
          <span className="flex items-center space-x-1.5 text-rose-600 bg-rose-50 dark:bg-rose-500/15 dark:text-rose-400 px-2.5 py-0.5 rounded-full text-[11px] font-semibold border border-rose-200 dark:border-rose-500/30">
            <AlertTriangle size={11} />
            <span>{t('taskBoard.statusFailed', '失败')}</span>
          </span>
        );
      default:
        return (
          <span className="flex items-center space-x-1.5 text-slate-500 bg-slate-100 dark:bg-white/10 px-2.5 py-0.5 rounded-full text-[11px] border border-slate-200 dark:border-white/10">
            <Clock size={11} />
            <span>{t('taskBoard.statusPending', '排队就绪')}</span>
          </span>
        );
    }
  };

  const stepsList = [
    { key: 'whisper', label: t('taskBoard.steps.whisper', '听写转录') },
    { key: 'split', label: t('taskBoard.steps.split', '语义分句') },
    { key: 'translate', label: t('taskBoard.steps.translate', '智能翻译') },
    { key: 'tts', label: t('taskBoard.steps.tts', '语音合成') },
    { key: 'merge', label: t('taskBoard.steps.merge', '音视频混音') },
    { key: 'burn', label: t('taskBoard.steps.burn', '压制合成') },
  ];

  const formatStepName = (step?: string) => {
    if (!step) return t('taskBoard.statusPending', '排队就绪');
    const map: Record<string, string> = {
      whisper: t('taskBoard.steps.whisper', '听写转录'),
      split: t('taskBoard.steps.split', '语义分句'),
      translate: t('taskBoard.steps.translate', '智能翻译'),
      tts: t('taskBoard.steps.tts', '语音合成'),
      merge: t('taskBoard.steps.merge', '音视频混音'),
      burn: t('taskBoard.steps.burn', '压制合成'),
    };
    return map[step] || step;
  };

  const formatStepDetail = (detail?: string) => {
    if (!detail) return '';
    // 严格清洗屏蔽底层命令字样 ffmpeg，使用规范领域词汇
    const cleaned = detail.replace(/ffmpeg/gi, '').trim();
    if (!cleaned) return '';
    return cleaned.startsWith('·') ? cleaned : `· ${cleaned}`;
  };

  const getStepStatus = (stepKey: string, task: Task) =>
    taskBoardStepStatus(stepKey, task, taskStepStates[task.id]);

  return (
    <div className="h-screen flex flex-col overflow-y-auto px-8 py-6 select-none relative">
      {/* 悬浮错误/状态通知队列，允许向下堆叠，最多显示 3 个 */}
      {toasts.length > 0 && (
        <div className="fixed top-5 left-1/2 -translate-x-1/2 z-50 flex flex-col items-center space-y-2 pointer-events-none w-full max-w-lg px-4">
          {toasts.map((toast) => (
            <div
              key={toast.id}
              className={`pointer-events-auto flex items-center justify-between w-full gap-2.5 px-4 py-2.5 rounded-2xl bg-white/95 dark:bg-[#1C1C1E]/95 backdrop-blur-xl border shadow-2xl transition-all animate-in fade-in slide-in-from-top-2 duration-200 ${
                toast.type === 'error'
                  ? 'border-rose-300 dark:border-rose-500/40 text-rose-600 dark:text-rose-400 shadow-rose-500/10'
                  : 'border-emerald-300 dark:border-emerald-500/40 text-emerald-600 dark:text-emerald-400 shadow-emerald-500/10'
              } text-xs font-bold`}
            >
              <div className="flex items-center gap-2.5 min-w-0 pr-1">
                <AlertCircle
                  size={15}
                  className={`shrink-0 ${toast.type === 'error' ? 'text-rose-500' : 'text-emerald-500'}`}
                />
                <span className="truncate">{toast.text}</span>
              </div>
              <button
                type="button"
                onClick={() => removeToast(toast.id)}
                className="ml-2 p-1 rounded-lg hover:bg-black/5 dark:hover:bg-white/10 opacity-60 hover:opacity-100 transition-opacity shrink-0"
              >
                <X size={13} />
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="max-w-5xl w-full mx-auto space-y-6 pt-4 pb-12">
        {/* 顶部标题与操作 */}
        <div className="flex items-center justify-between wails-drag">
          <div>
            <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
              {t('taskBoard.title', '任务执行看板')}
            </h2>
          </div>
          <RefreshButton
            onRefresh={loadTasks}
            size={13}
            title={t('asr.refreshList', '刷新列表')}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-xl border border-slate-200 dark:border-white/10 hover:bg-slate-100 dark:hover:bg-white/5 text-xs text-slate-600 dark:text-slate-300 transition-colors wails-no-drag"
          >
            <span>{t('asr.refreshList', '刷新')}</span>
          </RefreshButton>
        </div>

        {/* 状态统计卡片 */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {[
            { label: t('taskBoard.totalTasks', '全部任务'), count: tasks.length, color: 'text-slate-800 dark:text-white' },
            { label: t('taskBoard.runningTasks', '处理中'), count: tasks.filter((t) => t.status === 1).length, color: 'text-blue-600 dark:text-blue-400' },
            { label: t('taskBoard.completedTasks', '已完成'), count: tasks.filter((t) => t.status === 3).length, color: 'text-emerald-600 dark:text-emerald-400' },
            { label: t('taskBoard.failedTasks', '异常/失败'), count: tasks.filter((t) => t.status === 4).length, color: 'text-rose-600 dark:text-rose-400' },
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
          <div className="p-16 text-center text-xs text-slate-400">{t('common.loading', '正在载入流水线任务...')}</div>
        ) : tasks.length === 0 ? (
          <div className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-16 text-center space-y-3">
            <div className="w-12 h-12 rounded-2xl bg-slate-100 dark:bg-white/5 text-slate-400 flex items-center justify-center mx-auto">
              <Sparkles size={20} />
            </div>
            <p className="text-sm font-semibold text-slate-800 dark:text-white">
              {t('taskBoard.emptyTitle', '暂无正在进行或历史任务')}
            </p>
            <p className="text-xs text-slate-400 max-w-sm mx-auto">
              {t('taskBoard.emptyDesc', '前往「工作台」或「字幕合成」提交视频，即可在此处观测全流程流水线执行细节')}
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {tasks.map((task) => {
              const isExpanded = activeLogTaskId === task.id;
              const activeSteps = taskBoardActiveSteps(task, taskStepStates[task.id] || {});
              const fileName = task.original_name || (task.input_path ? task.input_path.split(/[\\/]/).pop() : '未命名任务视频');

              return (
                <div
                  key={task.id}
                  className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-4 sm:p-5 space-y-3 transition-all"
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
                        {t('taskBoard.outputDir', '输出目录')}：{task.batch_dir || task.output_dir}
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

                      {/* 失败任务提供直接的重试按钮 */}
                      {task.status === 4 && (
                        <button
                          type="button"
                          onClick={() => handleRetryTask(task)}
                          className="flex items-center space-x-1 px-2.5 py-1.5 rounded-xl bg-rose-50 hover:bg-rose-100 text-rose-600 dark:bg-rose-500/10 dark:hover:bg-rose-500/20 dark:text-rose-400 border border-rose-200 dark:border-rose-500/30 text-xs font-semibold transition-all active:scale-95 shadow-sm"
                          title={t('taskBoard.retryTask', '从失败节点重新执行')}
                        >
                          <RotateCcw size={13} className="shrink-0" />
                          <span>{t('taskBoard.retryBtn', '重试')}</span>
                        </button>
                      )}

                      {/* 已完成或失败时提供指定节点重跑与修改配方入口 */}
                      {(task.status === 3 || task.status === 4) && (
                        <div className="flex items-center space-x-1">
                          <button
                            type="button"
                            onClick={() => setRecipeModalTask(task)}
                            className="p-2 rounded-xl border border-blue-200 dark:border-blue-500/30 hover:bg-blue-50 dark:hover:bg-blue-500/10 text-blue-600 dark:text-blue-400 text-xs transition-colors"
                            title={t('taskBoard.rerunWithRecipe', '修改配方并重跑')}
                          >
                            <Sliders size={14} />
                          </button>

                          <div className="relative">
                            <button
                              type="button"
                              onClick={() => setRerunMenuTaskId(rerunMenuTaskId === task.id ? null : task.id)}
                              className="flex items-center space-x-1 p-2 rounded-xl border border-slate-200 dark:border-white/10 hover:bg-slate-100 dark:hover:bg-white/5 text-slate-600 dark:text-slate-300 text-xs transition-colors"
                              title={t('taskBoard.rerunQuick', '重跑起始节点')}
                            >
                              <RotateCcw size={14} />
                            </button>

                            {rerunMenuTaskId === task.id && (
                              <div className="absolute right-0 top-full mt-1.5 w-60 bg-white dark:bg-[#1E1E20] border border-slate-200 dark:border-[#2C2C2E] rounded-xl shadow-xl p-1.5 z-40 space-y-0.5 animate-in fade-in zoom-in-95">
                                <button
                                  type="button"
                                  onClick={() => {
                                    setRerunMenuTaskId(null);
                                    setRecipeModalTask(task);
                                  }}
                                  className="w-full text-left px-2.5 py-1.5 rounded-lg text-xs text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-500/10 font-bold transition-colors flex items-center space-x-1.5 border-b border-slate-100 dark:border-white/10 mb-1"
                                >
                                  <Sliders size={12} />
                                  <span>{t('taskBoard.rerunWithRecipe', '修改配方并重跑')}</span>
                                </button>
                                <div className="px-2.5 py-1 text-[10px] font-bold text-slate-400 uppercase tracking-wider">
                                  {t('taskBoard.rerunQuick', '选择快速重跑起始节点')}
                                </div>
                                {stepsList.map((step) => {
                                  let tip = '';
                                  if (step.key === 'whisper') tip = t('taskBoard.rerunTips.whisper', '全量从头跑');
                                  else if (step.key === 'translate') tip = t('taskBoard.rerunTips.translate', '重合成声音');
                                  else if (step.key === 'tts') tip = t('taskBoard.rerunTips.tts', '重新生成声音');
                                  else if (step.key === 'merge') tip = t('taskBoard.rerunTips.merge', '重算时间轴');
                                  else if (step.key === 'burn') tip = t('taskBoard.rerunTips.burn', '直接压成片');
                                  return (
                                    <button
                                      key={step.key}
                                      type="button"
                                      onClick={() => handleRerunFromStep(task.id, step.key, step.label)}
                                      className="w-full text-left px-2.5 py-1.5 rounded-lg text-xs text-slate-700 dark:text-slate-200 hover:bg-blue-50 dark:hover:bg-blue-500/10 hover:text-blue-600 dark:hover:text-blue-400 font-medium transition-colors flex items-center justify-between"
                                    >
                                      <span>从「{step.label}」</span>
                                      {tip && <span className="text-[10px] text-slate-400 font-normal">{tip}</span>}
                                    </button>
                                  );
                                })}
                              </div>
                            )}
                          </div>
                        </div>
                      )}

                      <button
                        onClick={() => toggleLogs(task.id)}
                        className={`p-2 rounded-xl border text-xs flex items-center space-x-1 transition-colors ${
                          isExpanded
                            ? 'bg-blue-600 text-white border-blue-600'
                            : 'border-slate-200 dark:border-white/10 hover:bg-slate-100 dark:hover:bg-white/5 text-slate-600 dark:text-slate-300'
                        }`}
                        title={isExpanded ? t('taskBoard.hideLogs', '收起日志流') : t('taskBoard.viewLogs', '查看终端日志流')}
                      >
                        <Terminal size={14} />
                        {isExpanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                      </button>

                      {deletingId === task.id ? (
                        <div className="flex items-center space-x-1.5 bg-rose-50 dark:bg-rose-950/40 border border-rose-300 dark:border-rose-500/40 rounded-xl px-1.5 py-1">
                          <button
                            type="button"
                            onClick={() => handleDelete(task.id)}
                            className="px-2 py-0.5 rounded-lg bg-rose-600 hover:bg-rose-700 text-white text-[11px] font-semibold transition-colors shadow-sm"
                            title={t('common.delete', '删除')}
                          >
                            {t('common.confirm', '确认')}
                          </button>
                          <button
                            type="button"
                            onClick={() => setDeletingId(null)}
                            className="px-1.5 py-0.5 rounded-lg text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 text-[11px] transition-colors"
                            title={t('common.cancel', '取消')}
                          >
                            {t('common.cancel', '取消')}
                          </button>
                        </div>
                      ) : (
                        <button
                          type="button"
                          onClick={() => setDeletingId(task.id)}
                          className="p-2 rounded-xl border border-rose-200 dark:border-rose-500/30 hover:bg-rose-50 dark:hover:bg-rose-500/10 text-rose-600 dark:text-rose-400 text-xs transition-colors"
                          title={t('taskBoard.deleteTask', '删除任务')}
                        >
                          <Trash2 size={14} />
                        </button>
                      )}
                    </div>
                  </div>

                  {/* 进度条与当前步骤 */}
                  <div className="space-y-1.5 pt-0.5">
                    <div className="flex justify-between items-center text-xs">
                      <span className="text-slate-500 dark:text-slate-400">
                        {t('taskBoard.currentStepPrefix', '当前步骤')}：
                        <strong className="text-slate-800 dark:text-slate-100 ml-1 font-bold">
                          {activeSteps.length > 1 ? activeSteps.map(formatStepName).join(" · ") : formatStepName(task.current_step)}
                          {activeSteps.length <= 1 && formatStepDetail(task.current_detail)}
                        </strong>
                      </span>
                      <div className="flex items-center gap-1.5">
                        <span className="font-mono font-bold text-blue-600 dark:text-blue-400 text-xs">
                          {task.progress}%
                        </span>
                        {task.status === 1 && (
                          <span className="relative flex h-2 w-2" title={t('taskBoard.statusRunning', '处理中')}>
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500" />
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="w-full h-1.5 bg-slate-100 dark:bg-white/10 rounded-full overflow-hidden">
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

                    {/* 失败状态错误提示框 */}
                    {task.status === 4 && task.error && (
                      <div className="p-2.5 rounded-xl bg-rose-50/80 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-500/30 text-rose-600 dark:text-rose-300 text-xs flex items-start gap-2 animate-in fade-in">
                        <AlertCircle size={14} className="shrink-0 mt-0.5 text-rose-500" />
                        <div className="min-w-0 flex-1">
                          <span className="font-semibold">{t('taskBoard.failureReason', '失败原因')}：</span>
                          <span className="font-mono break-all">{task.error}</span>
                        </div>
                      </div>
                    )}
                  </div>

                    {/* 步骤流转状态条（紧凑无分割线，轻量附着在进度条底下） */}
                    <div className="grid grid-cols-6 gap-1.5 pt-0.5">
                      {stepsList.map((step) => {
                        const status = getStepStatus(step.key, task);
                        const durationSec = taskStepsMap[task.id]?.[step.key];
                        // 严格门禁：只有步骤处于已完成（done）状态时才允许显示耗时，处理中（current/1s）严禁展示
                        const showDuration = status === 'done' && durationSec !== undefined && durationSec > 0;
                        return (
                          <div
                            key={step.key}
                            className={`text-center py-1 px-1 rounded-lg text-[10px] font-medium transition-colors ${
                              status === 'error'
                                ? 'bg-rose-500/15 text-rose-600 dark:text-rose-400 font-semibold'
                                : status === 'current'
                                ? 'bg-blue-500/15 text-blue-600 dark:text-blue-400 font-semibold'
                                : status === 'done'
                                ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
                                : 'text-slate-400 bg-slate-100/60 dark:bg-white/[0.03]'
                            }`}
                          >
                            <span>{step.label}</span>
                            {showDuration && (
                              <span className="ml-1 font-mono text-[9px] opacity-80">
                                {durationSec}s
                              </span>
                            )}
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
                          <span className="ml-2 font-sans font-bold text-slate-300">
                            {t('taskBoard.terminalLogTitle', '执行终端日志流')}
                          </span>
                        </div>
                        <span>
                          {t('taskBoard.terminalLogCount', '共 {{count}} 行输出').replace('{{count}}', String(logs.length))}
                        </span>
                      </div>

                      <div
                        ref={logContainerRef}
                        onScroll={handleLogScroll}
                        className="max-h-60 overflow-y-auto space-y-1 pr-2 select-text scroll-smooth"
                      >
                        {logs.length === 0 ? (
                          <div className="text-slate-500 italic py-2">
                            {t('taskBoard.terminalEmpty', '暂无流水线详细日志输出...')}
                          </div>
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
                                {log.level === 'success' ? 'SUCC' : log.level}
                              </span>
                              {log.step && (
                                <span className="text-purple-400 shrink-0 text-[10px]">[{formatStepName(log.step)}]</span>
                              )}
                              <span className="text-slate-300 break-all whitespace-pre-wrap">{log.message}</span>
                            </div>
                          ))
                        )}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* 调整配方并重跑弹窗 */}
      {recipeModalTask && (
        <RerunRecipeModal
          task={recipeModalTask}
          onClose={() => setRecipeModalTask(null)}
          onSubmit={handleRerunWithRecipe}
        />
      )}
    </div>
  );
};
