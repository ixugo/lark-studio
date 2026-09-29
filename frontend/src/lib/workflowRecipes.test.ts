import { describe, expect, it } from 'vitest';
import { buildRecipeCatalog, BUILTIN_PRESETS, recipeUnavailableReason, resolveWorkflowMode, subtitleContent, detectResourceType, rerunRecipeCatalog, currentTaskRecipe } from './workflowRecipes';
import { RecipeItem, Task } from '../types';

describe('共用工作流配方', () => {
  it('没有用户配方时仍能选择所有系统配方，用户配方保留原配置', () => {
    expect(buildRecipeCatalog([]).map(r => r.id)).toEqual(BUILTIN_PRESETS.map(r => r.id));
    const user: RecipeItem = { id: 'user', title: '我的字幕', subtitle: '', badge: '', is_custom: true, do_sub: true, do_translate: true, do_dub: false, do_video: false, translate_service: 'google', source_lang: 'ja' };
    expect(buildRecipeCatalog([user]).at(-1)).toEqual(user);
  });
  it('视频和音频不能选择跳过听写的纯文本配方', () => {
    const catalog = buildRecipeCatalog([]);
    for (const resource of ['video', 'audio'] as const) {
      expect(recipeUnavailableReason(catalog.find(r => r.id === 'text_translate')!, resource)).not.toBe('');
      expect(recipeUnavailableReason(catalog.find(r => r.id === 'text_dub')!, resource)).not.toBe('');
      expect(recipeUnavailableReason(catalog.find(r => r.id === 'bilingual_sub')!, resource)).toBe('');
    }
    expect(recipeUnavailableReason(catalog.find(r => r.id === 'text_translate')!, 'text')).toBe('');
    expect(recipeUnavailableReason(catalog.find(r => r.id === 'dub_full')!, 'text')).not.toBe('');
  });
  it('纯文本翻译、听写翻译、原文配音解析为各自的执行模式', () => {
    const catalog = buildRecipeCatalog([]);
    expect(resolveWorkflowMode(catalog.find(r => r.id === 'text_translate')!)).toBe(6);
    expect(resolveWorkflowMode(catalog.find(r => r.id === 'bilingual_sub')!)).toBe(2);
    expect(resolveWorkflowMode(catalog.find(r => r.id === 'direct_dub')!)).toBe(5);
  });
  it('听写固定原文，翻译固定译文，不沿用旧的输出选择', () => {
    expect(subtitleContent(false)).toBe('source');
    expect(subtitleContent(true)).toBe('translated');
    expect(detectResourceType('/tmp/input.MP4')).toBe('video');
  });
  it('重跑显示实际配方名，已在目录中的配方不重复，旧任务按流程命名', () => {
    const t = (_key: string, fallback?: string) => fallback || '';
    const task = { mode: 2, recipe_name: '视频/音频 → 双语字幕' } as Task;
    const catalog = rerunRecipeCatalog(task, [], t);
    expect(catalog.filter(recipe => recipe.title === task.recipe_name)).toHaveLength(1);
    expect(catalog.some(recipe => recipe.title === '当前任务配方')).toBe(false);
    expect(currentTaskRecipe({ ...task, recipe_name: '' }, t).title).toBe('视频/音频 → 双语字幕');
    expect(currentTaskRecipe({ ...task, recipe_name: '用户命名的配方' }, t).title).toBe('用户命名的配方');
  });
});
