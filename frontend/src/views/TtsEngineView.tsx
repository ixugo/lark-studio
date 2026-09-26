import React, { useEffect, useState } from 'react';
import {
  Volume2,
  Sparkles,
  CheckCircle2,
  AlertCircle,
  Save,
  Check,
  Eye,
  EyeOff,
  ShieldCheck,
} from 'lucide-react';
import { api } from '../lib/api';
import { ConfigDTO } from '../types';

interface TtsEngineItem {
  id: 'edge' | 'openai';
  name: string;
  tag: string;
  desc: string;
  icon: React.ReactNode;
  badge?: string;
}

const TTS_ENGINES: TtsEngineItem[] = [
  {
    id: 'edge',
    name: 'Edge TTS',
    tag: '免Key推荐',
    desc: '微软官方超拟真神经音色，发音自然流畅且无需配置密钥，推荐首选',
    icon: <Volume2 size={18} className="text-blue-500" />,
    badge: '推荐',
  },
  {
    id: 'openai',
    name: 'OpenAI TTS',
    tag: '大模型接口',
    desc: '兼容 OpenAI 协议的高保真语音合成大模型，音质细腻生动',
    icon: <Sparkles size={18} className="text-indigo-500" />,
  },
];

const POPULAR_EDGE_VOICES = [
  { value: 'zh-CN-YunjianNeural', label: '云健 · 男声沉稳推荐' },
  { value: 'zh-CN-XiaoxiaoNeural', label: '晓晓 · 女声亲切推荐' },
  { value: 'zh-CN-YunxiNeural', label: '云希 · 男声阳光解说' },
  { value: 'zh-CN-YunxiaNeural', label: '云夏 · 男声少年感' },
  { value: 'zh-CN-XiaoyiNeural', label: '晓伊 · 女声抒情阅读' },
  { value: 'en-US-ChristopherNeural', label: 'Christopher · 美式英语男声' },
  { value: 'en-US-JennyNeural', label: 'Jenny · 美式英语女声' },
];

