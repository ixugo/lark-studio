import { RecipeItem, Task, TaskMode } from '../types';
import { TtsEngine } from './ttsVoices';

// 资源分类类型
export type ResourceType = 'video' | 'audio' | 'text';

const VIDEO_EXTS = ['mp4', 'mkv', 'mov', 'avi', 'webm', 'flv'];
const AUDIO_EXTS = ['mp3', 'wav', 'm4a', 'aac', 'flac', 'ogg'];
const TEXT_EXTS = ['txt', 'srt', 'vtt'];

export function detectResourceType(filePath: string): ResourceType | null {
  const ext = filePath.split('.').pop()?.toLowerCase();
  if (!ext) return null;
  if (VIDEO_EXTS.includes(ext)) return 'video';
  if (AUDIO_EXTS.includes(ext)) return 'audio';
  if (TEXT_EXTS.includes(ext)) return 'text';
  return null;
}

// 工作流快捷预设模板
export interface WorkflowPreset {
  id: string;
  title: string;
  subtitle: string;
  badge: string;
  isCustom?: boolean;
  goals: {
    sub: boolean;
    translate: boolean;
    dub: boolean;
    video: boolean;
  };
  config?: {
    sourceLang?: string;
    whisperModel?: string;
    translateService?: string;
    ttsEngine?: TtsEngine;
    targetLang?: string;
    ttsVoice?: string;
    speechRate?: number;
    subtitleOutput?: string;
  };
}

export const BUILTIN_PRESETS: WorkflowPreset[] = [
  {
    id: 'dub_full',
    title: '视频 → 译文配音成片',
    subtitle: '全自动听写、翻译、AI配音、原声伴奏保留并秒级合成出片',
    badge: '全流程译制',
    goals: { sub: true, translate: true, dub: true, video: true },
  },
  {
    id: 'direct_dub',
    title: '视频 → 原文配音成片',
    subtitle: '原文听写直接配音成片，跳过文本翻译',
    badge: '原文重配',
    goals: { sub: true, translate: false, dub: true, video: true },
  },
  {
    id: 'bilingual_sub',
    title: '视频/音频 → 双语字幕',
    subtitle: '听写并翻译字幕，不配音不成片',
    badge: '双语字幕',
    goals: { sub: true, translate: true, dub: false, video: false },
  },
  {
    id: 'text_translate',
    title: '纯文本 → 智能翻译',
    subtitle: '纯文本或SRT文本直接翻译为目标语言',
    badge: '文本翻译',
    goals: { sub: false, translate: true, dub: false, video: false },
  },
  {
    id: 'text_dub',
    title: '纯文本 → AI朗读配音',
    subtitle: '直接将文本朗读配音为自然高质量音频',
    badge: '语音合成',
    goals: { sub: false, translate: false, dub: true, video: false },
  },
  {
    id: 'custom',
    title: '自定义智能流程',
    subtitle: '自由开启或关闭各个处理流水线阶段',
    badge: '自由组合',
    goals: { sub: true, translate: true, dub: true, video: true },
  },
];

export const getPresetDisplay = (preset: WorkflowPreset, t: (key: string, fallback?: string) => string) => {
    if (preset.isCustom) {
      return {
        title: preset.title,
        subtitle: preset.subtitle,
        badge: preset.badge || t('dashboard.badgeMyRecipe', '我的配方'),
      };
    }
    switch (preset.id) {
      case 'dub_full':
        return {
          title: t('dashboard.presetDubFull', '视频 → 译文配音成片'),
          subtitle: t('dashboard.presetDubFullDesc', '全自动听写、翻译、AI配音、原声伴奏保留并秒级合成出片'),
          badge: t('dashboard.badgeFullDub', '全流程译制'),
        };
      case 'direct_dub':
        return {
          title: t('dashboard.presetDirectDub', '视频 → 原文配音成片'),
          subtitle: t('dashboard.presetDirectDubDesc', '原文听写直接配音成片，跳过文本翻译'),
          badge: t('dashboard.badgeDirectDub', '原文重配'),
        };
      case 'bilingual_sub':
        return {
          title: t('dashboard.presetBilingualSub', '视频/音频 → 双语字幕'),
          subtitle: t('dashboard.presetBilingualSubDesc', '听写并翻译字幕，不配音不成片'),
          badge: t('dashboard.badgeBilingualSub', '双语字幕'),
        };
      case 'text_translate':
        return {
          title: t('dashboard.presetTextTranslate', '纯文本 → 智能翻译'),
          subtitle: t('dashboard.presetTextTranslateDesc', '纯文本或SRT文本直接翻译为目标语言'),
          badge: t('dashboard.badgeTextTranslate', '文本翻译'),
        };
      case 'text_dub':
        return {
          title: t('dashboard.presetTextDub', '纯文本 → AI朗读配音'),
          subtitle: t('dashboard.presetTextDubDesc', '直接将文本朗读配音为自然高质量音频'),
          badge: t('dashboard.badgeTextDub', '语音合成'),
        };
      case 'custom':
      default:
        return {
          title: t('dashboard.presetCustom', '自定义智能流程'),
          subtitle: t('dashboard.presetCustomDesc', '自由开启或关闭各个处理流水线阶段'),
          badge: t('dashboard.badgeCustom', '自由组合'),
        };
    }
  };


