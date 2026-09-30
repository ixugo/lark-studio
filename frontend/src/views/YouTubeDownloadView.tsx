import React, { useEffect, useRef, useState } from 'react';
import { Download, Link2, Loader2, FolderOpen, Youtube, ChevronDown } from 'lucide-react';
import { buildRecipeCatalog, recipeUnavailableReason, resolveWorkflowMode, subtitleContent } from '../lib/workflowRecipes';
import { api } from '../lib/api';
import { useTranslation } from '../i18n';
import { RecipeItem, YouTubeInfo, YouTubeStatus } from '../types';

export const YouTubeDownloadView: React.FC<{ onTaskCreated: () => void }> = ({ onTaskCreated }) => {
  const { t } = useTranslation();
  const [link, setLink] = useState('');
  const [info, setInfo] = useState<YouTubeInfo | null>(null);
  const [height, setHeight] = useState(1080);
  const [status, setStatus] = useState<YouTubeStatus | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const [recipes, setRecipes] = useState<RecipeItem[]>([]);
  const [recipeId, setRecipeId] = useState('none');
  useEffect(() => { let active = true; api.listRecipes().then(items => { if (active) setRecipes(buildRecipeCatalog(items, t).filter(recipe => recipe.id !== 'custom')); }).catch(err => { if (active) setError(String(err)); }); return () => { active = false; }; }, [t]);
  const operation = useRef(false);
  const mounted = useRef(true);
  const restored = useRef(false);
  const navigated = useRef('');
  const busy = pending || ['verifying', 'converting', 'downloading', 'checking', 'inspecting', 'processing'].includes(status?.phase || '');

  useEffect(() => {
    mounted.current = true;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      try { const next = await api.getYouTubeDownload(); if (!stopped) {
          setStatus(next);
          if (next.task_id) {
            if (restored.current && navigated.current !== next.task_id) onTaskCreated();
            navigated.current=next.task_id;
          }
          if (!restored.current && next.video) { setInfo(next.video); setLink(next.video.url); setHeight(next.video.resolutions[0]); }
          restored.current = true;
        } }
      catch (err) { if (!stopped) setError(String(err)); }
      if (!stopped) timer = setTimeout(poll, 1500);
    };
    void poll();
    return () => { stopped = true; mounted.current = false; clearTimeout(timer); };
  }, []);

  const run = async (action: () => Promise<void>) => {
    if (operation.current) return;
    operation.current = true; setPending(true); setError('');
    try { await action(); }
    catch (err) { if (mounted.current) setError(String(err)); }
    finally { operation.current = false; if (mounted.current) setPending(false); }
  };
  const parse = () => run(async () => {
    setInfo(null);
    const result = await api.resolveYouTubeVideo(link.trim());
    if (mounted.current) { setInfo(result); setHeight(result.resolutions[0]); }
  });
  const start = () => run(async () => {
    if (!info) return;
    const recipe = recipes.find(item => item.id === recipeId);
    await api.startYouTubeDownload(info.url,height,recipe ? {mode:resolveWorkflowMode(recipe),recipe_name:recipe.title,source_lang:recipe.source_lang,target_lang:recipe.target_lang,
      translator:recipe.translate_service==='local'?'openai':recipe.translate_service,output_content:subtitleContent(recipe.do_translate),
      tts_engine:recipe.tts_engine,tts_voice:recipe.tts_voice,speech_rate:recipe.speech_rate,subtitle_output:recipe.do_video?recipe.subtitle_output:'file',input_path:''} : null);
    const next = await api.getYouTubeDownload();
    if (mounted.current) setStatus(next);
  });
  const downloading = ['converting', 'downloading', 'checking'].includes(status?.phase || '');
  const field = 'w-full h-10 min-h-10 rounded-xl border border-slate-200 dark:border-white/10 bg-white dark:bg-[#242427] px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-blue-500/30 disabled:opacity-50';

  return (
    <div className="h-full overflow-y-auto p-8">
      <div className="mx-auto max-w-3xl space-y-6">
        <header className="flex items-center gap-4">
          <div className="rounded-2xl bg-red-500/10 p-3 text-red-500"><Youtube size={28} /></div>
          <div><h1 className="text-2xl font-semibold tracking-tight">{t('youtube.title')}</h1></div>
        </header>
        <section className="rounded-2xl border border-slate-200 dark:border-white/10 bg-white/70 dark:bg-[#18181B] p-6 space-y-5">
          <label className="block space-y-2"><span className="flex items-center gap-2 text-sm font-medium"><Link2 size={16} />{t('youtube.link')}</span>
            <input type="url" maxLength={2048} disabled={busy} value={link} onChange={e => { setLink(e.target.value); setInfo(null); setError(''); }} placeholder="https://www.youtube.com/watch?v=…" className={field} />
          </label>
          <button disabled={busy || !link.trim()} onClick={parse} className="rounded-xl bg-blue-600 px-5 py-2.5 text-sm font-medium text-white disabled:opacity-50 flex items-center gap-2">{pending && <Loader2 size={16} className="animate-spin" />}{t(status?.phase === 'inspecting' ? 'youtube.parsing' : 'youtube.parse')}</button>
          {info && <div className="border-t border-slate-200 dark:border-white/10 pt-5 space-y-4">
            <div><h2 className="text-lg font-semibold select-text">{info.title}</h2></div>
            <fieldset className="space-y-2"><legend className="text-sm font-medium">{t('youtube.resolution')}</legend><div className="flex flex-wrap gap-2">{info.resolutions.filter(h => h>=360 && h<=1080).map(h => <button key={h} type="button" disabled={busy} aria-pressed={height===h} onClick={() => setHeight(h)} className={`rounded-xl border px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50 ${height===h?'border-blue-600 bg-blue-600 text-white':'border-slate-200 dark:border-white/10 hover:border-blue-400'}`}>{h}P</button>)}</div></fieldset>
            <p className="text-xs text-slate-500">{t('youtube.resolutionHint')}</p>
            <label className="block space-y-2"><span className="text-sm font-medium">{t('youtube.followUp')}</span><div className="relative"><select value={recipeId} disabled={busy} onChange={e => setRecipeId(e.target.value)} className={`${field} appearance-none pr-12`}><option value="none">{t('youtube.noProcessing')}</option><optgroup label={t('youtube.selectRecipe')}>{recipes.filter(r => !recipeUnavailableReason(r, 'video')).map(r => <option value={r.id} key={r.id}>{r.title}</option>)}</optgroup></select><ChevronDown size={18} className="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 text-slate-400" /></div></label>
            <div className="flex items-center gap-3">
              <button disabled={busy} onClick={start} className="rounded-xl bg-blue-600 px-5 py-2.5 text-sm font-medium text-white disabled:opacity-50 flex items-center gap-2"><Download size={16} />{t('youtube.download')}</button>
            </div>
          </div>}
        </section>
        {status && status.phase !== 'idle' && status.phase !== 'inspecting' && <section className="rounded-2xl border border-slate-200 dark:border-white/10 p-5 space-y-3">
          <div className="flex items-center justify-between gap-3"><span className="text-sm font-medium" role="status">{t(`youtube.${status?.phase || 'idle'}`)}</span>{(downloading || status?.phase === 'verifying') && <button disabled={pending} onClick={() => run(() => api.cancelYouTubeDownload())} className="text-sm text-red-500">{t('youtube.cancel')}</button>}</div>
          {downloading && <><progress max={100} value={status?.total && status.total > 0 ? status.percent : undefined} className="w-full h-2 accent-blue-600" /><p className="text-xs text-slate-500">{status?.total && status.total > 0 ? `${Math.floor(status.percent)}%` : `${t('youtube.unknownProgress')}: ${((status?.bytes || 0) / 1048576).toFixed(1)} MiB`}</p></>}
          {status?.phase === 'completed' && <><p className="text-sm break-all select-text">{status.path}</p><button onClick={() => run(() => api.openInFileManager(status.path))} className="flex items-center gap-2 text-sm text-blue-600"><FolderOpen size={16} />{t('youtube.open')}</button>
          </>}
        </section>}
        {(error || status?.error) && <p role="alert" className="text-sm text-red-500 whitespace-pre-wrap break-words select-text">{error || status?.error}</p>}
        <p className="text-xs text-slate-500">{t('youtube.notice')}</p>
      </div>
    </div>
  );
};
