import { describe, expect, it } from 'vitest';
import {
  DEFAULT_EDGE_TTS_VOICE,
  DEFAULT_OPENAI_TTS_VOICE,
  EDGE_TTS_VOICES,
  normalizeTtsVoice,
} from './ttsVoices';

describe('TTS 音色数据源', () => {
  it('Edge TTS 默认使用晓晓', () => {
    expect(DEFAULT_EDGE_TTS_VOICE).toBe('zh-CN-XiaoxiaoNeural');
    expect(EDGE_TTS_VOICES[0].value).toBe(DEFAULT_EDGE_TTS_VOICE);
  });

  it('提供多语言 Edge TTS 音色且没有 OpenAI 音色', () => {
    expect(EDGE_TTS_VOICES.length).toBeGreaterThanOrEqual(20);
    expect(EDGE_TTS_VOICES.some((voice) => voice.value.startsWith('zh-CN-'))).toBe(true);
    expect(EDGE_TTS_VOICES.some((voice) => voice.value.startsWith('en-US-'))).toBe(true);
    expect(EDGE_TTS_VOICES.some((voice) => ['alloy', 'echo', 'nova'].includes(voice.value))).toBe(false);
  });

  it('切换引擎时替换不兼容的音色', () => {
    expect(normalizeTtsVoice('edge', 'nova')).toBe(DEFAULT_EDGE_TTS_VOICE);
    expect(normalizeTtsVoice('openai', 'zh-CN-XiaoxiaoNeural')).toBe(DEFAULT_OPENAI_TTS_VOICE);
    expect(normalizeTtsVoice('edge', 'en-US-JennyNeural')).toBe('en-US-JennyNeural');
    expect(normalizeTtsVoice('openai', 'custom-voice')).toBe('custom-voice');
  });
});
