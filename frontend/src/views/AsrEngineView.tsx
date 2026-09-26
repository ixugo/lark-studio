import React, { useEffect, useState } from 'react';
import {
  Cpu,
  Radio,
  CheckCircle2,
  AlertCircle,
  Save,
  Check,
  FolderOpen,
  FileCode,
  ShieldCheck,
  Info,
} from 'lucide-react';
import { api } from '../lib/api';
import { ConfigDTO } from '../types';

interface AsrEngineItem {
  id: 'whisper-cpp' | 'sense-voice';
  name: string;
  tag: string;
  desc: string;
  icon: React.ReactNode;
  badge?: string;
}

const ASR_ENGINES: AsrEngineItem[] = [
  {
    id: 'whisper-cpp',
    name: 'Whisper.cpp',
    tag: '离线硬件加速',
    desc: '基于 GGML 框架的高性能纯 C/C++ 离线识别，支持 Metal GPU 硬件加速',
    icon: <Cpu size={18} className="text-blue-500" />,
    badge: '推荐',
  },
  {
    id: 'sense-voice',
    name: 'SenseVoice',
    tag: '多语言极速',
    desc: '轻量端到端多语言富文本语音识别模型，极速推理转录',
    icon: <Radio size={18} className="text-indigo-500" />,
  },
];

