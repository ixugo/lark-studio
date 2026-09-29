import React, { useEffect, useState, useCallback } from 'react';
import {
  Cloud,
  CheckCircle2,
  AlertCircle,
  Save,
  Check,
  FolderOpen,
  ShieldCheck,
  Download,
  Trash2,
  Sparkles,
  Terminal,
  ChevronDown,
  ChevronRight,
  Loader2,
} from 'lucide-react';
import { api } from '../lib/api';
import { RefreshButton } from '../components/RefreshButton';
import { ModelDownloadProgress, initialModelDownloadProgress, listenModelDownload } from '../lib/modelDownload';
import { ConfigDTO, WhisperModelItem, WhisperRuntimeInfo } from '../types';
import { useTranslation } from '../i18n';

interface AsrEngineItem {
  id: 'whisper-cpp' | 'openai';
  name: string;
  tagKey: string;
  descKey?: string;
  icon: React.ReactNode;
  badge?: string;
}

const ASR_ENGINES: AsrEngineItem[] = [
  {
    id: 'whisper-cpp',
    name: 'Whisper.cpp',
    tagKey: 'asr.offlineAcceleration',
    icon: <ShieldCheck size={18} className="text-blue-500" />,
    badge: '推荐',
  },
  {
    id: 'openai',
    name: 'OpenAI 兼容',
    tagKey: 'asr.openaiCompatible',
    descKey: 'asr.openaiDesc',
    icon: <Cloud size={18} className="text-indigo-500" />,
  },
];

