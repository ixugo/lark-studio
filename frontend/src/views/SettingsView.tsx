import React, { useEffect, useState } from 'react';
import {
  Save,
  Languages,
  Volume2,
  Cpu,
  CheckCircle2,
  AlertCircle,
} from 'lucide-react';
import { api } from '../lib/api';
import { ConfigDTO } from '../types';

export const SettingsView: React.FC = () => {
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.getConfig().then(setConfig).catch(console.error);
  }, []);

  const handleSave = async () => {
    if (!config) return;
    setLoading(true);
    setErrorMsg(null);
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
    } finally {
      setLoading(false);
    }
  };

  if (!config) {
    return (
      <div className="h-screen flex items-center justify-center text-xs text-slate-400">
        正在加载全局配置...
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6 select-none">
      <div className="max-w-4xl w-full mx-auto space-y-6 pt-4 pb-12">
        {/* 顶部标题 */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
              全局设置中心
            </h2>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              配置翻译大模型、Edge/OpenAI 语音合成引擎、Whisper 语音识别及流水线参数
            </p>
          </div>
        </div>

        {/* 提示条 */}
        {savedSuccess && (
          <div className="p-3.5 bg-emerald-500/10 border border-emerald-500/20 rounded-xl flex items-center space-x-2 text-xs text-emerald-600 dark:text-emerald-400 animate-in fade-in">
            <CheckCircle2 size={16} />
            <span>全局配置已成功保存并实时生效！</span>
          </div>
        )}

        {errorMsg && (
          <div className="p-3.5 bg-rose-50 border border-rose-200 dark:bg-rose-500/10 dark:border-rose-500/30 rounded-xl flex items-center gap-2 text-xs text-rose-600 dark:text-rose-300 animate-in fade-in">
            <AlertCircle size={16} />
            <span>保存失败: {errorMsg}</span>
          </div>
        )}

        {/* 模块 1：翻译引擎 (LLM / 翻译服务) */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-6 shadow-sm space-y-5">
          <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-white/5">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-xl bg-blue-600/10 text-blue-600 dark:bg-blue-500/15 dark:text-blue-400 flex items-center justify-center font-bold">
                <Languages className="w-4 h-4" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-800 dark:text-slate-100">
                  翻译引擎与大语言模型 (LLM)
                </h3>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  支持微软必应免配置翻译、自定义 OpenAI 兼容协议（DeepSeek/Ollama/通义）及自建 DeepLX
                </p>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {/* 默认翻译服务商 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                默认翻译供应商 (Provider)
              </label>
              <select
                value={config.llm.provider}
                onChange={(e) =>
                  setConfig({ ...config, llm: { ...config.llm, provider: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              >
                <option value="bing">必应免费翻译 (免 API Key · 开箱即用)</option>
                <option value="openai">OpenAI 兼容接口 (DeepSeek / ChatGPT / 本地 Ollama)</option>
                <option value="deeplx">DeepLX 自建翻译服务</option>
              </select>
            </div>

            {/* 模型名称 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                模型名称 (Model)
              </label>
              <input
                type="text"
                placeholder="例如 deepseek-chat, gpt-4o-mini, qwen-plus"
                value={config.llm.model}
                onChange={(e) =>
                  setConfig({ ...config, llm: { ...config.llm, model: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              />
            </div>

            {/* OpenAI API Base URL */}
            <div className="sm:col-span-2">
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                OpenAI 兼容 API Base URL
              </label>
              <input
                type="text"
                placeholder="例如 https://api.deepseek.com/v1 或 http://localhost:11434/v1"
                value={config.llm.base_url}
                onChange={(e) =>
                  setConfig({ ...config, llm: { ...config.llm, base_url: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 font-mono"
              />
            </div>

            {/* API Key */}
            <div className="sm:col-span-2">
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                API 密钥 (API Key)
              </label>
              <input
                type="password"
                placeholder="sk-..."
                value={config.llm.api_key}
                onChange={(e) =>
                  setConfig({ ...config, llm: { ...config.llm, api_key: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 font-mono"
              />
            </div>

            {/* DeepLX URL */}
            <div className="sm:col-span-2">
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                DeepLX 自建接口地址 (仅当选择 deeplx 时生效)
              </label>
              <input
                type="text"
                placeholder="例如 http://127.0.0.1:1188/translate"
                value={config.llm.deeplx_url}
                onChange={(e) =>
                  setConfig({ ...config, llm: { ...config.llm, deeplx_url: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 font-mono"
              />
            </div>

            {/* 自定义翻译提示词 */}
            <div className="sm:col-span-2">
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                自定义系统翻译提示词 (System Prompt，留空使用内置优化提示词)
              </label>
              <textarea
                rows={2}
                placeholder="留空自动采用系统内置的最佳口语化与意译 Prompt"
                value={config.pipeline.translate_prompt}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    pipeline: { ...config.pipeline, translate_prompt: e.target.value },
                  })
                }
                className="w-full bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl p-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 resize-none leading-relaxed"
              />
            </div>
          </div>
        </div>

        {/* 模块 2：配音引擎 (TTS Engine) */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-6 shadow-sm space-y-5">
          <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-white/5">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-xl bg-purple-600/10 text-purple-600 dark:bg-purple-500/15 dark:text-purple-400 flex items-center justify-center font-bold">
                <Volume2 className="w-4 h-4" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-800 dark:text-slate-100">
                  配音引擎与语音合成 (TTS)
                </h3>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  默认搭载微软 Edge TTS 高清免配置声音，亦可对接云端商业 OpenAI 语音协议
                </p>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {/* 默认配音引擎 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                默认配音引擎类型
              </label>
              <select
                value={config.tts.type}
                onChange={(e) =>
                  setConfig({ ...config, tts: { ...config.tts, type: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              >
                <option value="edge">Edge TTS (微软免费高质量音色 · 推荐)</option>
                <option value="openai">OpenAI 兼容 TTS 服务</option>
              </select>
            </div>

            {/* 默认音色 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                默认音色名称
              </label>
              <input
                type="text"
                placeholder="例如 zh-CN-YunxiNeural 或 zh-CN-XiaoxiaoNeural"
                value={config.tts.voice}
                onChange={(e) =>
                  setConfig({ ...config, tts: { ...config.tts, voice: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              />
            </div>

            {/* OpenAI TTS Base URL */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                OpenAI TTS API Base URL (可选)
              </label>
              <input
                type="text"
                placeholder="https://api.openai.com/v1"
                value={config.tts.base_url}
                onChange={(e) =>
                  setConfig({ ...config, tts: { ...config.tts, base_url: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 font-mono"
              />
            </div>

            {/* OpenAI TTS API Key */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                OpenAI TTS API Key (可选)
              </label>
              <input
                type="password"
                placeholder="sk-..."
                value={config.tts.api_key}
                onChange={(e) =>
                  setConfig({ ...config, tts: { ...config.tts, api_key: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 font-mono"
              />
            </div>

            {/* OpenAI TTS 模型名称 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                OpenAI TTS 模型 (Model)
              </label>
              <input
                type="text"
                placeholder="tts-1 或 tts-1-hd"
                value={config.tts.model}
                onChange={(e) =>
                  setConfig({ ...config, tts: { ...config.tts, model: e.target.value } })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              />
            </div>

            {/* TTS 并发协程数 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                TTS 并发协程数 (1 ~ 4)
              </label>
              <input
                type="number"
                min="1"
                max="4"
                value={config.pipeline.tts_workers}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    pipeline: {
                      ...config.pipeline,
                      tts_workers: parseInt(e.target.value, 10) || 2,
                    },
                  })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              />
            </div>
          </div>
        </div>

        {/* 模块 3：语音识别与核心流水线 (Whisper & Pipeline) */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-6 shadow-sm space-y-5">
          <div className="flex items-center justify-between pb-3 border-b border-slate-100 dark:border-white/5">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-xl bg-emerald-600/10 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-400 flex items-center justify-center font-bold">
                <Cpu className="w-4 h-4" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-800 dark:text-slate-100">
                  语音识别 (Whisper) 与核心流水线
                </h3>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  底层多核调度并发 Worker、Whisper 离线模型与 FFmpeg 工具配置
                </p>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {/* Whisper 模型文件 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                Whisper 模型文件路径 / 标识
              </label>
              <input
                type="text"
                placeholder="例如 large-v3-turbo 或本地 ggml 绝对路径"
                value={config.pipeline.whisper_model}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    pipeline: { ...config.pipeline, whisper_model: e.target.value },
                  })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 font-mono"
              />
            </div>

            {/* 并行 Worker 数量 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                流水线并行任务 Workers 数 (1 ~ 8)
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
                      workers: parseInt(e.target.value, 10) || 1,
                    },
                  })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              />
            </div>

            {/* FFmpeg 路径 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                FFmpeg 可执行路径 (留空默认使用内置或环境变量中的 ffmpeg)
              </label>
              <input
                type="text"
                placeholder="ffmpeg"
                value={config.pipeline.ffmpeg_bin}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    pipeline: { ...config.pipeline, ffmpeg_bin: e.target.value },
                  })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 font-mono"
              />
            </div>

            {/* 默认目标语言 */}
            <div>
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block mb-1.5">
                默认目标语言代码
              </label>
              <input
                type="text"
                placeholder="zh-CN"
                value={config.pipeline.default_target_lang}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    pipeline: { ...config.pipeline, default_target_lang: e.target.value },
                  })
                }
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40"
              />
            </div>

            {/* 清理中间产物 */}
            <div className="sm:col-span-2 flex items-center justify-between p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-200/80 dark:border-white/[0.08]">
              <div>
                <div className="text-xs font-bold text-slate-800 dark:text-white">
                  处理完成后自动清理中间产物
                </div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">
                  任务成功后自动删除 raw.mp3 及切片音频片段，节约磁盘空间
                </div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
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
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-slate-200 dark:bg-white/20 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 shadow-inner" />
              </label>
            </div>
          </div>
        </div>
      </div>

      {/* 底部动作保存条 */}
      <div className="shrink-0 bg-white/95 dark:bg-[#1C1C1E]/95 backdrop-blur-2xl border-t border-slate-200/90 dark:border-[#2C2C2E] px-8 py-4 flex items-center justify-between shadow-sm z-20">
        <div className="text-xs text-slate-500 dark:text-slate-400">
          修改后点击保存，所有流水线及引擎变更将即刻写入系统配置文件。
        </div>

        <button
          type="button"
          onClick={handleSave}
          disabled={loading}
          className="px-6 py-2.5 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white transition-all flex items-center gap-2 shadow-sm active:scale-95 disabled:opacity-50"
        >
          <Save size={15} />
          <span>{loading ? '正在保存...' : '保存所有设置'}</span>
        </button>
      </div>
    </div>
  );
};
