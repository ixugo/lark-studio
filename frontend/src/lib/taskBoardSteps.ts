import type { Task } from '../types';

export type StepStates = Record<string, number | string>;
export type StepVisualStatus = 'pending' | 'current' | 'done' | 'error';
export const TASK_STEP_ORDER = ['whisper', 'split', 'translate', 'tts', 'merge', 'burn'] as const;

const isDone = (state: number | string | undefined) => state === 2 || state === 'done' || state === 'completed';
const isRunning = (state: number | string | undefined) => state === 1 || state === 'running';
const isFailed = (state: number | string | undefined) => state === 3 || state === 'failed' || state === 'error';

// 使用独立步骤状态表示并行，不把语音合成归并到翻译，也不把进度 100 当作完成。
export function taskBoardStepStatus(step: string, task: Task, states: StepStates = {}): StepVisualStatus {
  if (task.status === 3) return 'done';
  const state = states[step];
  if (isDone(state)) return 'done';
  if (isFailed(state) && task.status !== 2) return 'error';

  const speechPhase = task.mode === 3 && ['translate', 'tts'].includes(task.current_step);
  const translating = isRunning(states.translate) ||
    (task.current_step === 'translate' && !isDone(states.translate) && !isFailed(states.translate));
  if (task.status === 1 && speechPhase && translating && (step === 'translate' || step === 'tts')) return 'current';
  if (isRunning(state)) return task.status === 1 ? 'current' : 'pending';
  if (task.current_step === step) {
    if (task.status === 4) return 'error';
    if (task.status === 1) return 'current';
  }
  if (state !== undefined) return 'pending';
  if (speechPhase && (step === 'translate' || step === 'tts')) return 'pending';
  const order: readonly string[] = TASK_STEP_ORDER;
  return order.indexOf(task.current_step) > order.indexOf(step) ? 'done' : 'pending';
}

export function taskBoardActiveSteps(task: Task, states: StepStates): string[] {
  return TASK_STEP_ORDER.filter(step => taskBoardStepStatus(step, task, states) === 'current');
}

export function resetTaskStepStates(states: StepStates, from: string): StepStates {
  const order: readonly string[] = TASK_STEP_ORDER;
  const index = order.indexOf(from);
  if (index < 0) return states;
  const reset = { ...states };
  for (const step of order.slice(index)) reset[step] = 0;
  return reset;
}
