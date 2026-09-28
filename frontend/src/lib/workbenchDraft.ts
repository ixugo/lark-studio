import { TtsEngine } from './ttsVoices';

const WORKBENCH_DRAFT_KEY = 'lark-studio.workbench-draft.v1';

export interface WorkbenchDraft {
  selectedFiles: string[];
  activePreset: string;
  doSub: boolean;
  doTranslate: boolean;
  doDub: boolean;
  doVideo: boolean;
  whisperModel: string;
  videoLang: string;
  targetLang: string;
  translateService: 'openai' | 'local' | 'bing' | 'google';
  outputContent: string;
  ttsEngine: TtsEngine;
  ttsVoice: string;
  speechRate: number;
  subtitleOutput: string;
  subtitleStyle: string;
  videoQuality: string;
  encodeMethod: string;
}

export type WorkbenchDraftSnapshot = Partial<WorkbenchDraft>;

function storageAvailable(storage?: Storage): storage is Storage {
  return storage !== undefined;
}

export function loadWorkbenchDraft(storage: Storage | undefined = globalThis.localStorage): WorkbenchDraftSnapshot {
  if (!storageAvailable(storage)) return {};
  try {
    const value = JSON.parse(storage.getItem(WORKBENCH_DRAFT_KEY) || '{}');
    if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
    if (Array.isArray(value.selectedFiles)) {
      value.selectedFiles = value.selectedFiles.filter((item: unknown): item is string => typeof item === 'string');
    } else {
      delete value.selectedFiles;
    }
    return value as WorkbenchDraftSnapshot;
  } catch {
    return {};
  }
}

export function saveWorkbenchDraft(draft: WorkbenchDraft, storage: Storage | undefined = globalThis.localStorage): void {
  if (!storageAvailable(storage)) return;
  storage.setItem(WORKBENCH_DRAFT_KEY, JSON.stringify(draft));
}
