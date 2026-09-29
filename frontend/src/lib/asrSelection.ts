import { api } from './api';

export type ASREngine = '' | 'whisper-cpp' | 'openai';

export async function prepareASRSelection(engine: ASREngine, model: string): Promise<void> {
  if (!engine) throw new Error('请选择语音识别引擎');
  if (engine === 'whisper-cpp') {
    if (!model.trim()) throw new Error('请先下载 Whisper 识别模型，或在语音识别设置中配置自定义模型路径');
    await api.setActiveWhisperModel(model);
  } else {
    const { pipeline } = await api.getConfig();
    if (!pipeline.asr_base_url?.trim() || !pipeline.asr_model?.trim()) {
      throw new Error('请先配置 OpenAI 兼容语音识别的接口地址和模型名称');
    }
  }
  await api.updateConfig({ pipeline: { whisper_mode: engine } });
}
