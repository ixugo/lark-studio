import { WhisperModelItem } from '../types';

export interface WhisperModelChoice {
  name: string;
  path: string;
  custom: boolean;
}

const modelName = (value: string) => (value.split(/[\\/]/).pop() || '')
  .replace(/^Whisper(\.cpp)?\s*/i, '').replace(/^ggml-/i, '').replace(/\.bin$/i, '').trim();
const isVad = (value: string) => modelName(value).toLowerCase().startsWith('silero');

export function whisperModelChoices(items: WhisperModelItem[], configured: string): WhisperModelChoice[] {
  const choices = items.filter((item) => item.downloaded && item.kind !== 'vad' && item.path && !isVad(item.name) && !isVad(item.path))
    .map((item) => ({ name: modelName(item.name), path: item.path!, custom: false }));
  const path = configured.trim();
  if (path && /[\\/]|\.bin$/i.test(path) && !isVad(path) && !choices.some((item) => item.path === path)) {
    choices.unshift({ name: modelName(path), path, custom: true });
  }
  return choices;
}

export function selectWhisperModel(selection: string, configured: string, choices: WhisperModelChoice[]): string {
  const find = (value: string) => choices.find((item) => item.path === value || item.name === value);
  return find(selection)?.path || find(configured)?.path || choices.find((item) => item.name === 'large-v3-turbo')?.path || choices[0]?.path || '';
}

export function configuredWhisperModel(pipeline?: { whisper_model?: string; WhisperModel?: string }): string {
  return (pipeline?.whisper_model || pipeline?.WhisperModel || '').trim();
}
