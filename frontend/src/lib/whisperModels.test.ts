import { describe, expect, it } from 'vitest';
import { whisperModelChoices, selectWhisperModel, configuredWhisperModel } from './whisperModels';
import { WhisperModelItem } from '../types';

const items: WhisperModelItem[] = [
  { name: 'silero-v6.2.0', kind: 'vad', downloaded: true, path: '/models/ggml-silero-v6.2.0.bin', size: '', desc: '' },
  { name: 'tiny', kind: 'asr', downloaded: true, path: '/models/ggml-tiny.bin', size: '', desc: '' },
  { name: 'small', kind: 'asr', downloaded: false, size: '', desc: '' },
];

describe('Whisper 模型选择', () => {
  it('读取桌面配置的旧式字段名', () => {
    expect(configuredWhisperModel({ WhisperModel: '/custom/ggml-tiny.bin' })).toBe('/custom/ggml-tiny.bin');
    expect(configuredWhisperModel({ whisper_model: ' tiny ', WhisperModel: 'small' })).toBe('tiny');
  });
  it('只提供已下载的识别模型，排除 Silero 和未下载项', () => {
    expect(whisperModelChoices(items, '')).toEqual([{ name: 'tiny', path: '/models/ggml-tiny.bin', custom: false }]);
  });
  it('未标 kind 的旧 Silero 记录也不能用于识别', () => {
    expect(whisperModelChoices([{ ...items[0], kind: undefined }], '')).toEqual([]);
  });
  it('管理目录外的自定义模型仍可选择，并保留实际路径', () => {
    const path = '/custom/ggml-large-v3-turbo.bin';
    const choices = whisperModelChoices(items, path);
    expect(choices[0]).toEqual({ name: 'large-v3-turbo', path, custom: true });
    expect(selectWhisperModel('', path, choices)).toBe(path);
  });
  it('自定义路径与已下载记录相同时不重复', () => {
    expect(whisperModelChoices(items, items[1].path!)).toHaveLength(1);
  });
  it('旧草稿的模型简称解析为实际文件路径', () => {
    expect(selectWhisperModel('tiny', '', whisperModelChoices(items, ''))).toBe('/models/ggml-tiny.bin');
  });
  it('旧草稿选到 Silero 时改用有效配置；无识别模型时留空', () => {
    expect(selectWhisperModel('silero-v6.2.0', '', whisperModelChoices(items, ''))).toBe('/models/ggml-tiny.bin');
    expect(selectWhisperModel('large-v3-turbo', '', whisperModelChoices([items[0]], ''))).toBe('');
  });
});
