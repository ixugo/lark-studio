import { describe, expect, it, vi, afterEach } from 'vitest';
import { api } from './api';
import { prepareASRSelection } from './asrSelection';

afterEach(() => vi.restoreAllMocks());

describe('工作台识别引擎提交前校验', () => {
  it('未选引擎或本地模型时拒绝，不能修改配置', async () => {
    const update = vi.spyOn(api, 'updateConfig').mockResolvedValue();
    const activate = vi.spyOn(api, 'setActiveWhisperModel').mockResolvedValue();
    await expect(prepareASRSelection('', '')).rejects.toThrow('选择');
    await expect(prepareASRSelection('whisper-cpp', '')).rejects.toThrow('下载');
    expect(update).not.toHaveBeenCalled();
    expect(activate).not.toHaveBeenCalled();
  });
  it('本地模型通过后才切换到本地引擎', async () => {
    const order: string[] = [];
    vi.spyOn(api, 'setActiveWhisperModel').mockImplementation(async () => { order.push('model'); });
    vi.spyOn(api, 'updateConfig').mockImplementation(async (value) => {
      expect(value).toEqual({ pipeline: { whisper_mode: 'whisper-cpp' } });
      order.push('engine');
    });
    await prepareASRSelection('whisper-cpp', '/custom/model.bin');
    expect(order).toEqual(['model', 'engine']);
  });
  it('OpenAI 兼容无需本地模型，但地址和模型名必须完整', async () => {
    const config = await api.getConfig();
    vi.spyOn(api, 'getConfig').mockResolvedValue({ ...config, pipeline: { ...config.pipeline, asr_base_url: '', asr_model: '' } });
    const update = vi.spyOn(api, 'updateConfig').mockResolvedValue();
    await expect(prepareASRSelection('openai', '')).rejects.toThrow('配置');
    expect(update).not.toHaveBeenCalled();
  });
  it('OpenAI 兼容允许无密钥的本地端点，仅应用选择的引擎', async () => {
    const config = await api.getConfig();
    vi.spyOn(api, 'getConfig').mockResolvedValue({ ...config, pipeline: { ...config.pipeline, asr_base_url: 'http://localhost:8000/v1', asr_model: 'whisper-1', asr_api_key: '' } });
    const activate = vi.spyOn(api, 'setActiveWhisperModel').mockResolvedValue();
    const update = vi.spyOn(api, 'updateConfig').mockResolvedValue();
    await prepareASRSelection('openai', '');
    expect(activate).not.toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({ pipeline: { whisper_mode: 'openai' } });
  });
});