export const AsrEngineView: React.FC = () => {
  const { locale, t } = useTranslation();
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [activeEngine, setActiveEngine] = useState<'whisper-cpp' | 'openai'>('whisper-cpp');
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  // Whisper 运行时与模型状态
  const [runtime, setRuntime] = useState<WhisperRuntimeInfo | null>(null);
  const [runtimeInstalling, setRuntimeInstalling] = useState(false);
  const [models, setModels] = useState<WhisperModelItem[]>([]);
  const [downloadingModel, setDownloadingModel] = useState<string | null>(null);
  const [downloadProgress, setDownloadProgress] = useState<ModelDownloadProgress | null>(null);
  const [deletingModelName, setDeletingModelName] = useState<string | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);

  // 加载数据
  const loadRuntime = useCallback(async () => {
    try {
      const info = await api.inspectWhisperRuntime();
      setRuntime(info);
    } catch (err) {
      console.warn('获取 Whisper 运行时信息失败:', err);
    }
  }, []);

  const loadModels = useCallback(async () => {
    try {
      const list = await api.listWhisperModels();
      setModels(list);
    } catch (err) {
      console.warn('获取 Whisper 模型列表失败:', err);
    }
  }, []);

  useEffect(() => {
    api.getConfig().then((cfg) => {
      const legacyPipeline = cfg.pipeline as ConfigDTO['pipeline'] & { WhisperModel?: string };
      const currentEngine: 'whisper-cpp' | 'openai' = cfg.pipeline.whisper_mode === 'openai' ? 'openai' : 'whisper-cpp';
      const normalizedConfig = {
        ...cfg,
        pipeline: {
          ...cfg.pipeline,
          whisper_model: cfg.pipeline.whisper_model || legacyPipeline.WhisperModel || '',
          whisper_mode: currentEngine,
        },
      };
      setConfig(normalizedConfig);
      setActiveEngine(currentEngine);
    }).catch(console.error);

    loadRuntime();
    loadModels();
  }, [loadRuntime, loadModels]);

  // 监听后端下载与安装进度广播事件
  useEffect(() => {
    const unsubProgress = listenModelDownload({
      progress: (model, progress) => {
        setDownloadingModel(model);
        setDownloadProgress(progress);
      },
      done: () => {
        setDownloadingModel(null);
        setDownloadProgress(null);
        loadModels();
      },
      error: (message) => {
        setDownloadingModel(null);
        setDownloadProgress(null);
        setErrorMsg(message);
      },
    });

    const unsubRuntime = api.onEvent('whisper:runtime_progress', (data: any) => {
      if (!data) return;
      if (data.status === 'done') {
        setRuntimeInstalling(false);
        loadRuntime();
      } else if (data.status === 'error') {
        setRuntimeInstalling(false);
        setErrorMsg(`运行时安装失败: ${data.error || '未知错误'}`);
      }
    });

    return () => {
      unsubProgress();
      unsubRuntime();
    };
  }, [loadModels, loadRuntime]);

  // 安装 Whisper.cpp 运行时
  const handleInstallRuntime = async () => {
    setRuntimeInstalling(true);
    setErrorMsg(null);
    try {
      await api.installWhisperRuntime();
      setTimeout(() => {
        loadRuntime();
        setRuntimeInstalling(false);
      }, 3000);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
      setRuntimeInstalling(false);
    }
  };

  const handleRefreshRuntime = async () => {
    await Promise.all([loadRuntime(), loadModels()]);
  };

  // 立即开始下载模型，进度由后端实际传输事件更新。
  const handleDownloadModel = async (name: string) => {
    setErrorMsg(null);
    setDownloadingModel(name);
    setDownloadProgress(initialModelDownloadProgress);
    try {
      await api.downloadWhisperModel(name);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
      setDownloadingModel(null);
      setDownloadProgress(null);
    }
  };

  // 删除已下载的模型
  const handleDeleteModel = async (name: string) => {
    setDeletingModelName(null);
    // 乐观更新
    setModels((prev) =>
      prev.map((m) => (m.name === name ? { ...m, downloaded: false, size_str: '' } : m))
    );
    try {
      await api.deleteWhisperModel(name);
      await loadModels();
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
      await loadModels();
    }
  };

  // 激活使用此模型
  const handleActivateModel = async (model: WhisperModelItem) => {
    try {
      const targetPath = model.path || model.name;
      await api.setActiveWhisperModel(targetPath);
      if (config) {
        setConfig({
          ...config,
          pipeline: {
            ...config.pipeline,
            whisper_model: targetPath,
          },
        });
      }
      await loadModels();
      setSavedSuccess(true);
      setTimeout(() => setSavedSuccess(false), 2000);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
    }
  };

  // 手动选择模型文件
  const handleSelectModelFile = async () => {
    try {
      const wails = (window as any)?.wails;
      if (wails?.Dialogs?.OpenFile) {
        const filePath = await wails.Dialogs.OpenFile({
          Title: t('asr.selectModelTitle'),
          Filters: [{ DisplayName: 'Whisper Model (*.bin)', Pattern: '*.bin;*.ggml' }],
        });
        if (filePath && config) {
          const pathStr = Array.isArray(filePath) ? filePath[0] : filePath;
          if (pathStr) {
            setConfig({
              ...config,
              pipeline: { ...config.pipeline, whisper_model: pathStr },
            });
          }
        }
      }
    } catch (e) {
      console.warn('打开文件选择器失败', e);
    }
  };

  // 手动选择 whisper-cli 可执行文件
  const handleSelectBinFile = async () => {
    try {
      const wails = (window as any)?.wails;
      if (wails?.Dialogs?.OpenFile) {
        const filePath = await wails.Dialogs.OpenFile({
          Title: '选择 whisper-cli 可执行程序',
        });
        if (filePath && config) {
          const pathStr = Array.isArray(filePath) ? filePath[0] : filePath;
          if (pathStr) {
            setConfig({
              ...config,
              pipeline: { ...config.pipeline, whisper_bin: pathStr },
            });
          }
        }
      }
    } catch (e) {
      console.warn('打开文件选择器失败', e);
    }
  };

  // 保存全局配置
  const handleSave = async (engineToSet?: string) => {
    if (!config) return;
    setLoading(true);
    setErrorMsg(null);
    try {
      const targetMode = engineToSet || config.pipeline.whisper_mode || activeEngine;
      const updatedConfig = {
        ...config,
        pipeline: {
          ...config.pipeline,
          whisper_mode: targetMode,
        },
      };

      await api.updateConfig({
        pipeline: updatedConfig.pipeline,
        llm: updatedConfig.llm,
        tts: updatedConfig.tts,
      });

      setConfig(updatedConfig);
      setSavedSuccess(true);
      setTimeout(() => setSavedSuccess(false), 2500);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  const handleSetDefault = (engineId: 'whisper-cpp' | 'openai') => {
    if (!config) return;
    setConfig({
      ...config,
      pipeline: {
        ...config.pipeline,
        whisper_mode: engineId,
      },
    });
    handleSave(engineId);
  };

  if (!config) {
    return (
      <div className="h-screen flex items-center justify-center text-xs text-slate-400">
        <Loader2 className="animate-spin mr-2" size={16} />
        {t('common.loading')}
      </div>
    );
  }

  const isCurrentDefault = (config.pipeline.whisper_mode || 'whisper-cpp').toLowerCase() === activeEngine;
  const activeModelValue = (config.pipeline.whisper_model || '').toLowerCase();

  const getModelDesc = (name: string, fallback: string) => {
    const descMapZh: Record<string, string> = {
      tiny: '最小最快，适合极速测试与验证',
      base: '轻量首选，日常快速转写',
      small: '性价比极佳，兼顾速度与准确度',
      medium: '中文转写效果优异，适合长视频',
      'large-v3-turbo': '最新旗舰 Turbo，极致精度与超高速度推荐',
      'large-v3': '最高精度旗舰模型，适合高难专业术语转写',
    };
    const descMapEn: Record<string, string> = {
      tiny: 'Smallest & fastest, ideal for rapid testing',
      base: 'Lightweight & fast for everyday transcription',
      small: 'Great balance between speed and precision',
      medium: 'High quality for diverse audio & long videos',
      'large-v3-turbo': 'Latest flagship Turbo with high accuracy and speed',
      'large-v3': 'Maximum accuracy flagship for complex audio',
    };
    if (locale === 'en-US') {
      return descMapEn[name.toLowerCase()] || fallback;
    }
    return descMapZh[name.toLowerCase()] || fallback;
  };

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6 select-none bg-[#F5F5F7] dark:bg-[#121215]">
      <div className="max-w-5xl w-full mx-auto space-y-6 pt-2 pb-16">
        {/* 顶部标题区 */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
                {t('asr.title')}
              </h2>
              <span className="px-2 py-0.5 text-[11px] font-medium bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20 rounded-md">
                {t('asr.currentDefault')}: {ASR_ENGINES.find(e => e.id === (config.pipeline.whisper_mode || 'whisper-cpp').toLowerCase())?.name || config.pipeline.whisper_mode}
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('asr.subtitle')}
            </p>
          </div>
        </div>

        {/* 提示条 */}
        {savedSuccess && (
          <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-xl flex items-center space-x-2 text-xs text-emerald-600 dark:text-emerald-400 animate-in fade-in">
            <CheckCircle2 size={16} />
            <span>{t('asr.saveSuccess')}</span>
          </div>
        )}

        {errorMsg && (
          <div className="p-3 bg-rose-50 border border-rose-200 dark:bg-rose-500/10 dark:border-rose-500/30 rounded-xl flex items-center gap-2 text-xs text-rose-600 dark:text-rose-300 animate-in fade-in">
            <AlertCircle size={16} />
            <span>{errorMsg}</span>
          </div>
        )}

        {/* 主从二级布局卡片 (Master-Detail) */}
        <div className="grid grid-cols-12 gap-5 min-h-[560px]">
          {/* 左侧菜单 */}
          <div className="col-span-4 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-3 flex flex-col justify-between">
            <div className="space-y-1.5">
              <div className="px-3 py-2 text-[11px] font-semibold text-slate-400 dark:text-slate-500 tracking-wider uppercase">
                {t('asr.engineList')}
              </div>
              {ASR_ENGINES.map((item) => {
                const isSelected = activeEngine === item.id;
                const isDefault = (config.pipeline.whisper_mode || 'whisper-cpp').toLowerCase() === item.id;

                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setActiveEngine(item.id)}
                    className={`w-full text-left px-3.5 py-3 rounded-xl border transition-all relative flex items-center justify-between active:scale-[0.98] ${
                      isSelected
                        ? 'bg-blue-50/70 dark:bg-blue-500/15 border-blue-500/40 text-blue-900 dark:text-blue-100 font-medium'
                        : 'bg-transparent hover:bg-slate-50 dark:hover:bg-white/[0.04] border-transparent text-slate-700 dark:text-slate-300'
                    }`}
                  >
                    <div className="flex items-center gap-2.5 text-xs">
                      {item.icon}
                      <span>{item.name}</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      {item.badge && (
                        <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 rounded">
                          {t('asr.recommended')}
                        </span>
                      )}
                      {isDefault && (
                        <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-blue-600 text-white rounded flex items-center gap-0.5">
                          <Check size={10} /> {t('asr.defaultBadge')}
                        </span>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>

          </div>

          {/* 右侧详细配置展示 */}
          <div className="col-span-8 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-6 flex flex-col justify-between space-y-6">
            <div className="space-y-6">
              {/* 头部状态与默认切换 */}
              {(() => {
                const currentMeta = ASR_ENGINES.find((e) => e.id === activeEngine);
                return (
                  <div className="flex items-start justify-between pb-4 border-b border-slate-100 dark:border-white/5">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <h3 className="text-base font-bold text-slate-800 dark:text-white">
                          {currentMeta?.name}
                        </h3>
                        <span className="px-2 py-0.5 text-[11px] font-medium bg-slate-100 dark:bg-white/10 text-slate-600 dark:text-slate-300 rounded-md">
                          {t(currentMeta?.tagKey || '')}
                        </span>
                      </div>
                      {currentMeta?.descKey && (
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                          {t(currentMeta.descKey)}
                        </p>
                      )}
                    </div>
                    <div>
                      {isCurrentDefault ? (
                        <span className="px-3 py-1.5 bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20 rounded-xl text-xs font-semibold flex items-center gap-1.5">
                          <Check size={14} /> {t('asr.defaultBadge')}
                        </span>
                      ) : (
                        <button
                          type="button"
                          onClick={() => handleSetDefault(activeEngine)}
                          className="px-3.5 py-1.5 bg-blue-600 hover:bg-blue-500 active:scale-95 text-white rounded-xl text-xs font-semibold transition-all flex items-center gap-1.5"
                        >
                          {t('asr.setAsDefault')}
                        </button>
                      )}
                    </div>
                  </div>
                );
              })()}

              {/* Whisper.cpp 详细配置 */}
              {activeEngine === 'whisper-cpp' && (
                <div className="space-y-6 animate-in fade-in">
                  {/* 区块一：运行时状态与一键智能安装 */}
                  <div className="p-4 rounded-xl border border-slate-200/80 dark:border-white/10 bg-slate-50/70 dark:bg-white/[0.02] space-y-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Terminal size={16} className="text-blue-500" />
                        <span className="text-xs font-bold text-slate-800 dark:text-white">
                          {t('asr.runtimeTitle')}
                        </span>
                        {runtime?.installed ? (
                          <span className="px-2 py-0.5 text-[10px] font-semibold bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 rounded-full flex items-center gap-1">
                            <CheckCircle2 size={11} /> {t('asr.runtimeReady')}
                          </span>
                        ) : (
                          <span className="px-2 py-0.5 text-[10px] font-semibold bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 rounded-full flex items-center gap-1">
                            <AlertCircle size={11} /> {t('asr.runtimeNotInstalled')}
                          </span>
                        )}
                      </div>

                      <div className="flex items-center gap-2">
                        {!runtime?.installed && (
                          <button
                            type="button"
                            onClick={handleInstallRuntime}
                            disabled={runtimeInstalling}
                            className="px-3 py-1 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-all active:scale-95"
                          >
                            {runtimeInstalling ? (
                              <>
                                <Loader2 size={13} className="animate-spin" />
                                <span>{t('asr.installingRuntime')}</span>
                              </>
                            ) : (
                              <>
                                <Download size={13} />
                                <span>{locale === 'en-US' ? 'Install automatically' : '自动安装'}</span>
                              </>
                            )}
                          </button>
                        )}
                        <RefreshButton
                          onRefresh={handleRefreshRuntime}
                          size={13}
                          className="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200/50 dark:hover:bg-white/10 transition-colors"
                          title={t('asr.refreshList')}
                        />
                      </div>
                    </div>

                    <p className="text-[11px] text-slate-500 dark:text-slate-400 leading-relaxed">
                      {runtime?.installed
                        ? `${t('asr.acceleration')}: ${runtime.acceleration || 'Metal / CPU'} (${runtime.version || 'whisper-cli'})`
                        : t('asr.runtimeAutoDesc')}
                    </p>

                    {runtime?.installed && (
                      <div className="flex items-center gap-2 text-[11px] font-mono text-slate-400 bg-white/80 dark:bg-white/5 px-2.5 py-1.5 rounded-lg border border-slate-200/60 dark:border-white/5 truncate">
                        <span className="text-slate-500 font-sans">{locale === 'en-US' ? 'Path:' : '路径:'}</span>
                        <span className="truncate">{runtime.path || runtime.binary || 'whisper-cli (PATH)'}</span>
                      </div>
                    )}
                  </div>

                  {/* 区块二：模型库与下载管理 */}
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <div>
                        <h4 className="text-xs font-bold text-slate-800 dark:text-white flex items-center gap-1.5">
                          <span>{t('asr.modelListTitle')}</span>
                        </h4>
                      </div>
                      <span className="text-[10px] font-mono text-slate-400">
                        {models.filter(m => m.downloaded).length}/{models.length} {locale === 'en-US' ? 'Ready' : '已就绪'}
                      </span>
                    </div>

                    {/* 模型表格/卡片列表 */}
                    <div className="border border-slate-200/80 dark:border-white/10 rounded-xl overflow-hidden divide-y divide-slate-100 dark:divide-white/5">
                      {models.map((item) => {
                        const isDownloading = downloadingModel === item.name;
                        const isModelActive =
                          item.kind !== 'vad' &&
                          item.downloaded &&
                          (activeModelValue.includes(item.name.toLowerCase()) ||
                            (item.path && activeModelValue.includes(item.path.toLowerCase())));

                        return (
                          <div
                            key={item.name}
                            className={`p-3 transition-colors flex items-center justify-between gap-3 ${
                              isModelActive
                                ? 'bg-blue-50/40 dark:bg-blue-500/10'
                                : 'hover:bg-slate-50/80 dark:hover:bg-white/[0.02]'
                            }`}
                          >
                            <div className="space-y-0.5 min-w-0 flex-1">
                              <div className="flex items-center gap-2">
                                <span className="text-xs font-bold font-mono text-slate-800 dark:text-white">
                                  {item.name}
                                </span>
                                <span className="text-[10px] font-mono px-1.5 py-0.2 bg-slate-100 dark:bg-white/10 text-slate-500 dark:text-slate-300 rounded">
                                  {item.size}
                                </span>
                                {item.name === 'large-v3-turbo' && (
                                  <span className="text-[10px] font-semibold px-1.5 py-0.2 bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 rounded flex items-center gap-0.5">
                                    <Sparkles size={10} /> {t('asr.turboRecommended')}
                                  </span>
                                )}
                                {isModelActive && (
                                  <span className="text-[10px] font-semibold px-1.5 py-0.2 bg-blue-600 text-white rounded flex items-center gap-0.5">
                                    <Check size={10} /> {t('asr.currentActiveModel')}
                                  </span>
                                )}
                              </div>
                              <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                                {getModelDesc(item.name, item.desc)}
                              </p>

                              {/* 显示实际下载量；未知总大小时不捏造百分比。 */}
                              {isDownloading && downloadProgress && (
                                <div className="pt-2 space-y-1">
                                  <div className="flex items-center justify-between text-[10px] text-slate-400 font-mono">
                                    <span className="text-blue-600 dark:text-blue-400 font-semibold">
                                      {downloadProgress.speed ? `${locale === 'en-US' ? 'Download speed' : '下载速度'}: ${downloadProgress.speed}` : (locale === 'en-US' ? 'Downloading…' : '正在下载…')}
                                    </span>
                                    <span>{downloadProgress.downloaded}{downloadProgress.total && ` / ${downloadProgress.total}`}{downloadProgress.percent !== null && ` · ${downloadProgress.percent}%`}</span>
                                  </div>
                                  <div className="w-full bg-slate-100 dark:bg-white/10 rounded-full h-1.5 overflow-hidden">
                                    <div
                                      className={`bg-blue-600 h-1.5 rounded-full transition-all duration-300 ${downloadProgress.percent === null ? 'animate-pulse opacity-40' : ''}`}
                                      style={{ width: downloadProgress.percent === null ? '100%' : `${downloadProgress.percent}%` }}
                                    />
                                  </div>
                                </div>
                              )}
                            </div>

                            {/* 操作区 */}
                            <div className="flex items-center gap-2 shrink-0">
                              {item.downloaded ? (
                                <>
                                  {item.kind !== 'vad' && !isModelActive && (
                                    <button
                                      type="button"
                                      onClick={() => handleActivateModel(item)}
                                      className="px-2.5 py-1 text-xs font-semibold bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-slate-200 rounded-lg transition-colors active:scale-95"
                                    >
                                      {t('asr.activateModel')}
                                    </button>
                                  )}
                                  {item.kind !== 'vad' && (deletingModelName === item.name ? (
                                    <div className="flex items-center space-x-1.5 bg-rose-50 dark:bg-rose-950/40 border border-rose-300 dark:border-rose-500/40 rounded-lg px-1.5 py-0.5">
                                      <button
                                        type="button"
                                        onClick={() => handleDeleteModel(item.name)}
                                        className="px-2 py-0.5 rounded bg-rose-600 hover:bg-rose-700 text-white text-[10px] font-semibold transition-colors"
                                      >
                                        {t('common.confirm')}
                                      </button>
                                      <button
                                        type="button"
                                        onClick={() => setDeletingModelName(null)}
                                        className="px-1 py-0.5 rounded text-slate-500 hover:text-slate-700 dark:text-slate-400 text-[10px] transition-colors"
                                      >
                                        {t('common.cancel')}
                                      </button>
                                    </div>
                                  ) : (
                                    <button
                                      type="button"
                                      onClick={() => setDeletingModelName(item.name)}
                                      className="p-1.5 text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 rounded-lg transition-colors"
                                      title={t('asr.deleteModel')}
                                    >
                                      <Trash2 size={14} />
                                    </button>
                                  ))}
                                  {item.kind === 'vad' && (
                                    <span className="text-[10px] font-medium text-slate-500 dark:text-slate-400">
                                      {locale === 'en-US' ? 'VAD resource' : '检测资源'}
                                    </span>
                                  )}
                                </>
                              ) : isDownloading ? (
                                <span className="text-xs text-blue-600 dark:text-blue-400 font-semibold flex items-center gap-1">
                                  <Loader2 size={13} className="animate-spin" />
                                  <span>{t('asr.downloading')}</span>
                                </span>
                              ) : (
                                <button
                                  type="button"
                                  onClick={() => handleDownloadModel(item.name)}
                                  className="px-2.5 py-1 text-xs font-semibold bg-blue-50 hover:bg-blue-100 dark:bg-blue-500/10 dark:hover:bg-blue-500/20 text-blue-600 dark:text-blue-400 border border-blue-200/80 dark:border-blue-500/20 rounded-lg flex items-center gap-1 transition-all active:scale-95"
                                >
                                  <Download size={12} />
                                  <span>{t('asr.downloadModel')}</span>
                                </button>
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>

                  {/* 区块三：折叠式高级路径覆盖 */}
                  <div className="pt-2 border-t border-slate-100 dark:border-white/5">
                    <button
                      type="button"
                      onClick={() => setShowAdvanced(!showAdvanced)}
                      className="flex items-center gap-1.5 text-xs font-medium text-slate-500 dark:text-slate-400 hover:text-slate-800 dark:hover:text-white transition-colors"
                    >
                      {showAdvanced ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                      <span>{t('asr.customAdvanced')}</span>
                    </button>

                    {showAdvanced && (
                      <div className="mt-3 p-3.5 bg-slate-50/70 dark:bg-white/[0.02] border border-slate-200/80 dark:border-white/10 rounded-xl space-y-4 animate-in fade-in">
                        <div>
                          <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                            {t('asr.modelFilePath')}
                          </label>
                          <div className="flex gap-2">
                            <input
                              type="text"
                              placeholder={t('asr.modelFilePlaceholder', 'ggml-base.bin 或自定义模型文件绝对路径')}
                              value={config.pipeline.whisper_model}
                              onChange={(e) =>
                                setConfig({
                                  ...config,
                                  pipeline: {
                                    ...config.pipeline,
                                    whisper_model: e.target.value,
                                  },
                                })
                              }
                              className="flex-1 h-9 px-3 rounded-lg border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 text-xs text-slate-800 dark:text-white font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                            />
                            <button
                              type="button"
                              onClick={handleSelectModelFile}
                              className="px-3 h-9 rounded-lg bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 text-slate-700 dark:text-slate-200 text-xs font-medium border border-slate-200 dark:border-white/10 flex items-center gap-1 transition-colors shrink-0"
                            >
                              <FolderOpen size={13} />
                              <span>{t('common.browse')}</span>
                            </button>
                          </div>
                        </div>

                        <div>
                          <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                            {t('asr.manualCustomPath')}
                          </label>
                          <div className="flex gap-2">
                            <input
                              type="text"
                              placeholder={t('asr.customBinPlaceholder')}
                              value={config.pipeline.whisper_bin}
                              onChange={(e) =>
                                setConfig({
                                  ...config,
                                  pipeline: {
                                    ...config.pipeline,
                                    whisper_bin: e.target.value,
                                  },
                                })
                              }
                              className="flex-1 h-9 px-3 rounded-lg border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 text-xs text-slate-800 dark:text-white font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                            />
                            <button
                              type="button"
                              onClick={handleSelectBinFile}
                              className="px-3 h-9 rounded-lg bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 text-slate-700 dark:text-slate-200 text-xs font-medium border border-slate-200 dark:border-white/10 flex items-center gap-1 transition-colors shrink-0"
                            >
                              <FolderOpen size={13} />
                              <span>{t('common.browse')}</span>
                            </button>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* OpenAI 兼容 ASR 配置 */}
              {activeEngine === 'openai' && (
                <div className="space-y-4 animate-in fade-in">
                  <div className="p-4 bg-slate-50 dark:bg-white/[0.03] border border-slate-200/80 dark:border-white/10 rounded-xl space-y-4">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                        {t('asr.openaiBaseUrl')}
                      </label>
                      <input
                        type="url"
                        value={config.pipeline.asr_base_url || ''}
                        onChange={(e) => setConfig({ ...config, pipeline: { ...config.pipeline, asr_base_url: e.target.value } })}
                        placeholder="https://api.openai.com/v1"
                        className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 text-xs text-slate-800 dark:text-slate-200 outline-none focus:border-blue-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                        {t('asr.openaiApiKey')}
                      </label>
                      <input
                        type="password"
                        autoComplete="new-password"
                        value={config.pipeline.asr_api_key || ''}
                        onChange={(e) => setConfig({ ...config, pipeline: { ...config.pipeline, asr_api_key: e.target.value } })}
                        placeholder={t('asr.openaiApiKeyPlaceholder')}
                        className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 text-xs text-slate-800 dark:text-slate-200 outline-none focus:border-blue-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                        {t('asr.openaiModel')}
                      </label>
                      <input
                        type="text"
                        value={config.pipeline.asr_model || ''}
                        onChange={(e) => setConfig({ ...config, pipeline: { ...config.pipeline, asr_model: e.target.value } })}
                        placeholder="whisper-1"
                        className="w-full px-3 py-2 rounded-lg border border-slate-200 dark:border-white/10 bg-white dark:bg-white/5 text-xs text-slate-800 dark:text-slate-200 outline-none focus:border-blue-500"
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* 底部保存按钮（当用户手动微调路径或选择时保存） */}
            <div className="pt-4 border-t border-slate-100 dark:border-white/5 flex items-center justify-end">
              <button
                type="button"
                onClick={() => handleSave()}
                disabled={loading}
                className="px-5 py-2 bg-blue-600 hover:bg-blue-500 active:scale-95 disabled:opacity-50 text-white rounded-xl text-xs font-semibold flex items-center gap-2 transition-all cursor-pointer"
              >
                {loading ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
                <span>{loading ? t('common.saving') : t('common.saveSettings')}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
