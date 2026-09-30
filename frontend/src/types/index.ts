export type TaskMode = 1 | 2 | 3 | 4 | 5 | 6; // 1: 仅字幕, 2: 双语翻译, 3: 翻译配音(1-2-3-4), 4: 仅AI配音/小说朗读(3), 5: 原文配音(1-3-4), 6: 文本翻译(2)

export type TaskStatus = 0 | 1 | 2 | 3 | 4; // 0: 待处理, 1: 进行中, 2: 已暂停, 3: 已完成, 4: 失败

export interface Task {
 original_name?: string;
 batch_dir?: string;
 result_path?: string;
  id: string;
  created_at: string;
  updated_at: string;
  input_path: string;
  output_dir: string;
  mode: TaskMode;
  status: TaskStatus;
  current_step: string;
  current_detail: string;
  progress: number;
  step_progress: number;
  error?: string;
  target_lang: string;
  source_lang: string;
  translator: string;
  output_content: string;
  tts_engine: string;
  tts_voice: string;
  speech_rate: number;
  subtitle_output: string;
  recipe_name?: string;
}

export interface TaskLog {
  id: string;
  task_id: string;
  created_at: string;
  level: 'info' | 'warn' | 'error' | 'success';
  step: string;
  message: string;
}

export interface TaskStep {
  id: string;
  task_id: string;
  name: string;
  status: number | string;
  detail: string;
  progress: number;
  started_at?: string;
  ended_at?: string;
  finished_at?: string;
  error?: string;
}

export interface CreateTaskInput {
  input_path: string;
  output_dir?: string;
  mode: TaskMode;
  source_lang?: string;
  target_lang?: string;
  translator?: string;
  output_content?: string;
  tts_engine?: string;
  tts_voice?: string;
  speech_rate?: number;
  subtitle_output?: string;
  recipe_name?: string;
}

export interface RerunTaskOptions {
  from_step: string;
  mode?: TaskMode;
  target_lang?: string;
  source_lang?: string;
  translator?: string;
  output_content?: string;
  tts_engine?: string;
  tts_voice?: string;
  speech_rate?: number;
  subtitle_output?: string;
  recipe_name?: string;
}

export interface ConfigDTO {
  pipeline: {
    workers: number;
    whisper_mode: string;
    whisper_bin?: string;
    whisper_model: string;
    asr_base_url?: string;
    asr_api_key?: string;
    asr_model?: string;
    ffmpeg_bin: string;
    default_output_dir?: string;
    default_target_lang: string;
    translate_prompt: string;
    max_speed_factor: number;
    translate_chunk_size: number;
    tts_workers: number;
    clean_intermediate: boolean;
    subtitle_output: string;
  };
  llm: {
    provider: string;
    base_url: string;
    api_key: string;
    model: string;
    deeplx_url: string;
  };
  tts: {
    type: string;
    voice: string;
    edge_voice?: string;
    openai_voice?: string;
    protocol?: string;
    language?: string;
    instructions?: string;
    base_url: string;
    api_key: string;
    model: string;
  };
  lip_sync: {
    enabled: boolean;
    base_url: string;
    api_key: string;
  };
  runtime: {
    debug: boolean;
    build_version: string;
    config_dir: string;
    config_path: string;
  };
}

export interface Term {
  id: number;
  glossary_id: number;
  text: string;
  translation: string;
  note?: string;
  created_at?: string;
}

export interface WhisperModelItem {
  name: string;
  kind?: 'asr' | 'vad';
  size: string;
  desc: string;
  downloaded: boolean;
  path?: string;
  downloading?: boolean;
  progress?: number;
  speed?: string;
}

export interface WhisperRuntimeInfo {
  installed: boolean;
  binary: string;
  path?: string;
  version: string;
  acceleration: string;
  installing?: boolean;
}

export interface RecipeItem {
  id: string;
  title: string;
  subtitle: string;
  badge: string;
  is_custom: boolean;
  do_sub: boolean;
  do_translate: boolean;
  do_dub: boolean;
  do_video: boolean;
  target_lang?: string;
  source_lang?: string;
  whisper_model?: string;
  translate_service?: string;
  tts_engine?: string;
  tts_voice?: string;
  speech_rate?: number;
  subtitle_output?: string;
  subtitle_style?: string;
  video_quality?: string;
  created_at?: string;
  updated_at?: string;
}

export interface AppInfo {
  app_name: string;
  build_version: string;
  platform: string;
  arch: string;
}

export interface TTSCapabilities {
  protocol: 'openai' | 'mlx';
  voices: Array<{ id: string; name: string }>;
  voice_source: 'remote' | 'qwen_builtin' | 'manual';
  instructions: boolean;
  languages: string[];
}

export interface UpdateInfo {
  version: string;
  notes: string;
  available: boolean;
  supported: boolean;
  reason: string;
}

export interface UpdateStatus {
  phase: 'idle' | 'downloading' | 'preparing' | 'restarting' | 'error';
  message: string;
  percent: number;
}

export interface YouTubeInfo { url: string; title: string; resolutions: number[]; }
export interface YouTubeStatus { task_id?: string; phase: 'processing' | 'idle' | 'verifying' | 'inspecting' | 'converting' | 'downloading' | 'checking' | 'completed' | 'failed' | 'cancelled'; percent: number; path: string; error: string; directory: string; verified: boolean; bytes: number; total: number; video?: YouTubeInfo; }
