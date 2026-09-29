import { describe, expect, it } from 'vitest';
import type { TTSCapabilities } from '../types';
import { validateTTSSelection } from './ttsCapabilities';

const capabilities = { protocol: 'mlx', voices: [{ id: 'Vivian', name: 'Vivian' }], voice_source: 'qwen_builtin',
  instructions: false, languages: ['Auto', 'Chinese'] } as TTSCapabilities;

describe('语音合成能力选择校验', () => {
  it('固定音色只能使用服务器允许的列表', () => {
    expect(validateTTSSelection(capabilities, { voice: 'Vivian', language: 'Auto' })).toBe('');
    expect(validateTTSSelection(capabilities, { voice: 'alloy' })).toBe('voice');
    expect(validateTTSSelection({ ...capabilities, voices: [], voice_source: 'remote' }, { voice: 'old-voice' })).toBe('voice');
  });
  it('无音色枚举的兼容服务仍允许手填音色，但不允许留空', () => {
    const manual = { ...capabilities, voices: [], voice_source: 'manual' as const };
    expect(validateTTSSelection(manual, { voice: 'custom-speaker' })).toBe('');
    expect(validateTTSSelection(manual, { voice: '' })).toBe('voice');
  });
  it('服务不支持情绪或所选语言时拒绝提交', () => {
    expect(validateTTSSelection(capabilities, { voice: 'Vivian', instructions: '开心' })).toBe('instructions');
    expect(validateTTSSelection(capabilities, { voice: 'Vivian', language: 'Invented' })).toBe('language');
  });
  it('支持指令的模型允许 4096 字符，但拒绝更长输入', () => {
    const supported = { ...capabilities, instructions: true };
    expect(validateTTSSelection(supported, { voice: 'Vivian', instructions: 'a'.repeat(4096) })).toBe('');
    expect(validateTTSSelection(supported, { voice: 'Vivian', instructions: 'a'.repeat(4097) })).toBe('instructions_length');
  });
});
