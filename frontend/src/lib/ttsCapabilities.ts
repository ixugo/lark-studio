import type { TTSCapabilities } from '../types';

type TTSSelection = { voice?: string; language?: string; instructions?: string };
export type TTSSelectionError = '' | 'voice' | 'instructions' | 'instructions_length' | 'language';

export function validateTTSSelection(capabilities: TTSCapabilities, selection: TTSSelection): TTSSelectionError {
  if ((selection.instructions || '').length > 4096) return 'instructions_length';
  if (selection.instructions?.trim() && !capabilities.instructions) return 'instructions';
  if (!selection.voice?.trim() ||
    (capabilities.voice_source !== 'manual' && capabilities.voices.length === 0) ||
    (capabilities.voices.length > 0 && !capabilities.voices.some(voice => voice.id === selection.voice))) return 'voice';
  if (capabilities.protocol === 'mlx' && capabilities.languages.length > 0 &&
    !capabilities.languages.includes(selection.language || 'Auto')) return 'language';
  return '';
}
