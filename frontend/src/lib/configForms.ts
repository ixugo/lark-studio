import type { ConfigDTO } from '../types';
import { normalizeTtsVoice, type TtsEngine } from './ttsVoices';

type Form = 'tts' | 'asr' | 'translation' | 'settings';
type Updates = {
  pipeline?: Partial<ConfigDTO['pipeline']>;
  llm?: ConfigDTO['llm'];
  tts?: ConfigDTO['tts'];
};

export function ttsVoiceForEngine(engine: TtsEngine, tts: ConfigDTO['tts']): string {
  const saved = engine === 'openai' ? tts.openai_voice : tts.edge_voice;
  return normalizeTtsVoice(engine, saved || ((tts.type?.toLowerCase() || 'edge') === engine ? tts.voice : undefined));
}

// 每页只保存自己编辑的字段，避免用旧快照覆盖其它页面的配置。
export function configFormUpdates(
  form: Form,
  config: ConfigDTO,
  activeEngine?: TtsEngine,
  defaultEngine?: TtsEngine,
): Updates {
  const p = config.pipeline;
  if (form === 'tts') {
    const engine = activeEngine || (config.tts.type?.toLowerCase() === 'openai' ? 'openai' : 'edge');
    const target = defaultEngine || (config.tts.type?.toLowerCase() === 'openai' ? 'openai' : 'edge');
    const tts = {
      ...config.tts,
      type: target,
      [engine === 'openai' ? 'openai_voice' : 'edge_voice']: normalizeTtsVoice(engine, config.tts.voice),
    };
    tts.voice = ttsVoiceForEngine(target, tts);
    return { tts, pipeline: { max_speed_factor: p.max_speed_factor, tts_workers: p.tts_workers } };
  }
  if (form === 'asr') {
    return { pipeline: {
      whisper_mode: p.whisper_mode,
      whisper_bin: p.whisper_bin,
      whisper_model: p.whisper_model,
      asr_base_url: p.asr_base_url,
      asr_api_key: p.asr_api_key,
      asr_model: p.asr_model,
    } };
  }
  if (form === 'translation') {
    return { llm: config.llm, pipeline: {
      translate_prompt: p.translate_prompt,
      translate_chunk_size: p.translate_chunk_size,
      default_target_lang: p.default_target_lang,
    } };
  }
  return { pipeline: {
    workers: p.workers,
    ffmpeg_bin: p.ffmpeg_bin,
    default_output_dir: p.default_output_dir,
    default_target_lang: p.default_target_lang,
    clean_intermediate: p.clean_intermediate,
  } };
}
