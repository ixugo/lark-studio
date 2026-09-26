export type TaskMode = 1 | 2 | 3; // 1: 仅字幕, 2: 双语翻译, 3: 翻译配音

export type TaskStatus = 0 | 1 | 2 | 3 | 4; // 0: 待处理, 1: 进行中, 2: 已暂停, 3: 已完成, 4: 失败

export interface Task {
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
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped';
  detail: string;
  progress: number;
  started_at?: string;
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

export interface ConfigDTO {
  pipeline: {
    workers: number;
    whisper_mode: string;
    whisper_bin?: string;
    whisper_model: string;
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
