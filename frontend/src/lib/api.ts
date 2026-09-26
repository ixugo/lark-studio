import { Task, TaskLog, CreateTaskInput, ConfigDTO, Term } from '../types';

declare global {
  interface Window {
    __onWailsFilesDropped?: (files: string[]) => void;
    _wails?: {
      invoke?: (binding: string, args?: unknown) => Promise<unknown>;
      flags?: Record<string, unknown>;
      handlePlatformFileDrop?: (filenames: string[], x: number, y: number) => void;
    };
    wails?: {
      Call?: {
        ByName: <T = unknown>(name: string, ...args: unknown[]) => Promise<T>;
        ByID: <T = unknown>(id: number, ...args: unknown[]) => Promise<T>;
      };
      Dialogs?: {
        OpenFile: (options?: {
          CanChooseFiles?: boolean;
          CanChooseDirectories?: boolean;
          AllowsMultipleSelection?: boolean;
          Title?: string;
          Filters?: Array<{ DisplayName: string; Pattern: string }>;
        }) => Promise<string[] | string>;
      };
      Events?: {
        On: (event: string, callback: (data: unknown) => void) => () => void;
        Emit: (event: string, data?: unknown) => void;
      };
    };
    go?: {
      wails?: {
        AppService?: {
          ListTasks: () => Promise<Task[]>;
          GetTask: (id: string) => Promise<Task>;
          ListTaskLogs: (id: string) => Promise<TaskLog[]>;
          CreateTask: (input: CreateTaskInput) => Promise<Task>;
          BatchCreateTasks: (videos: string[], recipe: CreateTaskInput) => Promise<Task[]>;
          PauseTask: (id: string) => Promise<void>;
          ResumeTask: (id: string) => Promise<void>;
          DeleteTask: (id: string) => Promise<void>;
          PickFiles: () => Promise<string[]>;
          OpenInFileManager: (path: string) => Promise<void>;
          GetConfig: () => Promise<ConfigDTO>;
          UpdateConfig: (updates: Record<string, unknown>) => Promise<void>;
          ListTerms: () => Promise<Term[]>;
          SaveTerm: (source: string, target: string) => Promise<void>;
          DeleteTerm: (id: number) => Promise<void>;
        };
      };
    };
  }
}

// 模拟本地内存状态（用于纯浏览器开发与无桌面环境单元测试）
let mockTasks: Task[] = [
  {
    id: 'demo_task_01',
    created_at: new Date(Date.now() - 3600000).toISOString(),
    updated_at: new Date().toISOString(),
    input_path: 'sample-media/OpenAI_DevDay_Keynote.mp4',
    output_dir: 'sample-output/OpenAI_DevDay_Keynote_vdub',
    mode: 3,
    status: 1,
    current_step: 'translate',
    current_detail: 'gpt-4o-mini',
    progress: 45,
    step_progress: 80,
    target_lang: 'zh-CN',
    source_lang: 'en',
    translator: 'openai',
    output_content: 'bilingual',
    tts_engine: 'edge',
    tts_voice: 'zh-CN-XiaoxiaoNeural',
    speech_rate: 1.1,
    subtitle_output: 'burn',
  },
  {
    id: 'demo_task_02',
    created_at: new Date(Date.now() - 86400000).toISOString(),
    updated_at: new Date(Date.now() - 85000000).toISOString(),
    input_path: 'sample-media/WWDC_VisionPro_Demo.mp4',
    output_dir: 'sample-output/WWDC_VisionPro_Demo_vdub',
    mode: 2,
    status: 3,
    current_step: 'burn',
    current_detail: 'ffmpeg',
    progress: 100,
    step_progress: 100,
    target_lang: 'zh-CN',
    source_lang: 'en',
    translator: 'bing',
    output_content: 'bilingual',
    tts_engine: 'edge',
    tts_voice: 'zh-CN-YunxiNeural',
    speech_rate: 1.0,
    subtitle_output: 'burn',
  }
];

let mockLogs: Record<string, TaskLog[]> = {
  demo_task_01: [
    { id: '1', task_id: 'demo_task_01', created_at: new Date().toISOString(), level: 'info', step: '', message: '任务创建，模式：翻译配音' },
    { id: '2', task_id: 'demo_task_01', created_at: new Date().toISOString(), level: 'info', step: 'whisper', message: '语音识别开始 (Whisper.cpp large-v3-turbo)' },
    { id: '3', task_id: 'demo_task_01', created_at: new Date().toISOString(), level: 'success', step: 'whisper', message: '语音识别完成，共提取 128 条字幕' },
    { id: '4', task_id: 'demo_task_01', created_at: new Date().toISOString(), level: 'info', step: 'translate', message: '正在调用大模型进行上下文智能翻译 (gpt-4o-mini)...' },
  ],
};

