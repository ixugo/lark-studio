import React, { useEffect, useState } from 'react';
import { Save, Languages, Volume2, Cpu, Check, AlertCircle } from 'lucide-react';
import { api } from '../lib/api';
import { ConfigDTO } from '../types';

export const SettingsView: React.FC = () => {
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [activeTab, setActiveTab] = useState<'llm' | 'tts' | 'whisper'>('llm');
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    api.getConfig().then(setConfig).catch(console.error);
  }, []);

  const handleSave = async () => {
    if (!config) return;
    try {
      await api.updateConfig({
        pipeline: config.pipeline,
        llm: config.llm,
        tts: config.tts,
      });
      setSavedSuccess(true);
      setTimeout(() => setSavedSuccess(false), 2500);
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
    }
  };

  if (!config) {
    return <div className="p-8 text-center text-xs text-apple-muted">加载配置中...</div>;
  }

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6">
      <div className="max-w-4xl w-full mx-auto space-y-6">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-apple-text dark:text-apple-darkText">
            全局设置中心
          </h2>
          <p className="text-xs text-apple-muted mt-1">
            配置 OpenAI 兼容模型、DeepLX、本地 Whisper 引擎与 Edge 配音
          </p>
        </div>

        {/* 提示信息 */}
        {savedSuccess && (
          <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-xl flex items-center space-x-2 text-xs text-emerald-500">
            <Check size={16} />
            <span>配置已成功保存并实时生效！</span>
          </div>
        )}
        {errorMsg && (
          <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center space-x-2 text-xs text-red-500">
            <AlertCircle size={16} />
            <span>保存失败: {errorMsg}</span>
          </div>
        )}

        {/* 设置分类 Tab */}
        <div className="flex space-x-2 border-b border-apple-border dark:border-apple-darkBorder pb-2">
          {[
            { id: 'llm' as const, label: '翻译引擎 (LLM / DeepLX)', icon: <Languages size={15} /> },
            { id: 'tts' as const, label: '配音服务 (TTS)', icon: <Volume2 size={15} /> },
            { id: 'whisper' as const, label: '语音识别 (Whisper)', icon: <Cpu size={15} /> },
          ].map((item) => (
            <button
              key={item.id}
              onClick={() => setActiveTab(item.id)}
              className={`flex items-center space-x-2 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${
                activeTab === item.id
                  ? 'bg-apple-accent text-white shadow-sm'
                  : 'text-apple-muted hover:text-apple-text dark:hover:text-apple-darkText hover:bg-black/5 dark:hover:bg-white/5'
              }`}
            >
              {item.icon}
              <span>{item.label}</span>
            </button>
          ))}
        </div>

        {/* 翻译设置板块 */}
        {activeTab === 'llm' && (
          <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-5 shadow-sm space-y-4">
            <h3 className="text-xs font-bold text-apple-text dark:text-apple-darkText">
              大语言模型与翻译接口
            </h3>
            <p className="text-[11px] text-apple-muted">
              支持任何 OpenAI 兼容规范的接口（如 One-API, New-API, Ollama, DeepSeek 等）。
            </p>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-[11px] text-apple-muted block mb-1">默认翻译供应商</label>
                <select
                  value={config.llm.provider}
                  onChange={(e) =>
                    setConfig({ ...config, llm: { ...config.llm, provider: e.target.value } })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                >
                  <option value="openai">OpenAI / 本地自建兼容接口</option>
                  <option value="bing">必应免费翻译</option>
                  <option value="deeplx">DeepLX 翻译</option>
                </select>
              </div>

              <div>
                <label className="text-[11px] text-apple-muted block mb-1">模型名称 (Model)</label>
                <input
                  type="text"
                  placeholder="如 gpt-4o-mini 或 deepseek-chat"
                  value={config.llm.model}
                  onChange={(e) =>
                    setConfig({ ...config, llm: { ...config.llm, model: e.target.value } })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                />
              </div>

              <div className="col-span-2">
                <label className="text-[11px] text-apple-muted block mb-1">API Base URL</label>
                <input
                  type="text"
                  placeholder="如 https://api.openai.com/v1"
                  value={config.llm.base_url}
                  onChange={(e) =>
                    setConfig({ ...config, llm: { ...config.llm, base_url: e.target.value } })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent font-mono"
                />
              </div>

              <div className="col-span-2">
                <label className="text-[11px] text-apple-muted block mb-1">API Key</label>
                <input
                  type="password"
                  placeholder="sk-..."
                  value={config.llm.api_key}
                  onChange={(e) =>
                    setConfig({ ...config, llm: { ...config.llm, api_key: e.target.value } })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent font-mono"
                />
              </div>

              <div className="col-span-2">
                <label className="text-[11px] text-apple-muted block mb-1">DeepLX 专用 URL (选填)</label>
                <input
                  type="text"
                  placeholder="如 http://127.0.0.1:1188/translate"
                  value={config.llm.deeplx_url}
                  onChange={(e) =>
                    setConfig({ ...config, llm: { ...config.llm, deeplx_url: e.target.value } })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent font-mono"
                />
              </div>
            </div>
          </div>
        )}

        {/* TTS 设置板块 */}
        {activeTab === 'tts' && (
          <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-5 shadow-sm space-y-4">
            <h3 className="text-xs font-bold text-apple-text dark:text-apple-darkText">
              语音合成引擎设置
            </h3>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-[11px] text-apple-muted block mb-1">默认配音引擎</label>
                <select
                  value={config.tts.type}
                  onChange={(e) =>
                    setConfig({ ...config, tts: { ...config.tts, type: e.target.value } })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                >
                  <option value="edge">Edge-TTS (免 API Key，自然流畅)</option>
                  <option value="openai">OpenAI TTS</option>
                </select>
              </div>

              <div>
                <label className="text-[11px] text-apple-muted block mb-1">默认音色</label>
                <input
                  type="text"
                  value={config.tts.voice}
                  onChange={(e) =>
                    setConfig({ ...config, tts: { ...config.tts, voice: e.target.value } })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                />
              </div>
            </div>
          </div>
        )}

        {/* Whisper 设置板块 */}
        {activeTab === 'whisper' && (
          <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-5 shadow-sm space-y-4">
            <h3 className="text-xs font-bold text-apple-text dark:text-apple-darkText">
              Whisper 本地语音识别设置
            </h3>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-[11px] text-apple-muted block mb-1">模型选择</label>
                <input
                  type="text"
                  value={config.pipeline.whisper_model}
                  onChange={(e) =>
                    setConfig({
                      ...config,
                      pipeline: { ...config.pipeline, whisper_model: e.target.value },
                    })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent font-mono"
                />
              </div>

              <div>
                <label className="text-[11px] text-apple-muted block mb-1">并发工作协程数</label>
                <input
                  type="number"
                  min="1"
                  max="8"
                  value={config.pipeline.workers}
                  onChange={(e) =>
                    setConfig({
                      ...config,
                      pipeline: { ...config.pipeline, workers: parseInt(e.target.value, 10) || 1 },
                    })
                  }
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-2 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                />
              </div>
            </div>
          </div>
        )}
      </div>

      {/* 底部保存按钮 */}
      <div className="max-w-4xl w-full mx-auto pt-4 border-t border-apple-border dark:border-apple-darkBorder flex justify-end">
        <button
          onClick={handleSave}
          className="flex items-center space-x-1.5 bg-apple-accent hover:bg-apple-accentHover text-white px-5 py-2 rounded-xl text-xs font-semibold shadow-sm transition-all"
        >
          <Save size={15} />
          <span>保存所有配置</span>
        </button>
      </div>
    </div>
  );
};
