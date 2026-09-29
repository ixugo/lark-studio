import { describe, expect, it } from 'vitest';
import type { ConfigDTO, TTSCapabilities } from '../types';
import { configuredWorkbenchVoice, validateWorkbenchTTS } from './workbenchTTS';

const tts = { type: 'openai', voice: 'Vivian', openai_voice: 'Vivian', language: 'Auto' } as ConfigDTO['tts'];
const capabilities = { protocol: 'mlx', voices: [{ id: 'Vivian', name: 'Vivian' }], voice_source: 'qwen_builtin',
  instructions: false, languages: ['Auto'] } as TTSCapabilities;

describe('工作台固定音色校验', () => {
  it('未填写音色时取引擎独立配置，不把 Edge 音色用到 OpenAI', () => {
    expect(configuredWorkbenchVoice('openai', tts)).toBe('Vivian');
    expect(configuredWorkbenchVoice('openai', { ...tts, type: 'edge', voice: 'zh-CN-XiaoxiaoNeural', openai_voice: undefined })).toBe('');
    expect(configuredWorkbenchVoice('openai', { ...tts, openai_voice: undefined })).toBe('Vivian');
    expect(configuredWorkbenchVoice('edge', tts)).toBe('zh-CN-XiaoxiaoNeural');
  });
  it('草稿或配方的旧 alloy 不可静默覆盖当前固定音色', () => {
    expect(validateWorkbenchTTS(capabilities, tts, 'alloy')).toBe('voice');
    expect(validateWorkbenchTTS(capabilities, tts, 'Vivian')).toBe('');
    expect(validateWorkbenchTTS(null, tts, 'Vivian')).toBe('capabilities');
  });
});