let mockConfig: ConfigDTO = {
  pipeline: {
    workers: 2,
    whisper_mode: 'whisper-cpp',
    whisper_model: 'large-v3-turbo',
    ffmpeg_bin: 'ffmpeg',
    default_target_lang: 'zh-CN',
    translate_prompt: '',
    max_speed_factor: 1.5,
    translate_chunk_size: 10,
    tts_workers: 2,
    clean_intermediate: false,
    subtitle_output: 'burn',
  },
  llm: {
    provider: 'openai',
    base_url: 'https://api.openai.com/v1',
    api_key: '',
    model: 'gpt-4o-mini',
    deeplx_url: 'http://127.0.0.1:1188/translate',
  },
  tts: {
    type: 'edge',
    voice: 'zh-CN-XiaoxiaoNeural',
    base_url: '',
    api_key: '',
    model: 'tts-1',
  },
  lip_sync: {
    enabled: false,
    base_url: '',
    api_key: '',
  },
  runtime: {
    debug: true,
    build_version: '1.0.0-wails3',
    config_dir: '~/dsub/configs',
    config_path: '~/dsub/configs/config.toml',
  },
};

let mockTerms: Term[] = [
  { id: 1, glossary_id: 1, text: 'Agent', translation: '智能体', note: 'AI Agent' },
  { id: 2, glossary_id: 1, text: 'Prompt', translation: '提示词', note: 'LLM Prompt' },
  { id: 3, glossary_id: 1, text: 'Fine-tuning', translation: '微调', note: '模型调优' },
];

const WAILS_SERVICE_PREFIX = 'github.com/ixugo/vdub/internal/wails.AppService';

async function invokeWailsMethod<T>(methodName: string, ...args: unknown[]): Promise<{ called: boolean; result?: T }> {
  if (typeof window === 'undefined') return { called: false };

  // 1. 优先尝试 Wails 3 Call.ByName
  if (window.wails?.Call?.ByName) {
    try {
      const res = await window.wails.Call.ByName<T>(`${WAILS_SERVICE_PREFIX}.${methodName}`, ...args);
      return { called: true, result: res };
    } catch (err) {
      console.warn(`[Wails3] Call.ByName ${methodName} 失败:`, err);
    }
  }

  // 2. 兼容 Wails 2 window.go.wails.AppService
  const wails2Svc = window.go?.wails?.AppService as Record<string, (...a: unknown[]) => Promise<T>> | undefined;
  if (wails2Svc && typeof wails2Svc[methodName] === 'function') {
    try {
      const res = await wails2Svc[methodName](...args);
      return { called: true, result: res };
    } catch (err) {
      console.warn(`[Wails2] ${methodName} 失败:`, err);
    }
  }

  return { called: false };
}

