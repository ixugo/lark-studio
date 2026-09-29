import { configFormUpdates } from '../lib/configForms';
import React, { useEffect, useState } from 'react';
import {
  Save,
  Cpu,
  Layers,
  FolderOpen,
  Film,
  Trash2,
  CheckCircle2,
  AlertCircle,
  ExternalLink,
  Languages,
  Volume2,
} from 'lucide-react';
import { api } from '../lib/api';
import { ConfigDTO } from '../types';
import { useTranslation } from '../i18n';

interface SettingsViewProps {
  onNavigate?: (tab: string) => void;
}

export const SettingsView: React.FC<SettingsViewProps> = ({ onNavigate }) => {
  const { locale } = useTranslation();
  const english = locale === 'en-US';
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.getConfig().then((cfg) => {
      const isWin = typeof navigator !== 'undefined' && navigator.platform?.toLowerCase().includes('win');
      const fallbackDocs = isWin ? 'C:\\Users\\Documents\\lark-studio' : '~/Documents/lark-studio';
      const patched: ConfigDTO = {
        ...cfg,
        pipeline: {
          ...cfg.pipeline,
          workers: cfg.pipeline?.workers ?? 2,
          default_target_lang: cfg.pipeline?.default_target_lang || 'zh-CN',
          ffmpeg_bin: cfg.pipeline?.ffmpeg_bin || 'ffmpeg',
          default_output_dir: cfg.pipeline?.default_output_dir || fallbackDocs,
          max_speed_factor: cfg.pipeline?.max_speed_factor ?? 1.2,
          translate_chunk_size: cfg.pipeline?.translate_chunk_size ?? 10,
          tts_workers: cfg.pipeline?.tts_workers ?? 2,
          clean_intermediate: cfg.pipeline?.clean_intermediate ?? false,
        },
        llm: {
          ...cfg.llm,
          provider: cfg.llm?.provider || 'bing',
          base_url: cfg.llm?.base_url || 'https://api.openai.com/v1',
          model: cfg.llm?.model || 'gpt-4o-mini',
        },
        tts: {
          ...cfg.tts,
          type: cfg.tts?.type || 'edge',
          voice: cfg.tts?.voice || 'zh-CN-XiaoxiaoNeural',
          model: cfg.tts?.model || 'tts-1',
        },
      };
      setConfig(patched);
    }).catch(console.error);
  }, []);

  const handleSelectFFmpeg = async () => {
    try {
      const filePath = await api.pickFFmpegFile();
      if (filePath && config) {
        setConfig({
          ...config,
          pipeline: { ...config.pipeline, ffmpeg_bin: filePath },
        });
      }
    } catch (e) {
      console.warn('选择文件失败', e);
    }
  };

  const handleSelectOutputDir = async () => {
    try {
      const dirPath = await api.pickDirectory();
      if (dirPath && config) {
        setConfig({
          ...config,
          pipeline: { ...config.pipeline, default_output_dir: dirPath },
        });
      }
    } catch (e) {
      console.warn('选择目录失败', e);
    }
  };

  const handleSave = async () => {
    if (!config) return;
    setLoading(true);
    setErrorMsg(null);
    try {
      await api.updateConfig(configFormUpdates('settings', config));
      setSavedSuccess(true);
      setTimeout(() => setSavedSuccess(false), 2500);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  if (!config) {
    return (
      <div className="h-screen flex items-center justify-center text-xs text-slate-400">
        {english ? 'Loading global settings...' : '正在读取系统全局配置...'}
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6 select-none bg-[#F5F5F7] dark:bg-[#121215]">
      <div className="max-w-4xl w-full mx-auto space-y-6 pt-2 pb-16">
        {/* 顶部标题 */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
              {english ? 'Global Settings' : '全局系统设置'}
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {english
                ? 'Manage task scheduling, media tools, output folders, and temporary file cleanup.'
              : '管理任务调度、多媒体工具、输出目录及临时文件清理。'}
            </p>
          </div>
        </div>

        {/* 提示条 */}
        {savedSuccess && (
          <div className="p-3.5 bg-emerald-500/10 border border-emerald-500/20 rounded-xl flex items-center space-x-2 text-xs text-emerald-600 dark:text-emerald-400 animate-in fade-in">
            <CheckCircle2 size={16} />
            <span>{english ? 'Global settings saved and applied.' : '全局配置已成功保存并实时生效！'}</span>
          </div>
        )}

        {errorMsg && (
          <div className="p-3.5 bg-rose-50 border border-rose-200 dark:bg-rose-500/10 dark:border-rose-500/30 rounded-xl flex items-center gap-2 text-xs text-rose-600 dark:text-rose-300 animate-in fade-in">
            <AlertCircle size={16} />
            <span>{english ? 'Save failed:' : '保存失败:'} {errorMsg}</span>
          </div>
        )}

        {/* 引擎快速直达卡片 (Apple 分组卡片) */}
        <div className="grid grid-cols-3 gap-4">
          <button
            type="button"
            onClick={() => onNavigate?.('translation-engine')}
            className="p-4 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl text-left hover:border-blue-500/40 transition-all flex flex-col justify-between group active:scale-[0.98]"
          >
            <div className="flex items-center justify-between w-full">
              <div className="w-8 h-8 rounded-xl bg-blue-50 dark:bg-blue-500/15 flex items-center justify-center text-blue-600 dark:text-blue-400">
                <Languages size={18} />
              </div>
              <ExternalLink size={14} className="text-slate-300 group-hover:text-blue-500 transition-colors" />
            </div>
            <div className="mt-3">
              <div className="text-xs font-bold text-slate-800 dark:text-white">{english ? 'Translation' : '翻译引擎'}</div>
              <div className="text-[11px] text-slate-400 mt-0.5">
                {english ? 'Current:' : '当前:'} {config.llm.provider || 'bing'}
              </div>
            </div>
          </button>

          <button
            type="button"
            onClick={() => onNavigate?.('tts-engine')}
            className="p-4 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl text-left hover:border-blue-500/40 transition-all flex flex-col justify-between group active:scale-[0.98]"
          >
            <div className="flex items-center justify-between w-full">
              <div className="w-8 h-8 rounded-xl bg-indigo-50 dark:bg-indigo-500/15 flex items-center justify-center text-indigo-600 dark:text-indigo-400">
                <Volume2 size={18} />
              </div>
              <ExternalLink size={14} className="text-slate-300 group-hover:text-indigo-500 transition-colors" />
            </div>
            <div className="mt-3">
              <div className="text-xs font-bold text-slate-800 dark:text-white">{english ? 'Text to Speech' : '语音合成'}</div>
              <div className="text-[11px] text-slate-400 mt-0.5">
                {english ? 'Current:' : '当前:'} {config.tts.type || 'edge'}
              </div>
            </div>
          </button>

          <button
            type="button"
            onClick={() => onNavigate?.('asr-engine')}
            className="p-4 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl text-left hover:border-blue-500/40 transition-all flex flex-col justify-between group active:scale-[0.98]"
          >
            <div className="flex items-center justify-between w-full">
              <div className="w-8 h-8 rounded-xl bg-cyan-50 dark:bg-cyan-500/15 flex items-center justify-center text-cyan-600 dark:text-cyan-400">
                <Cpu size={18} />
              </div>
              <ExternalLink size={14} className="text-slate-300 group-hover:text-cyan-500 transition-colors" />
            </div>
            <div className="mt-3">
              <div className="text-xs font-bold text-slate-800 dark:text-white">{english ? 'Speech Recognition' : '语音识别'}</div>
              <div className="text-[11px] text-slate-400 mt-0.5">
                {english ? 'Current:' : '当前:'} {config.pipeline.whisper_mode || 'whisper-cpp'}
              </div>
            </div>
          </button>
        </div>

        {/* 流水线并发与硬件调度 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-6 space-y-5">
          <div className="flex items-center space-x-2 border-b border-slate-100 dark:border-white/5 pb-3">
            <Layers className="text-blue-500" size={18} />
            <h3 className="text-sm font-semibold text-slate-800 dark:text-white">
              {english ? 'Pipeline Scheduling and Concurrency' : '流水线调度与任务并发'}
            </h3>
          </div>

          <div className="grid grid-cols-2 gap-6">
            <div>
              <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                {english ? 'Concurrent tasks' : '并发任务数'}
              </label>
              <input
                type="number"
                min="1"
                max="8"
                value={config.pipeline.workers}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    pipeline: {
                      ...config.pipeline,
                      workers: parseInt(e.target.value) || 1,
                    },
                  })
                }
                className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
              />
              <p className="text-[11px] text-slate-400 mt-1">{english ? 'Recommended: 2–4, depending on your CPU cores.' : '推荐 2~4，根据 CPU 核心数适度调配'}</p>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                {english ? 'Default target language' : '默认目标语言'}
              </label>
              <input
                type="text"
                value={config.pipeline.default_target_lang}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    pipeline: {
                      ...config.pipeline,
                      default_target_lang: e.target.value,
                    },
                  })
                }
                className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
              />
              <p className="text-[11px] text-slate-400 mt-1">{english ? 'Used when no target language is specified (e.g. zh-CN, en).' : '缺省将自动使用该目标语言（如 zh-CN, en）'}</p>
            </div>
          </div>
        </div>

        {/* 媒体处理工具与路径 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-6 space-y-5">
          <div className="flex items-center space-x-2 border-b border-slate-100 dark:border-white/5 pb-3">
            <Film className="text-indigo-500" size={18} />
            <h3 className="text-sm font-semibold text-slate-800 dark:text-white">
              {english ? 'Media Tools and Storage' : '多媒体工具与存储路径'}
            </h3>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                {english ? 'FFmpeg executable path' : 'FFmpeg 可执行文件路径'}
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder={english ? 'Leave blank to use the bundled macOS version or ffmpeg on PATH.' : '留空即优先使用 macOS Bundle 内嵌或系统 PATH 中的 ffmpeg'}
                  value={config.pipeline.ffmpeg_bin}
                  onChange={(e) =>
                    setConfig({
                      ...config,
                      pipeline: {
                        ...config.pipeline,
                        ffmpeg_bin: e.target.value,
                      },
                    })
                  }
                  className="flex-1 h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                />
                <button
                  type="button"
                  onClick={handleSelectFFmpeg}
                  className="px-3.5 h-10 rounded-xl bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-slate-200 text-xs font-semibold border border-slate-200 dark:border-white/10 flex items-center gap-1.5 transition-colors shrink-0"
                >
                  <FolderOpen size={14} />
                  <span>{english ? 'Browse' : '浏览'}</span>
                </button>
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                {english ? 'Default video output folder' : '默认视频输出目录'}
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder={english ? 'Default documents folder' : '默认文档目录'}
                  value={config.pipeline.default_output_dir}
                  onChange={(e) =>
                    setConfig({
                      ...config,
                      pipeline: {
                        ...config.pipeline,
                        default_output_dir: e.target.value,
                      },
                    })
                  }
                  className="flex-1 h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                />
                <button
                  type="button"
                  onClick={handleSelectOutputDir}
                  className="px-3.5 h-10 rounded-xl bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-slate-200 text-xs font-semibold border border-slate-200 dark:border-white/10 flex items-center gap-1.5 transition-colors shrink-0"
                >
                  <FolderOpen size={14} />
                  <span>{english ? 'Choose folder' : '选择目录'}</span>
                </button>
              </div>
            </div>

            <div className="pt-2">
              <label className="flex items-center justify-between cursor-pointer p-3.5 rounded-xl bg-slate-50/70 dark:bg-white/[0.03] border border-slate-200/70 dark:border-white/5 hover:border-slate-300 transition-colors">
                <div className="space-y-0.5">
                  <div className="text-xs font-semibold text-slate-800 dark:text-slate-200 flex items-center gap-1.5">
                    <Trash2 size={14} className="text-amber-500" />
                    <span>{english ? 'Delete temporary files when processing finishes' : '处理完成后自动删除中间产物'}</span>
                  </div>
                  <div className="text-[11px] text-slate-400">
                    {english
                      ? 'When enabled, temporary raw.mp3 audio and segment files are removed after a successful run to save disk space.'
                      : '开启后，流水线成功结束后将自动清理临时的 raw.mp3 音频和切片中间文件，节省磁盘空间'}
                  </div>
                </div>
                <input
                  type="checkbox"
                  checked={config.pipeline.clean_intermediate}
                  onChange={(e) =>
                    setConfig({
                      ...config,
                      pipeline: {
                        ...config.pipeline,
                        clean_intermediate: e.target.checked,
                      },
                    })
                  }
                  className="w-4 h-4 rounded text-blue-600 focus:ring-blue-500 border-slate-300 cursor-pointer"
                />
              </label>
            </div>
          </div>
        </div>
      </div>

      {/* 悬浮大圆角胶囊底栏 (符合 Apple Design 规范，彻底告别直角铺满) */}
      <div className="sticky bottom-4 mx-auto max-w-3xl w-full z-30 px-4">
        <div className="bg-white/90 dark:bg-[#202024]/90 backdrop-blur-2xl border border-slate-200/90 dark:border-white/10 rounded-2xl px-6 py-3 flex items-center justify-between transition-all">
          <div className="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
            <span>{english ? 'Changes are saved locally and take effect immediately.' : '配置修改后即刻写入本地并热重载生效'}</span>
          </div>
          <button
            type="button"
            onClick={handleSave}
            disabled={loading}
            className="px-5 py-2 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white transition-all flex items-center gap-1.5 active:scale-[0.98] disabled:opacity-50"
          >
            <Save size={14} />
            <span>{loading ? (english ? 'Saving...' : '正在保存...') : (english ? 'Save settings' : '保存全局设置')}</span>
          </button>
        </div>
      </div>
    </div>
  );
};