export const AsrEngineView: React.FC = () => {
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [activeEngine, setActiveEngine] = useState<'whisper-cpp' | 'sense-voice'>('whisper-cpp');
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.getConfig().then((cfg) => {
      setConfig(cfg);
      const current = (cfg.pipeline?.whisper_mode?.toLowerCase() || 'whisper-cpp') as any;
      if (['whisper-cpp', 'sense-voice'].includes(current)) {
        setActiveEngine(current);
      }
    }).catch(console.error);
  }, []);

  const handleSelectModelFile = async () => {
    try {
      const wails = (window as any)?.wails;
      if (wails?.Dialogs?.OpenFile) {
        const filePath = await wails.Dialogs.OpenFile({
          Title: '选择 Whisper GGML 模型文件 (*.bin)',
          Filters: [{ DisplayName: 'Whisper Model (*.bin)', Pattern: '*.bin;*.ggml' }],
        });
        if (filePath && config) {
          setConfig({
            ...config,
            pipeline: { ...config.pipeline, whisper_model: filePath },
          });
        }
      }
    } catch (e) {
      console.warn('打开文件选择器失败', e);
    }
  };

  const handleSelectBinFile = async () => {
    try {
      const wails = (window as any)?.wails;
      if (wails?.Dialogs?.OpenFile) {
        const filePath = await wails.Dialogs.OpenFile({
          Title: '选择 whisper-cli 可执行文件',
        });
        if (filePath && config) {
          setConfig({
            ...config,
            pipeline: { ...config.pipeline, whisper_bin: filePath },
          });
        }
      }
    } catch (e) {
      console.warn('打开文件选择器失败', e);
    }
  };

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

  const handleSetDefault = (engineId: 'whisper-cpp' | 'sense-voice') => {
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
        正在读取语音识别配置...
      </div>
    );
  }

  const isCurrentDefault = (config.pipeline.whisper_mode || 'whisper-cpp').toLowerCase() === activeEngine;

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6 select-none bg-[#F5F5F7] dark:bg-[#121215]">
      <div className="max-w-5xl w-full mx-auto space-y-6 pt-2 pb-16">
        {/* 顶部标题区 */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
                语音识别引擎
              </h2>
              <span className="px-2 py-0.5 text-[11px] font-medium bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20 rounded-md">
                当前默认: {ASR_ENGINES.find(e => e.id === (config.pipeline.whisper_mode || 'whisper-cpp').toLowerCase())?.name || config.pipeline.whisper_mode}
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              配置离线语音识别后端、Whisper GGML 模型权重路径及硬件推理
            </p>
          </div>
        </div>

        {/* 提示条 */}
        {savedSuccess && (
          <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-xl flex items-center space-x-2 text-xs text-emerald-600 dark:text-emerald-400 animate-in fade-in">
            <CheckCircle2 size={16} />
            <span>语音识别配置已保存并生效！</span>
          </div>
        )}

        {errorMsg && (
          <div className="p-3 bg-rose-50 border border-rose-200 dark:bg-rose-500/10 dark:border-rose-500/30 rounded-xl flex items-center gap-2 text-xs text-rose-600 dark:text-rose-300 animate-in fade-in">
            <AlertCircle size={16} />
            <span>保存失败: {errorMsg}</span>
          </div>
        )}

        {/* 主从二级布局卡片 (Master-Detail) */}
        <div className="grid grid-cols-12 gap-5 min-h-[520px]">
          {/* 左侧菜单 */}
          <div className="col-span-4 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-3 flex flex-col justify-between">
            <div className="space-y-1.5">
              <div className="px-3 py-2 text-[11px] font-semibold text-slate-400 dark:text-slate-500 tracking-wider uppercase">
                识别引擎列表
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
                          {item.badge}
                        </span>
                      )}
                      {isDefault && (
                        <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-blue-600 text-white rounded flex items-center gap-0.5">
                          <Check size={10} /> 默认
                        </span>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>

            <div className="p-3 bg-slate-50 dark:bg-white/[0.03] border border-slate-200/70 dark:border-white/5 rounded-xl text-[11px] text-slate-500 dark:text-slate-400 space-y-1">
              <div className="flex items-center gap-1 font-semibold text-slate-700 dark:text-slate-300">
                <ShieldCheck size={14} className="text-emerald-500" />
                <span>离线与数据隐私</span>
              </div>
              <p>Whisper.cpp 完全在您的本地设备上执行推理，音频内容绝不上云，保障隐私安全。</p>
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
                        {currentMeta?.tag && (
                          <span className="px-2 py-0.5 text-[11px] font-medium bg-slate-100 dark:bg-white/10 text-slate-600 dark:text-slate-300 rounded-md">
                            {currentMeta.tag}
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        {currentMeta?.desc}
                      </p>
                    </div>
                    <div>
                      {isCurrentDefault ? (
                        <span className="px-3 py-1.5 bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20 rounded-xl text-xs font-semibold flex items-center gap-1.5">
                          <Check size={14} /> 默认引擎
                        </span>
                      ) : (
                        <button
                          type="button"
                          onClick={() => handleSetDefault(activeEngine)}
                          className="px-3.5 py-1.5 bg-blue-600 hover:bg-blue-500 active:scale-95 text-white rounded-xl text-xs font-semibold transition-all flex items-center gap-1.5"
                        >
                          设为默认引擎
                        </button>
                      )}
                    </div>
                  </div>
                );
              })()}

              {/* Whisper.cpp 配置 */}
              {activeEngine === 'whisper-cpp' && (
                <div className="space-y-5 animate-in fade-in">
                  <div>
                    <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                      GGML 模型文件路径
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        placeholder="如 ~/dsub/models/ggml-base.bin 或 /usr/local/share/whisper/ggml-small.bin"
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
                        className="flex-1 h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                      <button
                        type="button"
                        onClick={handleSelectModelFile}
                        className="px-3.5 h-10 rounded-xl bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-slate-200 text-xs font-semibold border border-slate-200 dark:border-white/10 flex items-center gap-1.5 transition-colors shrink-0"
                      >
                        <FolderOpen size={14} />
                        <span>浏览...</span>
                      </button>
                    </div>
                    <p className="text-[11px] text-slate-400 mt-1">
                      可直接填入模型文件绝对路径，或点击右侧按钮在系统访达中选择。推荐使用 ggml-base.bin 或 ggml-small.bin。
                    </p>
                  </div>

                  <div>
                    <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                      whisper-cli 程序路径
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        placeholder="留空即使用应用内嵌运行时，或输入外部 whisper-cli 路径"
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
                        className="flex-1 h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white font-mono focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                      <button
                        type="button"
                        onClick={handleSelectBinFile}
                        className="px-3.5 h-10 rounded-xl bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-slate-200 text-xs font-semibold border border-slate-200 dark:border-white/10 flex items-center gap-1.5 transition-colors shrink-0"
                      >
                        <FileCode size={14} />
                        <span>浏览...</span>
                      </button>
                    </div>
                  </div>

                  <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-200/80 dark:border-white/5 flex items-start gap-2.5 text-xs text-slate-600 dark:text-slate-400">
                    <Info size={16} className="text-blue-500 shrink-0 mt-0.5" />
                    <div>
                      <span className="font-semibold text-slate-700 dark:text-slate-200">提示：</span>
                      macOS Apple Silicon 系列芯片设备已自动启用 Metal GPU 硬件加速，推理速度相较纯 CPU 提升 3~5 倍。
                    </div>
                  </div>
                </div>
              )}

              {/* SenseVoice 配置 */}
              {activeEngine === 'sense-voice' && (
                <div className="space-y-4 animate-in fade-in">
                  <div className="p-4 rounded-xl bg-indigo-50/60 dark:bg-indigo-500/10 border border-indigo-100 dark:border-indigo-500/20 text-xs text-indigo-900 dark:text-indigo-200 space-y-2">
                    <div className="font-semibold flex items-center gap-1.5">
                      <Radio size={16} /> SenseVoice 多语言端到端识别
                    </div>
                    <p className="text-[12px] leading-relaxed opacity-90">
                      SenseVoice 是轻量级多语言语音识别模型，具备极快的声音转文字速度，适用于中文及常见外语的高效转录。
                    </p>
                  </div>
                </div>
              )}
            </div>

            <div className="pt-4 border-t border-slate-100 dark:border-white/5 flex items-center justify-between text-xs text-slate-500">
              <span>所做配置自动保存至本地配置中心</span>
              <button
                type="button"
                onClick={() => handleSave()}
                disabled={loading}
                className="px-5 py-2 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white transition-all flex items-center gap-1.5 active:scale-[0.98] disabled:opacity-50"
              >
                <Save size={14} />
                <span>{loading ? '正在保存...' : '保存识别配置'}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