export const api = {
  // ─── 任务操作 ───────────────────────────────────────────
  async listTasks(): Promise<Task[]> {
    const res = await invokeWailsMethod<Task[]>('ListTasks');
    if (res.called && res.result) return res.result;
    return [...mockTasks];
  },

  async getTask(id: string): Promise<Task> {
    const res = await invokeWailsMethod<Task>('GetTask', id);
    if (res.called && res.result) return res.result;
    const t = mockTasks.find((item) => item.id === id);
    if (!t) throw new Error('任务不存在');
    return t;
  },

  async listTaskLogs(id: string): Promise<TaskLog[]> {
    const res = await invokeWailsMethod<TaskLog[]>('ListTaskLogs', id);
    if (res.called && res.result) return res.result;
    return mockLogs[id] || [];
  },

  async createTask(input: CreateTaskInput): Promise<Task> {
    const res = await invokeWailsMethod<Task>('CreateTask', input);
    if (res.called && res.result) return res.result;
    const newTask: Task = {
      id: `task_${Date.now()}`,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      input_path: input.input_path,
      output_dir: input.output_dir || `${input.input_path}_vdub`,
      mode: input.mode,
      status: 1,
      current_step: 'whisper',
      current_detail: 'Whisper.cpp',
      progress: 5,
      step_progress: 10,
      target_lang: input.target_lang || 'zh-CN',
      source_lang: input.source_lang || 'auto',
      translator: input.translator === 'local' ? 'openai' : (input.translator || 'bing'),
      output_content: input.output_content || 'bilingual',
      tts_engine: input.tts_engine || 'edge',
      tts_voice: input.tts_voice || 'zh-CN-XiaoxiaoNeural',
      speech_rate: input.speech_rate || 1.1,
      subtitle_output: input.subtitle_output || 'burn',
    };
    mockTasks = [newTask, ...mockTasks];
    mockLogs[newTask.id] = [
      { id: `${Date.now()}`, task_id: newTask.id, created_at: new Date().toISOString(), level: 'info', step: '', message: '任务创建成功' }
    ];
    return newTask;
  },

  async batchCreateTasks(videos: string[], recipe: CreateTaskInput): Promise<Task[]> {
    const res = await invokeWailsMethod<Task[]>('BatchCreateTasks', videos, recipe);
    if (res.called && res.result) return res.result;
    const tasks: Task[] = [];
    for (const v of videos) {
      tasks.push(await this.createTask({ ...recipe, input_path: v }));
    }
    return tasks;
  },

  async pauseTask(id: string): Promise<void> {
    const res = await invokeWailsMethod<void>('PauseTask', id);
    if (res.called) return;
    mockTasks = mockTasks.map((t) => (t.id === id ? { ...t, status: 2 } : t));
  },

  async resumeTask(id: string): Promise<void> {
    const res = await invokeWailsMethod<void>('ResumeTask', id);
    if (res.called) return;
    mockTasks = mockTasks.map((t) => (t.id === id ? { ...t, status: 1 } : t));
  },

  async deleteTask(id: string): Promise<void> {
    const res = await invokeWailsMethod<void>('DeleteTask', id);
    if (res.called) return;
    mockTasks = mockTasks.filter((t) => t.id !== id);
    delete mockLogs[id];
  },

  // ─── 文件交互 ───────────────────────────────────────────
  async pickFiles(): Promise<string[]> {
    // 1. 优先调用 Wails 3 原生 Dialogs.OpenFile
    if (typeof window !== 'undefined' && window.wails?.Dialogs?.OpenFile) {
      try {
        const selected = await window.wails.Dialogs.OpenFile({
          CanChooseFiles: true,
          AllowsMultipleSelection: true,
          Title: '选择音视频或字幕文件',
          Filters: [
            { DisplayName: '音视频与字幕 (*.mp4,*.mkv,*.mov,*.mp3,*.wav,*.srt...)', Pattern: '*.mp4;*.mkv;*.mov;*.avi;*.webm;*.flv;*.mp3;*.wav;*.m4a;*.srt' },
            { DisplayName: '所有文件 (*.*)', Pattern: '*.*' },
          ],
        });
        if (Array.isArray(selected) && selected.length > 0) return selected;
        if (typeof selected === 'string' && selected) return [selected];
        if (Array.isArray(selected) && selected.length === 0) return [];
      } catch (err) {
        console.warn('[Wails3] Dialogs.OpenFile 异常，尝试服务方法:', err);
      }
    }

    // 2. 尝试调用 Go 后端 AppService.PickFiles
    const res = await invokeWailsMethod<string[]>('PickFiles');
    if (res.called && res.result) {
      return res.result;
    }

    // 3. 浏览器端兜底处理
    if (typeof window === 'undefined') return [];
    return new Promise((resolve) => {
      const input = document.createElement('input');
      input.type = 'file';
      input.multiple = true;
      input.accept = 'video/*,audio/*,.srt';
      input.onchange = () => {
        if (!input.files || input.files.length === 0) {
          resolve([]);
          return;
        }
        const paths = Array.from(input.files).map((f) => (f as unknown as { path?: string }).path || f.name);
        resolve(paths);
      };
      input.click();
    });
  },

  async openInFileManager(path: string): Promise<void> {
    const res = await invokeWailsMethod<void>('OpenInFileManager', path);
    if (res.called) return;
    console.log('Open in file manager:', path);
  },

  // ─── 配置与服务 ─────────────────────────────────────────
  async getConfig(): Promise<ConfigDTO> {
    const res = await invokeWailsMethod<ConfigDTO>('GetConfig');
    if (res.called && res.result) return res.result;
    return { ...mockConfig };
  },

  async updateConfig(updates: Record<string, unknown>): Promise<void> {
    const res = await invokeWailsMethod<void>('UpdateConfig', updates);
    if (res.called) return;
    mockConfig = { ...mockConfig, ...(updates as unknown as ConfigDTO) };
  },

  // ─── 术语库 ─────────────────────────────────────────────
  async listTerms(): Promise<Term[]> {
    const res = await invokeWailsMethod<Term[]>('ListTerms');
    if (res.called && res.result) return res.result;
    return [...mockTerms];
  },

  async saveTerm(source: string, target: string): Promise<void> {
    const res = await invokeWailsMethod<void>('SaveTerm', source, target);
    if (res.called) return;
    mockTerms.push({ id: Date.now(), glossary_id: 1, text: source, translation: target });
  },

  async deleteTerm(id: number): Promise<void> {
    const res = await invokeWailsMethod<void>('DeleteTerm', id);
    if (res.called) return;
    mockTerms = mockTerms.filter((item) => item.id !== id);
  },

  // ─── 事件监听 ───────────────────────────────────────────
  onEvent(event: string, callback: (data: unknown) => void): () => void {
    if (typeof window !== 'undefined' && window.wails?.Events?.On) {
      return window.wails.Events.On(event, (eventObj: unknown) => {
        const payload =
          eventObj && typeof eventObj === 'object' && 'data' in eventObj
            ? (eventObj as { data: unknown }).data
            : eventObj;
        callback(payload);
      });
    }
    if (typeof window === 'undefined') return () => {};
    // 本地事件总线模拟
    const handler = (e: CustomEvent) => callback(e.detail);
    window.addEventListener(`wails:${event}`, handler as EventListener);
    return () => window.removeEventListener(`wails:${event}`, handler as EventListener);
  },

  isWailsEnvironment(): boolean {
    return typeof window !== 'undefined' && (!!window.wails?.Call || !!window.wails?.Dialogs || !!window.go?.wails?.AppService);
  },
};
