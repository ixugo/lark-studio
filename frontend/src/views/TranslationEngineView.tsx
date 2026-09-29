import { configFormUpdates } from '../lib/configForms';
import React, { useEffect, useState } from 'react';
import {
  Globe,
  Sparkles,
  Cloud,
  CheckCircle2,
  AlertCircle,
  Save,
  Check,
  Eye,
  EyeOff,
  Zap,
  Loader2,
} from 'lucide-react';
import { api } from '../lib/api';
import { ConfigDTO } from '../types';
import { useTranslation } from '../i18n';
import { RemoteModelSelect, useRemoteModels } from '../components/RemoteModelSelect';
import { useSaveFeedback } from '../lib/useSaveFeedback';

interface EngineItem {
  id: 'bing' | 'google' | 'openai';
  name: string;
  tag: string;
  desc: string;
  icon: React.ReactNode;
  badge?: string;
}

const ENGINES: EngineItem[] = [
  {
    id: 'bing',
    name: '必应翻译',
    tag: '免Key推荐',
    desc: '微软 Edge 翻译通道，端点异常时由系统自动平滑降级至谷歌翻译',
    icon: <Cloud size={18} className="text-cyan-500" />,
    badge: '推荐',
  },
  {
    id: 'google',
    name: '谷歌翻译',
    tag: '免Key推荐',
    desc: '基于官方公共接口，近百种语言支持，速度极快且稳定可靠，推荐首选',
    icon: <Globe size={18} className="text-blue-500" />,
  },
  {
    id: 'openai',
    name: 'OpenAI 兼容',
    tag: '大模型自定义',
    desc: '支持 DeepSeek、GPT-4o、Claude、Ollama 等任何兼容协议的大模型端点',
    icon: <Sparkles size={18} className="text-indigo-500" />,
  },
];

