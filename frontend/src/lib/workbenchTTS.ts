import type { ConfigDTO, TTSCapabilities } from '../types';
import { normalizeTtsVoice, type TtsEngine } from './ttsVoices';
import { validateTTSSelection, type TTSSelectionError } from './ttsCapabilities';

export function configuredWorkbenchVoice(engine: TtsEngine, tts: ConfigDTO['tts']): string {
  if (engine === 'openai') return tts.openai_voice ?? (tts.type?.toLowerCase() === 'openai' ? tts.voice : '') ?? '';
  return normalizeTtsVoice('edge', tts.edge_voice || (tts.type?.toLowerCase() === 'edge' ? tts.voice : undefined));
}

export function validateWorkbenchTTS(capabilities: TTSCapabilities | null, tts: ConfigDTO['tts'], voice: string): TTSSelectionError | 'capabilities' {
  if (!capabilities) return 'capabilities';
  return validateTTSSelection(capabilities, { ...tts, voice });
}