export function buildRecipeCatalog(custom: RecipeItem[], t?: (key: string, fallback?: string) => string): RecipeItem[] {
  return [...BUILTIN_PRESETS.map(preset => ({
    id: preset.id,
    ...(t ? getPresetDisplay(preset, t) : { title: preset.title, subtitle: preset.subtitle, badge: preset.badge }),
    is_custom: false,
    do_sub: preset.goals.sub,
    do_translate: preset.goals.translate,
    do_dub: preset.goals.dub,
    do_video: preset.goals.video,
  })), ...custom];
}

export function recipeUnavailableReason(recipe: Pick<RecipeItem, 'do_sub' | 'do_video'>, resource: ResourceType | null): string {
  if ((resource === 'video' || resource === 'audio') && !recipe.do_sub) return '视频和音频必须先经过听写转录，不能使用纯文本配方';
  if (resource === 'text' && recipe.do_sub) return '文本已包含原文，请选择文本翻译或朗读配方';
  if ((resource === 'text' || resource === 'audio') && recipe.do_video) return '当前资源没有视频画面，请选择不含成片的配方';
  return '';
}

export function resolveWorkflowMode(recipe: Pick<RecipeItem, 'do_sub' | 'do_translate' | 'do_dub' | 'do_video'>): TaskMode {
  if (recipe.do_sub && !recipe.do_translate && recipe.do_dub && recipe.do_video) return 5;
  if (!recipe.do_sub && !recipe.do_translate && recipe.do_dub) return 4;
  if (!recipe.do_sub && recipe.do_translate && !recipe.do_dub) return 6;
  if (recipe.do_dub && recipe.do_video) return 3;
  if (recipe.do_translate) return 2;
  return 1;
}

export function subtitleContent(translate: boolean): 'source' | 'translated' {
  return translate ? 'translated' : 'source';
}

// 保留任务现有配置，不因打开重跑窗口就替用户换配方。
export function currentTaskRecipe(task: Task, t: (key: string, fallback?: string) => string): RecipeItem {
  const preset = buildRecipeCatalog([], t).find(recipe => resolveWorkflowMode(recipe) === task.mode);
  const title = task.recipe_name || preset?.title || t('rerunModal.recipeTranscribeOnly', '仅提取原文字幕');
  return {
    id: 'task-recipe', title, subtitle: preset?.subtitle || '', badge: '', is_custom: false,
    do_sub: ![4, 6].includes(task.mode),
    do_translate: [2, 3, 6].includes(task.mode),
    do_dub: [3, 4, 5].includes(task.mode),
    do_video: [3, 5].includes(task.mode),
    source_lang: task.source_lang, target_lang: task.target_lang,
    translate_service: task.translator, tts_engine: task.tts_engine,
    tts_voice: task.tts_voice, speech_rate: task.speech_rate, subtitle_output: task.subtitle_output,
  };
}

export function rerunRecipeCatalog(task: Task, custom: RecipeItem[], t: (key: string, fallback?: string) => string): RecipeItem[] {
  const recipes = buildRecipeCatalog(custom, t);
  const taskRecipe = currentTaskRecipe(task, t);
  return recipes.some(recipe => recipe.title === taskRecipe.title) ? recipes : [taskRecipe, ...recipes];
}