export const TranslationEngineView: React.FC = () => {
  const { locale, t } = useTranslation();
  const english = locale === 'en-US';
  // 保留页面现有双语文案，并随全局语言切换，避免供应商配置项只显示中文。
  const tr = (zh: string, en: string) => english ? en : zh;
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [activeEngine, setActiveEngine] = useState<'bing' | 'google' | 'openai'>('bing');
  const [showApiKey, setShowApiKey] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { saved, clearSaved, showSaved } = useSaveFeedback();
  const [testingConn, setTestingConn] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const remoteModels = useRemoteModels(config?.llm.base_url || '', config?.llm.api_key || '',
    !!config && (activeEngine === 'openai' || config.llm.provider === 'openai'));
  const engineDetails = {
    bing: { name: tr('必应翻译', 'Bing Translator'), tag: tr('免 Key 推荐', 'Recommended · No Key'), desc: tr('微软 Edge 翻译通道，异常时自动降级至谷歌翻译', 'Microsoft Edge translation with automatic fallback to Google when unavailable.'), badge: tr('推荐', 'Recommended') },
    google: { name: tr('谷歌翻译', 'Google Translate'), tag: tr('免 Key 服务', 'No Key Required'), desc: tr('公共翻译服务，支持近百种语言，速度快且稳定。', 'Public translation service supporting nearly 100 languages with fast, reliable responses.'), badge: undefined },
    openai: { name: tr('OpenAI 兼容', 'OpenAI Compatible'), tag: tr('自定义大模型', 'Custom LLM'), desc: '', badge: undefined },
  };

  useEffect(() => {
    api.getConfig().then((cfg) => {
      const patched: ConfigDTO = {
        ...cfg,
        pipeline: {
          ...cfg.pipeline,
          translate_chunk_size: cfg.pipeline?.translate_chunk_size ? cfg.pipeline.translate_chunk_size : 10,
          default_target_lang: cfg.pipeline?.default_target_lang || 'zh-CN',
        },
        llm: {
          ...cfg.llm,
          base_url: cfg.llm?.base_url || 'https://api.openai.com/v1',
          model: cfg.llm?.model || '',
        },
      };
      setConfig(patched);
      const configuredProvider = patched.llm?.provider?.toLowerCase() || 'bing';
      const current = ['google', 'bing', 'openai'].includes(configuredProvider)
        ? configuredProvider as 'bing' | 'google' | 'openai'
        : 'bing';
      patched.llm.provider = current;
      setConfig(patched);
      setActiveEngine(current);
    }).catch(console.error);
  }, []);

  const handleTestConnection = async () => {
    if (!config) return;
    if (!remoteModels.hasModel(config.llm.model || '')) {
      setTestResult({ success: false, message: tr('请先获取远程模型并选择模型。', 'Fetch the remote models and select a model first.') });
      return;
    }
    setTestingConn(true);
    setTestResult(null);
    try {
      const res = await api.testOpenAITranslate(
        config.llm.base_url || '',
        config.llm.api_key || '',
        config.llm.model
      );
      setTestResult({ success: true, message: res });
    } catch (err) {
      setTestResult({
        success: false,
        message: err instanceof Error ? err.message : String(err),
      });
    } finally {
      setTestingConn(false);
    }
  };

  const handleSave = async (engineToSet?: string) => {
    if (!config) return;
    clearSaved();
    const targetProvider = engineToSet || config.llm.provider || activeEngine;
    if ((activeEngine === 'openai' || targetProvider === 'openai') && !remoteModels.hasModel(config.llm.model || '')) {
      setErrorMsg(tr('请先从当前接口获取并选择远程模型。', 'Fetch and select a model from the current service first.'));
      return;
    }
    setLoading(true);
    setErrorMsg(null);
    try {
      const updatedConfig = {
        ...config,
        llm: {
          ...config.llm,
          provider: targetProvider,
        },
      };

      await api.updateConfig(configFormUpdates('translation', updatedConfig));

      setConfig(updatedConfig);
      showSaved();
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  const handleSetDefault = (engineId: 'bing' | 'google' | 'openai') => {
    if (!config) return;
    void handleSave(engineId);
  };

  if (!config) {
    return (
      <div className="h-screen flex items-center justify-center text-xs text-slate-400">
        {tr('正在读取翻译引擎配置...', 'Loading translation settings...')}
      </div>
    );
  }

  const isCurrentDefault = (config.llm.provider || 'bing').toLowerCase() === activeEngine;

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6 select-none bg-[#F5F5F7] dark:bg-[#121215]">
      <div className="max-w-5xl w-full mx-auto space-y-6 pt-2 pb-16">
        {/* 顶部标题区 */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
                {tr('翻译引擎设置', 'Translation Engine Settings')}
              </h2>
              <span className="px-2 py-0.5 text-[11px] font-medium bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20 rounded-md">
                {tr('当前默认：', 'Current default: ')}{engineDetails[config.llm.provider as keyof typeof engineDetails]?.name || config.llm.provider}
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {tr('左侧选择翻译服务，右侧进行详细配置与参数调校', 'Choose a translation service on the left and configure it on the right.')}
            </p>
          </div>
        </div>

        {/* 提示条 */}


        {errorMsg && (
          <div className="p-3 bg-rose-50 border border-rose-200 dark:bg-rose-500/10 dark:border-rose-500/30 rounded-xl flex items-center gap-2 text-xs text-rose-600 dark:text-rose-300 animate-in fade-in">
            <AlertCircle size={16} />
            <span>{tr('保存失败：', 'Save failed: ')}{errorMsg}</span>
          </div>
        )}

        {/* 主从二级布局卡片 (Master-Detail) */}
        <div className="grid grid-cols-12 gap-5 min-h-[520px]">
          {/* 左侧菜单 */}
          <div className="col-span-4 bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-[#2C2C2E] rounded-2xl p-3 flex flex-col justify-between">
            <div className="space-y-1.5">
              <div className="px-3 py-2 text-[11px] font-semibold text-slate-400 dark:text-slate-500 tracking-wider uppercase">
                {tr('支持的翻译供应商', 'Translation Providers')}
              </div>
              {ENGINES.map((item) => {
                const isSelected = activeEngine === item.id;
                const isDefault = (config.llm.provider || 'bing').toLowerCase() === item.id;

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
                      <span>{engineDetails[item.id].name}</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      {engineDetails[item.id].badge && (
                        <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 rounded">
                          {engineDetails[item.id].badge}
                        </span>
                      )}
                      {isDefault && (
                        <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-blue-600 text-white rounded flex items-center gap-0.5">
                          <Check size={10} /> {tr('默认', 'Default')}
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
                const currentMeta = engineDetails[activeEngine];
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
                      {currentMeta?.desc && (
                        <p className="text-xs text-slate-500 dark:text-slate-400">
                          {currentMeta.desc}
                        </p>
                      )}
                    </div>
                    <div>
                      {isCurrentDefault ? (
                        <span className="px-3 py-1.5 bg-emerald-50 dark:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-500/20 rounded-xl text-xs font-semibold flex items-center gap-1.5">
                          <Check size={14} /> {tr('默认引擎', 'Default Engine')}
                        </span>
                      ) : (
                        <button
                          type="button"
                          onClick={() => handleSetDefault(activeEngine)}
                          className="px-3.5 py-1.5 bg-blue-600 hover:bg-blue-500 active:scale-95 text-white rounded-xl text-xs font-semibold transition-all flex items-center gap-1.5"
                        >
                          {tr('设为默认引擎', 'Set as Default')}
                        </button>
                      )}
                    </div>
                  </div>
                );
              })()}

              {/* 公共翻译服务配置 */}
              {activeEngine === 'google' && (
                <div className="space-y-4 animate-in fade-in">
                  <div className="p-4 rounded-xl bg-blue-50/60 dark:bg-blue-500/10 border border-blue-100 dark:border-blue-500/20 text-xs text-blue-900 dark:text-blue-200 space-y-2">
                    <div className="font-semibold flex items-center gap-1.5">
                      <Globe size={16} /> {tr('谷歌公共翻译通道 · 免配置即开即用', 'Google Translate · Ready to use')}
                    </div>
                    <p className="text-[12px] leading-relaxed opacity-90">
                      {tr('谷歌翻译支持近百种语言，速度快，无需配置密钥或自建代理。', 'Google Translate supports nearly 100 languages and requires no API key or proxy.')}
                    </p>
                  </div>

                  <div className="grid grid-cols-2 gap-4 pt-2">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('单次切片句子数', 'Sentences per Batch')}
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
                      <p className="text-[11px] text-slate-400 mt-1">{tr('推荐 10~15 句，兼顾上下文连贯与响应速度', '10–15 sentences balances context and speed.')}</p>
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('默认目标语言', 'Default Target Language')}
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
                      <p className="text-[11px] text-slate-400 mt-1">{tr('例如 zh-CN、en、ja、ko 等', 'For example: zh-CN, en, ja, ko')}</p>
                    </div>
                  </div>
                </div>
              )}

              {/* 必应翻译配置 */}
              {activeEngine === 'bing' && (
                <div className="space-y-4 animate-in fade-in">
                  <div className="p-4 rounded-xl bg-cyan-50/60 dark:bg-cyan-500/10 border border-cyan-100 dark:border-cyan-500/20 text-xs text-cyan-900 dark:text-cyan-200 space-y-2">
                    <div className="font-semibold flex items-center gap-1.5">
                      <Cloud size={16} /> {tr('必应翻译 · Edge 浏览器通道', 'Bing Translator · Edge service')}
                    </div>
                    <p className="text-[12px] leading-relaxed opacity-90">
                      {tr('使用必应翻译服务；端点不可用时自动回退到谷歌翻译。', 'Uses Bing translation and falls back to Google when the service is unavailable.')}
                    </p>
                  </div>

                  <div className="grid grid-cols-2 gap-4 pt-2">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('故障自动回退', 'Automatic Fallback')}
                      </label>
                      <div className="h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-600 dark:text-slate-300 flex items-center">
                        <span className="w-2 h-2 rounded-full bg-emerald-500 mr-2" />
                        {tr('已启用 · 自动回退至谷歌翻译', 'Enabled · Falls back to Google Translate')}
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('单次切片句子数', 'Sentences per Batch')}
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
                        {tr('接口地址 Base URL', 'API Base URL')}
                      </label>
                      <input
                        type="text"
                        placeholder="http://localhost:11434/v1 or https://api.openai.com/v1"
                        value={config.llm.base_url}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            llm: { ...config.llm, base_url: e.target.value },
                          })
                        }
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                      <p className="text-[11px] text-slate-400 mt-1">{tr('兼容 OpenAI 规范的任何大模型接口', 'Any service compatible with the OpenAI API')}</p>
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('访问密钥 API Key', 'API Key')}
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
                      <p className="text-[11px] text-slate-400 mt-1">{tr('本地 Ollama 无需填 Key', 'A key is not required for local Ollama.')}</p>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('模型名称 Model', 'Model Name')}
                      </label>
                      <RemoteModelSelect catalog={remoteModels} value={config.llm.model || ''}
                        hasAddress={!!config.llm.base_url?.trim()}
                        onChange={model => setConfig({ ...config, llm: { ...config.llm, model } })} />
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('每次并发句子数', 'Concurrent Sentences')}
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
                      {tr('自定义翻译 Prompt 提示词', 'Custom Translation Prompt')}
                    </label>
                    <textarea
                      rows={3}
                      placeholder={tr('留空使用内置专业视频字幕翻译提示词。可用变量：{{target_lang}} {{count}}', 'Leave blank to use the built-in subtitle prompt. Variables: {{target_lang}} {{count}}')}
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

                  {/* 接口连通性测试 */}
                  <div className="pt-2 flex items-center justify-between">
                    <button
                      type="button"
                      onClick={handleTestConnection}
                      disabled={testingConn}
                      className="px-3.5 py-1.5 rounded-xl border border-blue-500/30 bg-blue-50/70 hover:bg-blue-100 dark:bg-blue-500/10 dark:hover:bg-blue-500/20 text-blue-600 dark:text-blue-400 text-xs font-semibold flex items-center gap-1.5 transition-all active:scale-95 disabled:opacity-50"
                    >
                      {testingConn ? (
                        <>
                          <Loader2 size={13} className="animate-spin" />
                          <span>{tr('正在测试端点连通性...', 'Testing endpoint...')}</span>
                        </>
                      ) : (
                        <>
                          <Zap size={13} />
                          <span>{tr('测试端点连通性', 'Test Endpoint')}</span>
                        </>
                      )}
                    </button>

                    {testResult && (
                      <div
                        className={`text-xs px-3 py-1.5 rounded-xl border flex items-center gap-1.5 animate-in fade-in ${
                          testResult.success
                            ? 'bg-emerald-50 dark:bg-emerald-500/10 border-emerald-200 dark:border-emerald-500/20 text-emerald-600 dark:text-emerald-400 font-medium'
                            : 'bg-rose-50 dark:bg-rose-500/10 border-rose-200 dark:border-rose-500/20 text-rose-600 dark:text-rose-400'
                        }`}
                      >
                        {testResult.success ? <CheckCircle2 size={13} /> : <AlertCircle size={13} />}
                        <span>{testResult.message}</span>
                      </div>
                    )}
                  </div>
                </div>
              )}

            </div>

            <div className="pt-4 border-t border-slate-100 dark:border-white/5 flex items-center justify-end text-xs text-slate-500">
              <button
                type="button"
                onClick={() => handleSave()}
                disabled={loading}
                className={`w-36 px-5 py-2 rounded-xl text-xs font-bold text-white whitespace-nowrap transition-[background-color,transform] duration-200 flex items-center justify-center gap-1.5 active:scale-[0.98] disabled:opacity-50 ${saved ? 'bg-emerald-600 hover:bg-emerald-600' : 'bg-blue-600 hover:bg-blue-500'}`}
              >
                {saved ? <Check size={14} /> : <Save size={14} />}
                <span role="status">{loading ? tr('正在保存...', 'Saving...') : saved ? tr('保存成功', 'Saved') : t('common.save')}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