export const TtsEngineView: React.FC = () => {
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [activeEngine, setActiveEngine] = useState<'edge' | 'openai'>('edge');
  const [showApiKey, setShowApiKey] = useState(false);
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.getConfig().then((cfg) => {
      setConfig(cfg);
      const current = (cfg.tts?.type?.toLowerCase() || 'edge') as any;
      if (['edge', 'openai'].includes(current)) {
        setActiveEngine(current);
      }
    }).catch(console.error);
  }, []);

  const handleSave = async (engineToSet?: string) => {
    if (!config) return;
    setLoading(true);
    setErrorMsg(null);
    try {
      const targetType = engineToSet || config.tts.type || activeEngine;
      const updatedConfig = {
        ...config,
        tts: {
          ...config.tts,
          type: targetType,
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

  const handleSetDefault = (engineId: 'edge' | 'openai') => {
    if (!config) return;
    setConfig({
      ...config,
      tts: {
        ...config.tts,
        type: engineId,
      },
    });
    handleSave(engineId);
  };

  if (!config) {
    return (
      <div className="h-screen flex items-center justify-center text-xs text-slate-400">
        正在读取语音合成配置...
      </div>
    );
  }

  const isCurrentDefault = (config.tts.type || 'edge').toLowerCase() === activeEngine;

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6 select-none bg-[#F5F5F7] dark:bg-[#121215]">
      <div className="max-w-5xl w-full mx-auto space-y-6 pt-2 pb-16">
        {/* 顶部标题区 */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
                语音合成引擎
              </h2>
              <span className="px-2 py-0.5 text-[11px] font-medium bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20 rounded-md">
                当前默认: {TTS_ENGINES.find(e => e.id === (config.tts.type || 'edge').toLowerCase())?.name || config.tts.type}
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              配置视频配音与语音合成音色、并发线程及自动语速调整
            </p>
          </div>
        </div>

        {/* 提示条 */}
        {savedSuccess && (
          <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-xl flex items-center space-x-2 text-xs text-emerald-600 dark:text-emerald-400 animate-in fade-in">
            <CheckCircle2 size={16} />
            <span>语音合成配置已保存并生效！</span>
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
                合成引擎列表
              </div>
              {TTS_ENGINES.map((item) => {
                const isSelected = activeEngine === item.id;
                const isDefault = (config.tts.type || 'edge').toLowerCase() === item.id;

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
                <span>智能音画对齐</span>
              </div>
              <p>系统内置动态调速算法，确保生成的译文配音与原视频片段精准对齐，无吞字、无拖沓。</p>
            </div>
          </div>

          {/* 右侧详细配置展示 */}
          <div className="col-span-8 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-6 flex flex-col justify-between space-y-6">
            <div className="space-y-6">
              {/* 头部状态与默认切换 */}
              {(() => {
                const currentMeta = TTS_ENGINES.find((e) => e.id === activeEngine);
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

              {/* Edge TTS 配置 */}
              {activeEngine === 'edge' && (
                <div className="space-y-4 animate-in fade-in">
                  <div>
                    <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                      默认神经音色
                    </label>
                    <select
                      value={config.tts.voice}
                      onChange={(e) =>
                        setConfig({
                          ...config,
                          tts: { ...config.tts, voice: e.target.value },
                        })
                      }
                      className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                    >
                      {POPULAR_EDGE_VOICES.map((v) => (
                        <option key={v.value} value={v.value}>
                          {v.label}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        TTS 调速上限
                      </label>
                      <input
                        type="number"
                        step="0.05"
                        min="1.0"
                        max="2.0"
                        value={config.pipeline.max_speed_factor}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            pipeline: {
                              ...config.pipeline,
                              max_speed_factor: parseFloat(e.target.value) || 1.2,
                            },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                      <p className="text-[11px] text-slate-400 mt-1">推荐 1.2 ~ 1.3，译文超时时自动轻微倍速压制</p>
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        TTS 并发生成协程数
                      </label>
                      <input
                        type="number"
                        min={1}
                        max={6}
                        value={config.pipeline.tts_workers}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            pipeline: {
                              ...config.pipeline,
                              tts_workers: parseInt(e.target.value) || 2,
                            },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                      <p className="text-[11px] text-slate-400 mt-1">默认 2 协程并发，兼顾速度与避免被服务端流控</p>
                    </div>
                  </div>
                </div>
              )}

              {/* OpenAI TTS 配置 */}
              {activeEngine === 'openai' && (
                <div className="space-y-4 animate-in fade-in">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        OpenAI TTS Base URL
                      </label>
                      <input
                        type="text"
                        placeholder="https://api.openai.com/v1"
                        value={config.tts.base_url}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            tts: { ...config.tts, base_url: e.target.value },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        访问密钥 API Key
                      </label>
                      <div className="relative">
                        <input
                          type={showApiKey ? 'text' : 'password'}
                          placeholder="sk-..."
                          value={config.tts.api_key}
                          onChange={(e) =>
                            setConfig({
                              ...config,
                              tts: { ...config.tts, api_key: e.target.value },
                            })
                          }
                          className="w-full h-10 pl-3 pr-9 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                        />
                        <button
                          type="button"
                          onClick={() => setShowApiKey(!showApiKey)}
                          className="absolute right-2.5 top-2.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
                        >
                          {showApiKey ? <EyeOff size={15} /> : <Eye size={15} />}
                        </button>
                      </div>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        TTS 模型
                      </label>
                      <select
                        value={config.tts.model || 'tts-1'}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            tts: { ...config.tts, model: e.target.value },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      >
                        <option value="tts-1">tts-1 · 标准低延迟</option>
                        <option value="tts-1-hd">tts-1-hd · 高保真高清</option>
                      </select>
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        默认音色
                      </label>
                      <select
                        value={config.tts.voice || 'alloy'}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            tts: { ...config.tts, voice: e.target.value },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      >
                        <option value="alloy">alloy · 中性沉稳</option>
                        <option value="echo">echo · 圆润男声</option>
                        <option value="fable">fable · 英伦质感</option>
                        <option value="onyx">onyx · 低沉浑厚男声</option>
                        <option value="nova">nova · 明亮女声</option>
                        <option value="shimmer">shimmer · 清脆女声</option>
                      </select>
                    </div>
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
                <span>{loading ? '正在保存...' : '保存语音配置'}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
