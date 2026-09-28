import { describe, expect, it } from 'vitest';
import { loadWorkbenchDraft, saveWorkbenchDraft, WorkbenchDraft } from './workbenchDraft';

function memoryStorage(): Storage {
  const values = new Map<string, string>();
  return {
    get length() { return values.size; },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => Array.from(values.keys())[index] ?? null,
    removeItem: (key) => { values.delete(key); },
    setItem: (key, value) => { values.set(key, value); },
  };
}

const draft: WorkbenchDraft = {
  selectedFiles: ['/tmp/video.mp4'],
  activePreset: 'dub_full',
  doSub: true,
  doTranslate: true,
  doDub: true,
  doVideo: true,
  whisperModel: 'large-v3-turbo',
  videoLang: 'auto',
  targetLang: 'zh-CN',
  translateService: 'bing',
  outputContent: 'bilingual',
  ttsEngine: 'edge',
  ttsVoice: 'zh-CN-YunyangNeural',
  speechRate: 1,
  subtitleOutput: 'soft',
  subtitleStyle: '经典白字黑边',
  videoQuality: '原画质(推荐)',
  encodeMethod: '默认(推荐)',
};

describe('工作台草稿', () => {
  it('重新进入工作台后恢复文件与用户选择的音色', () => {
    const storage = memoryStorage();
    saveWorkbenchDraft(draft, storage);

    expect(loadWorkbenchDraft(storage)).toEqual(draft);
  });

  it('损坏的草稿不会阻止工作台打开', () => {
    const storage = memoryStorage();
    storage.setItem('lark-studio.workbench-draft.v1', '{bad json');

    expect(loadWorkbenchDraft(storage)).toEqual({});
  });
});
