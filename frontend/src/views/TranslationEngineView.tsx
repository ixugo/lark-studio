import React, { useEffect, useState } from 'react';
import {
  Globe,
  Sparkles,
  Server,
  Cloud,
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

interface EngineItem {
  id: 'google' | 'bing' | 'openai' | 'deeplx';
  name: string;
  tag: string;
  desc: string;
  icon: React.ReactNode;
  badge?: string;
}

const ENGINES: EngineItem[] = [
  {
    id: 'google',
    name: '谷歌翻译 (Google)',
    tag: '免配置 · 免Key',
    desc: '基于官方公共接口，稳定可靠，推荐首选',
    icon: <Globe size={18} className="text-blue-500" />,
    badge: '推荐',
  },
  {
    id: 'bing',
    name: '必应翻译 (Bing)',
    tag: '免Key · 自动容灾',
    desc: 'Edge 浏览器匿名端点，异常自动平滑切换',
    icon: <Cloud size={18} className="text-cyan-500" />,
  },
  {
    id: 'openai',
    name: 'OpenAI 兼容接口',
    tag: '大模型 · 自定义',
    desc: '支持 DeepSeek, GPT-4o, Ollama 本地模型',
    icon: <Sparkles size={18} className="text-indigo-500" />,
  },
  {
    id: 'deeplx',
    name: 'DeepLX 自建端点',
    tag: '自建服务',
    desc: '私有化部署的高性能 DeepL 反代翻译端点',
    icon: <Server size={18} className="text-emerald-500" />,
  },
];

export const TranslationEngineView: React.FC = () => {
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [activeEngine, setActiveEngine] = useState<'google' | 'bing' | 'openai' | 'deeplx'>('google');
  const [showApiKey, setShowApiKey] = useState(false);
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.getConfig().then((cfg) => {
      setConfig(cfg);
      const current = (cfg.llm?.provider?.toLowerCase() || 'google') as any;
      if (['google', 'bing', 'openai', 'deeplx'].includes(current)) {
        setActiveEngine(current);
      }
    }).catch(console.error);
  }, []);

  const handleSave = async (engineToSet?: string) => {
    if (!config) return;
    setLoading(true);
    setErrorMsg(null);
    try {
      const targetProvider = engineToSet || config.llm.provider || activeEngine;
      const updatedConfig = {
        ...config,
        llm: {
          ...config.llm,
          provider: targetProvider,
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

  const handleSetDefault = (engineId: 'google' | 'bing' | 'openai' | 'deeplx') => {
    if (!config) return;
    setConfig({
      ...config,
      llm: {
        ...config.llm,
        provider: engineId,
      },
    });
    handleSave(engineId);
  };

  if (!config) {
    return (
      <div className="h-screen flex items-center justify-center text-xs text-slate-400">
        正在读取翻译引擎配置...
      </div>
    );
  }

  const isCurrentDefault = (config.llm.provider || 'google').toLowerCase() === activeEngine;

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6 select-none bg-[#F5F5F7] dark:bg-[#121215]">
      <div className="max-w-5xl w-full mx-auto space-y-6 pt-2 pb-16">
        {/* 顶部标题区 */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
                翻译引擎设置
              </h2>
              <span className="px-2 py-0.5 text-[11px] font-medium bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20 rounded-md">
                当前默认: {ENGINES.find(e => e.id === (config.llm.provider || 'google').toLowerCase())?.name || config.llm.provider}
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              左侧选择翻译服务，右侧进行详细配置与参数调校
            </p>
          </div>
        </div>

        {/* 提示条 */}
        {savedSuccess && (
          <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-xl flex items-center space-x-2 text-xs text-emerald-600 dark:text-emerald-400 animate-in fade-in">
            <CheckCircle2 size={16} />
            <span>翻译引擎配置已更新并即时热重载生效！</span>
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
                支持的翻译供应商
              </div>
              {ENGINES.map((item) => {
                const isSelected = activeEngine === item.id;
                const isDefault = (config.llm.provider || 'google').toLowerCase() === item.id;

                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setActiveEngine(item.id)}
                    className={`w-full text-left p-3 rounded-xl border transition-all relative flex flex-col gap-1 active:scale-[0.98] ${
                      isSelected
                        ? 'bg-blue-50/70 dark:bg-blue-500/15 border-blue-500/40 text-blue-900 dark:text-blue-100'
                        : 'bg-transparent hover:bg-slate-50 dark:hover:bg-white/[0.04] border-transparent text-slate-700 dark:text-slate-300'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2 font-medium text-xs">
                        {item.icon}
                        <span>{item.name}</span>
                      </div>
                      <div className="flex items-center gap-1">
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
                    </div>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 line-clamp-1 pl-6">
                      {item.desc}
                    </p>
                  </button>
                );
              })}
            </div>

            <div className="p-3 bg-slate-50 dark:bg-white/[0.03] border border-slate-200/70 dark:border-white/5 rounded-xl text-[11px] text-slate-500 dark:text-slate-400 space-y-1">
              <div className="flex items-center gap-1 font-semibold text-slate-700 dark:text-slate-300">
                <ShieldCheck size={14} className="text-emerald-500" />
                <span>免 Key 智能保障</span>
              </div>
              <p>系统已集成高可用故障转移机制，当公共端点波动时将自动平滑降级，确保任务不中断。</p>
            </div>
          </div>

          {/* 右侧详细配置展示 */}
          <div className="col-span-8 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-6 flex flex-col justify-between space-y-6">
            <div className="space-y-6">
              {/* 头部状态与默认切换 */}
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 dark:border-white/5">
                <div>
                  <h3 className="text-base font-bold text-slate-800 dark:text-white flex items-center gap-2">
                    {ENGINES.find((e) => e.id === activeEngine)?.name}
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                    {ENGINES.find((e) => e.id === activeEngine)?.desc}
                  </p>
                </div>
                <div>
                  {isCurrentDefault ? (
                    <span className="px-3 py-1.5 bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20 rounded-xl text-xs font-semibold flex items-center gap-1.5">
                      <Check size={14} /> 正在作为默认引擎生效
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

              {/* 谷歌翻译配置 */}
              {activeEngine === 'google' && (
                <div className="space-y-4 animate-in fade-in">
                  <div className="p-4 rounded-xl bg-blue-50/60 dark:bg-blue-500/10 border border-blue-100 dark:border-blue-500/20 text-xs text-blue-900 dark:text-blue-200 space-y-2">
                    <div className="font-semibold flex items-center gap-1.5">
                      <Globe size={16} /> 谷歌官方公共翻译接口（免配置即开即用）
                    </div>
                    <p className="text-[12px] leading-relaxed opacity-90">
                      谷歌翻译公共服务支持近百种语言，速度极快，无需配置 API Key、Token 令牌或自建代理，开箱即用。已作为 VDub 全局高可靠兜底服务。
                    </p>
                  </div>

                  <div className="grid grid-cols-2 gap-4 pt-2">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        单次切片句子数 (Chunk Size)
                      </label>
                      <input
                        type="number"
                        min={3}
                        max={30}
                        value={config.pipeline.translate_chunk_size}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            pipeline: {
                              ...config.pipeline,
                              translate_chunk_size: parseInt(e.target.value) || 10,
                            },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                      <p className="text-[11px] text-slate-400 mt-1">推荐 10~15 句/批，兼顾上下文连贯与响应速度</p>
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        默认目标语言
                      </label>
                      <input
                        type="text"
                        value={config.pipeline.default_target_lang || 'zh-CN'}
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
                      <p className="text-[11px] text-slate-400 mt-1">如 zh-CN, en, ja, ko 等</p>
                    </div>
                  </div>
                </div>
              )}

              {/* 必应翻译配置 */}
              {activeEngine === 'bing' && (
                <div className="space-y-4 animate-in fade-in">
                  <div className="p-4 rounded-xl bg-cyan-50/60 dark:bg-cyan-500/10 border border-cyan-100 dark:border-cyan-500/20 text-xs text-cyan-900 dark:text-cyan-200 space-y-2">
                    <div className="font-semibold flex items-center gap-1.5">
                      <Cloud size={16} /> 必应 (Edge 浏览器翻译端点)
                    </div>
                    <p className="text-[12px] leading-relaxed opacity-90">
                      使用微软 Edge 浏览器的匿名认证凭据。注：因微软官方接口策略偶尔调整，当必应端点不可用时，VDub 内部会自动容灾平滑降级到 Google 免费翻译，保障任务顺利完成。
                    </p>
                  </div>

                  <div className="grid grid-cols-2 gap-4 pt-2">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        故障自动容灾转移
                      </label>
                      <div className="h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-600 dark:text-slate-300 flex items-center">
                        <span className="w-2 h-2 rounded-full bg-emerald-500 mr-2" />
                        已默认启用 (自动回退至 Google 免费翻译)
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        单次切片句子数
                      </label>
                      <input
                        type="number"
                        min={3}
                        max={30}
                        value={config.pipeline.translate_chunk_size}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            pipeline: {
                              ...config.pipeline,
                              translate_chunk_size: parseInt(e.target.value) || 10,
                            },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                    </div>
                  </div>
                </div>
              )}

              {/* OpenAI 兼容配置 */}
              {activeEngine === 'openai' && (
                <div className="space-y-4 animate-in fade-in">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        Base URL (API 接口地址)
                      </label>
                      <input
                        type="text"
                        placeholder="http://localhost:11434/v1 或 https://api.openai.com/v1"
                        value={config.llm.base_url}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            llm: { ...config.llm, base_url: e.target.value },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                      <p className="text-[11px] text-slate-400 mt-1">兼容 OpenAI 规范的任何大模型接口</p>
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        API Key (密钥)
                      </label>
                      <div className="relative">
                        <input
                          type={showApiKey ? 'text' : 'password'}
                          placeholder="sk-..."
                          value={config.llm.api_key}
                          onChange={(e) =>
                            setConfig({
                              ...config,
                              llm: { ...config.llm, api_key: e.target.value },
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
                      <p className="text-[11px] text-slate-400 mt-1">本地 Ollama 无需填 Key</p>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        模型名称 (Model)
                      </label>
                      <input
                        type="text"
                        placeholder="qwen2.5:7b, deepseek-chat, gpt-4o-mini"
                        value={config.llm.model}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            llm: { ...config.llm, model: e.target.value },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        每次并发句子数
                      </label>
                      <input
                        type="number"
                        min={3}
                        max={30}
                        value={config.pipeline.translate_chunk_size}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            pipeline: {
                              ...config.pipeline,
                              translate_chunk_size: parseInt(e.target.value) || 10,
                            },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                      自定义翻译 Prompt 提示词
                    </label>
                    <textarea
                      rows={3}
                      placeholder="留空使用内置专业视频字幕翻译提示词。可用变量：{{target_lang}} {{count}}"
                      value={config.pipeline.translate_prompt}
                      onChange={(e) =>
                        setConfig({
                          ...config,
                          pipeline: {
                            ...config.pipeline,
                            translate_prompt: e.target.value,
                          },
                        })
                      }
                      className="w-full p-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30 resize-none font-mono"
                    />
                  </div>
                </div>
              )}

              {/* DeepLX 配置 */}
              {activeEngine === 'deeplx' && (
                <div className="space-y-4 animate-in fade-in">
                  <div>
                    <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                      DeepLX API 端点地址
                    </label>
                    <input
                      type="text"
                      placeholder="http://localhost:1188/translate"
                      value={config.llm.deeplx_url}
                      onChange={(e) =>
                        setConfig({
                          ...config,
                          llm: { ...config.llm, deeplx_url: e.target.value },
                        })
                      }
                      className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                    />
                    <p className="text-[11px] text-slate-400 mt-1">例如本地或内网自建的 deepLX 服务的 /translate 路由</p>
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
                <span>{loading ? '正在保存...' : '保存翻译配置'}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
