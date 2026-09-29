import { describe, expect, it } from 'vitest';
import type { Task } from '../types';
import { taskBoardStepStatus, taskBoardActiveSteps, resetTaskStepStates } from './taskBoardSteps';

const parallelTask = { mode: 3, status: 1, current_step: 'translate' } as Task;

describe('看板独立并行步骤', () => {
  it('翻译与合成事件交替到达时，两项均保持蓝色进行中', () => {
    for (const current_step of ['translate', 'tts', 'translate', 'tts']) {
      const task = { ...parallelTask, current_step };
      expect(taskBoardStepStatus('translate', task, { translate: 1, tts: 1 })).toBe('current');
      expect(taskBoardStepStatus('tts', task, { translate: 1, tts: 1 })).toBe('current');
    }
  });
  it('配音任务刚开始翻译时，准备中的语音合成也显示进行中', () => {
    expect(taskBoardStepStatus('tts', parallelTask, { translate: 1 })).toBe('current');
  });
  it('仅重跑语音合成时，不重新点亮已完成的翻译', () => {
    const task = { ...parallelTask, current_step: 'tts' };
    expect(taskBoardStepStatus('translate', task, { translate: 2, tts: 1 })).toBe('done');
    expect(taskBoardStepStatus('tts', task, { translate: 2, tts: 1 })).toBe('current');
  });
  it('仅翻译和原文配音分别只点亮自己的阶段', () => {
    expect(taskBoardStepStatus('tts', { ...parallelTask, mode: 2 }, { translate: 1 })).toBe('pending');
    expect(taskBoardStepStatus('tts', { ...parallelTask, mode: 5, current_step: 'tts' }, { tts: 1 })).toBe('current');
  });
  it('暂停和失败不会留着两个进行中蓝块，完成以各自状态为准', () => {
    expect(taskBoardStepStatus('tts', { ...parallelTask, status: 2 }, { translate: 1, tts: 1 })).toBe('pending');
    expect(taskBoardStepStatus('tts', { ...parallelTask, status: 4, current_step: 'tts' }, { translate: 1, tts: 3 })).toBe('error');
    expect(taskBoardStepStatus('translate', { ...parallelTask, status: 4, current_step: 'tts' }, { translate: 1, tts: 3 })).toBe('pending');
    expect(taskBoardStepStatus('translate', { ...parallelTask, current_step: 'tts' }, { translate: 'done', tts: 'running' })).toBe('done');
  });
});

it('并行标题始终按固定顺序显示，重跑清除合成及后续旧状态', () => {
  for (const current_step of ['translate', 'tts', 'translate']) {
    expect(taskBoardActiveSteps({ ...parallelTask, current_step }, { translate: 1, tts: 1 })).toEqual(['translate', 'tts']);
  }
  const reset = resetTaskStepStates({ translate: 2, tts: 2, merge: 2, burn: 2 }, 'tts');
  expect(reset).toEqual({ translate: 2, tts: 0, merge: 0, burn: 0 });
  expect(taskBoardActiveSteps({ ...parallelTask, current_step: 'tts' }, reset)).toEqual(['tts']);
});
