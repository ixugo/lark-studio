import { Task, TaskLog, TaskStep, CreateTaskInput, RerunTaskOptions, ConfigDTO, Term, AppInfo, UpdateInfo, UpdateStatus } from '../types';

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
          PickDirectory: () => Promise<string>;
          PickFFmpegFile: () => Promise<string>;
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
    output_dir: 'sample-output/OpenAI_DevDay_Keynote_lark_studio',
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
    output_dir: 'sample-output/WWDC_VisionPro_Demo_lark_studio',
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
    default_output_dir: '~/Documents/lark-studio',
    default_target_lang: 'zh-CN',
    translate_prompt: '',
    max_speed_factor: 1.2,
    translate_chunk_size: 10,
    tts_workers: 2,
    clean_intermediate: false,
    subtitle_output: 'burn',
  },
  llm: {
    provider: 'bing',
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

const WAILS_SERVICE_CANDIDATES = [
  'github.com/ixugo/vdub/internal/wails.AppService',
  'AppService',
  'main.AppService',
  '',
];

let detectedServicePrefix: string | null = null;

async function invokeWailsMethod<T>(methodName: string, ...args: unknown[]): Promise<{ called: boolean; result?: T }> {
  if (typeof window === 'undefined') return { called: false };

  // 1. 优先尝试 Wails 3 Call.ByName
  if (window.wails?.Call?.ByName) {
    const candidates = detectedServicePrefix
      ? [detectedServicePrefix]
      : WAILS_SERVICE_CANDIDATES;

    for (const prefix of candidates) {
      const fullMethod = prefix ? `${prefix}.${methodName}` : methodName;
      try {
        const res = await window.wails.Call.ByName<T>(fullMethod, ...args);
        detectedServicePrefix = prefix;
        return { called: true, result: res };
      } catch (err: unknown) {
        const errMsg = err instanceof Error ? err.message : String(err);
        // 若为方法未注册，尝试下一个前缀候选
        if (errMsg.includes('unknown bound method')) {
          continue;
        }
        // 若已命中绑定方法但后端执行抛错（如目录不存在），锁定前缀并向上抛出业务异常
        detectedServicePrefix = prefix;
        throw err;
      }
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
      throw err;
    }
  }

  if (window.location?.protocol === 'wails:' || window.wails?.Call?.ByName || window.go?.wails?.AppService) {
    throw new Error('桌面服务尚未就绪，请稍后重试');
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

  async openProjectWebsite(): Promise<void> {
    const res = await invokeWailsMethod<void>('OpenProjectWebsite');
    if (!res.called) window.open('https://github.com/ixugo/lark-studio', '_blank');
  },

  async listTaskLogs(id: string): Promise<TaskLog[]> {
    const res = await invokeWailsMethod<TaskLog[]>('ListTaskLogs', id);
    if (res.called && res.result) return res.result;
    return mockLogs[id] || [];
  },

  async listTaskSteps(taskId: string): Promise<TaskStep[]> {
    const res = await invokeWailsMethod<TaskStep[]>('ListTaskSteps', taskId);
    if (res.called && res.result) return res.result;
    return [];
  },

  async createTask(input: CreateTaskInput): Promise<Task> {
    const res = await invokeWailsMethod<Task>('CreateTask', input);
    if (res.called && res.result) return res.result;
    const newTask: Task = {
      id: `task_${Date.now()}`,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      input_path: input.input_path,
      output_dir: input.output_dir || `${input.input_path}_lark_studio`,
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

  async rerunTaskFromStep(id: string, fromStep: string): Promise<void> {
    const res = await invokeWailsMethod<void>('RerunTaskFromStep', id, fromStep);
    if (res.called) return;
    mockTasks = mockTasks.map((t) =>
      t.id === id ? { ...t, status: 1, current_step: fromStep, current_detail: `重跑节点: ${fromStep}` } : t
    );
  },

  async rerunTaskWithRecipe(id: string, opts: RerunTaskOptions): Promise<void> {
    const res = await invokeWailsMethod<void>('RerunTaskWithRecipe', id, opts);
    if (res.called) return;
    mockTasks = mockTasks.map((t) =>
      t.id === id
        ? {
            ...t,
            status: 1,
            current_step: opts.from_step,
            current_detail: `重跑节点: ${opts.from_step}`,
            ...(opts.mode ? { mode: opts.mode } : {}),
            ...(opts.source_lang ? { source_lang: opts.source_lang } : {}),
            ...(opts.speech_rate ? { speech_rate: opts.speech_rate } : {}),
            ...(opts.tts_voice ? { tts_voice: opts.tts_voice } : {}),
            ...(opts.tts_engine ? { tts_engine: opts.tts_engine } : {}),
            ...(opts.subtitle_output ? { subtitle_output: opts.subtitle_output } : {}),
            ...(opts.target_lang ? { target_lang: opts.target_lang } : {}),
            ...(opts.translator ? { translator: opts.translator } : {}),
            ...(opts.output_content ? { output_content: opts.output_content } : {}),
            ...(opts.recipe_name ? { recipe_name: opts.recipe_name } : {}),
          }
        : t
    );
  },

  async deleteTask(id: string): Promise<void> {
    const res = await invokeWailsMethod<void>('DeleteTask', id);
    mockTasks = mockTasks.filter((t) => t.id !== id);
    delete mockLogs[id];
    if (res.called) return;
  },

  // ─── 文件交互 ───────────────────────────────────────────
  async pickFiles(options?: { title?: string; multiple?: boolean; extensions?: string[] }): Promise<string[]> {
    // 1. 优先调用 Wails 3 原生 Dialogs.OpenFile
    if (typeof window !== 'undefined' && window.wails?.Dialogs?.OpenFile) {
      try {
        const dialogOpts: Record<string, unknown> = {
          CanChooseFiles: true,
          AllowsMultipleSelection: options?.multiple ?? true,
          Title: options?.title || '选择音视频或字幕文件',
          Filters: options?.extensions
            ? [{ DisplayName: '支持的文件', Pattern: options.extensions.map((ext) => (ext.startsWith('*') ? ext : `*.${ext}`)).join(';') }]
            : [
                { DisplayName: '音视频与字幕 (*.mp4,*.mkv,*.mov,*.mp3,*.wav,*.srt...)', Pattern: '*.mp4;*.mkv;*.mov;*.avi;*.webm;*.flv;*.mp3;*.wav;*.m4a;*.srt;*.vtt;*.ass' },
                { DisplayName: '所有文件 (*.*)', Pattern: '*.*' },
              ],
        };
        const selected = await window.wails.Dialogs.OpenFile(dialogOpts);
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
      input.multiple = options?.multiple ?? true;
      if (options?.extensions && options.extensions.length > 0) {
        input.accept = options.extensions.map((ext) => (ext.startsWith('.') ? ext : `.${ext}`)).join(',');
      } else {
        input.accept = 'video/*,audio/*,.srt,.vtt,.ass';
      }
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

  async mergeSubtitle(input: {
    video_path: string;
    primary_sub_path: string;
    secondary_sub_path?: string;
    output_dir?: string;
    output_content?: string;
    subtitle_output?: string;
  }): Promise<Task> {
    const res = await invokeWailsMethod<Task>('MergeSubtitle', input);
    if (res.called && res.result) return res.result;

    const newTask: Task = {
      id: `merge_${Date.now()}`,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      input_path: input.video_path,
      output_dir: input.output_dir || `${input.video_path}_lark_studio`,
      mode: 1,
      status: 1,
      current_step: 'burn',
      current_detail: '字幕压制合成',
      progress: 30,
      step_progress: 30,
      target_lang: 'zh-CN',
      source_lang: 'auto',
      translator: 'bing',
      output_content: input.output_content || 'source',
      tts_engine: 'edge',
      tts_voice: '',
      speech_rate: 1.0,
      subtitle_output: input.subtitle_output || 'soft',
    };
    mockTasks = [newTask, ...mockTasks];
    return newTask;
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

  // ─── 配方管理 (SQLite 存储) ──────────────────────────────
  async listRecipes(): Promise<import('../types').RecipeItem[]> {
    const res = await invokeWailsMethod<import('../types').RecipeItem[]>('ListRecipes');
    if (res.called && res.result) return res.result;
    try {
      const saved = localStorage.getItem('lark_custom_recipes') || localStorage.getItem('vdub_custom_recipes');
      if (saved) {
        const parsed = JSON.parse(saved);
        return parsed.map((p: any) => ({
          id: p.id,
          title: p.title,
          subtitle: p.subtitle,
          badge: p.badge || '我的配方',
          is_custom: true,
          do_sub: p.goals?.sub ?? true,
          do_translate: p.goals?.translate ?? true,
          do_dub: p.goals?.dub ?? true,
          do_video: p.goals?.video ?? true,
          target_lang: p.config?.targetLang,
          tts_voice: p.config?.ttsVoice,
          speech_rate: p.config?.speechRate,
          subtitle_output: p.config?.subtitleOutput,
        }));
      }
    } catch (e) {
      console.error(e);
    }
    return [];
  },

  async saveRecipe(recipe: Partial<import('../types').RecipeItem>): Promise<import('../types').RecipeItem> {
    const res = await invokeWailsMethod<import('../types').RecipeItem>('SaveRecipe', recipe);
    if (res.called && res.result) return res.result;
    const item: import('../types').RecipeItem = {
      id: recipe.id || `rcp_${Date.now()}`,
      title: recipe.title || '自定义配方',
      subtitle: recipe.subtitle || '',
      badge: recipe.badge || '我的配方',
      is_custom: true,
      do_sub: recipe.do_sub ?? true,
      do_translate: recipe.do_translate ?? true,
      do_dub: recipe.do_dub ?? true,
      do_video: recipe.do_video ?? true,
      target_lang: recipe.target_lang,
      tts_voice: recipe.tts_voice,
      speech_rate: recipe.speech_rate,
      subtitle_output: recipe.subtitle_output,
    };
    return item;
  },

  async deleteRecipe(id: string): Promise<void> {
    const res = await invokeWailsMethod<void>('DeleteRecipe', id);
    if (res.called) return;
  },

  // ─── Whisper 模型与运行时管理 ──────────────────────────────
  async listWhisperModels(): Promise<import('../types').WhisperModelItem[]> {
    const res = await invokeWailsMethod<import('../types').WhisperModelItem[]>('ListWhisperModels');
    if (res.called && res.result) return res.result;

    // 浏览器开发模式下的模拟列表
    return [
      { name: 'tiny', size: '75 MiB', desc: '最小最快，适合极速测试', downloaded: true, path: 'models/ggml-tiny.bin' },
      { name: 'base', size: '148 MiB', desc: '轻量首选，日常快速转写', downloaded: true, path: 'models/ggml-base.bin' },
      { name: 'small', size: '488 MiB', desc: '性价比极佳，兼顾速度与准确度', downloaded: false },
      { name: 'medium', size: '1.53 GiB', desc: '中文效果好，适合正式视频', downloaded: false },
      { name: 'large-v3-turbo', size: '1.62 GiB', desc: '最新旗舰 Turbo，极致精度与极佳速度推荐', downloaded: true, path: 'models/ggml-large-v3-turbo.bin' },
      { name: 'large-v3', size: '3.1 GiB', desc: '最高精度旗舰模型', downloaded: false },
    ];
  },

  async downloadWhisperModel(name: string): Promise<void> {
    const res = await invokeWailsMethod<void>('DownloadWhisperModel', name);
    if (res.called) return;
    console.log('触发模型下载:', name);
  },

  async deleteWhisperModel(name: string): Promise<void> {
    const res = await invokeWailsMethod<void>('DeleteWhisperModel', name);
    if (res.called) return;
    console.log('删除模型:', name);
  },

  async inspectWhisperRuntime(): Promise<import('../types').WhisperRuntimeInfo> {
    const res = await invokeWailsMethod<import('../types').WhisperRuntimeInfo>('InspectWhisperRuntime');
    if (res.called && res.result) return res.result;

    return {
      installed: true,
      binary: 'whisper-cli',
      version: 'whisper.cpp (v1.7.4)',
      acceleration: 'Metal (Apple Silicon)',
    };
  },

  async installWhisperRuntime(): Promise<void> {
    const res = await invokeWailsMethod<void>('InstallWhisperRuntime');
    if (res.called) return;
    console.log('触发运行时安装');
  },

  async setActiveWhisperModel(nameOrPath: string): Promise<void> {
    const res = await invokeWailsMethod<void>('SetActiveWhisperModel', nameOrPath);
    if (res.called) return;
    console.log('设为默认模型:', nameOrPath);
  },

  async testOpenAITranslate(baseUrl: string, apiKey: string, model: string): Promise<string> {
    const res = await invokeWailsMethod<string>('TestOpenAITranslate', baseUrl, apiKey, model);
    if (res.called && res.result) return res.result;

    // 浏览器开发模式下模拟
    await new Promise((r) => setTimeout(r, 600));
    if (!baseUrl) {
      throw new Error('API Base URL 不能为空');
    }
    return '测试连通成功！延迟 128ms';
  },

  async pickDirectory(): Promise<string> {
    const res = await invokeWailsMethod<string>('PickDirectory');
    if (res.called && res.result) return res.result;
    return '';
  },

  async pickFFmpegFile(): Promise<string> {
    const res = await invokeWailsMethod<string>('PickFFmpegFile');
    if (res.called && res.result) return res.result;
    return '';
  },

  async listRemoteModels(baseURL: string, apiKey: string): Promise<Array<{ id: string }>> {
    const res = await invokeWailsMethod<Array<{ id: string }>>('ListRemoteModels', baseURL, apiKey);
    if (res.called && res.result) return res.result;
    throw new Error('请在桌面应用中读取远程模型列表');
  },

  async getTTSCapabilities(baseURL: string, apiKey: string, model: string): Promise<import('../types').TTSCapabilities> {
    const res = await invokeWailsMethod<import('../types').TTSCapabilities>('GetTTSCapabilities', baseURL, apiKey, model);
    if (res.called && res.result) return res.result;
    throw new Error('请在桌面应用中读取语音服务能力');
  },

  async testConfiguredTTS(config: ConfigDTO['tts'], text: string): Promise<string> {
    const res = await invokeWailsMethod<string>('TestConfiguredTTS', config, text);
    if (res.called && res.result) return res.result;
    throw new Error('请在桌面应用中试听');
  },

  async testOpenAITTS(baseUrl: string, apiKey: string, model: string, voice: string, text: string): Promise<string> {
    const res = await invokeWailsMethod<string>('TestOpenAITTS', baseUrl, apiKey, model, voice, text);
    if (res.called && res.result) return res.result;
    throw new Error('TTS试听需要在桌面应用中运行');
  },

  async testEdgeTTS(voice: string, text: string): Promise<string> {
    const res = await invokeWailsMethod<string>('TestEdgeTTS', voice, text);
    if (res.called && res.result) return res.result;
    throw new Error('TTS试听需要在桌面应用中运行');
  },

  // ─── 系统信息与版本 ───────────────────────────────────────
  async getAppInfo(): Promise<AppInfo> {
    try {
      const res = await invokeWailsMethod<AppInfo>('GetAppInfo');
      if (res.called && res.result && res.result.build_version) {
        return res.result;
      }
    } catch {
      // ignore
    }
    return {
      app_name: 'Lark Studio',
      build_version: '0.1.0',
      platform: 'darwin',
      arch: 'arm64',
    };
  },

  async checkForUpdates(manual: boolean): Promise<UpdateInfo> {
    const res = await invokeWailsMethod<UpdateInfo>('CheckForUpdates', manual);
    if (!res.called || !res.result) throw new Error('Updates require the desktop app');
    return res.result;
  },

  async ignoreUpdate(version: string): Promise<void> {
    const res = await invokeWailsMethod<void>('IgnoreUpdate', version);
    if (!res.called) throw new Error('Updates require the desktop app');
  },

  async installUpdate(version: string): Promise<void> {
    const res = await invokeWailsMethod<void>('InstallUpdate', version);
    if (!res.called) throw new Error('Updates require the desktop app');
  },

  async getUpdateStatus(): Promise<UpdateStatus> {
    const res = await invokeWailsMethod<UpdateStatus>('GetUpdateStatus');
    if (!res.called || !res.result) throw new Error('Updates require the desktop app');
    return res.result;
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
