import { configFormUpdates, ttsVoiceForEngine } from '../lib/configForms';
import React, { useEffect, useRef, useState } from 'react';
import {
  Volume2,
  Sparkles,
  AlertCircle,
  Save,
  Check,
  Eye,
  EyeOff,
  ShieldCheck,
  Play,
  Loader2,
} from 'lucide-react';
import { api } from '../lib/api';
import { ConfigDTO, TTSCapabilities } from '../types';
import { useTranslation } from '../i18n';
import { RemoteModelSelect, useRemoteModels } from '../components/RemoteModelSelect';
import { useSaveFeedback } from '../lib/useSaveFeedback';
import { validateTTSSelection } from '../lib/ttsCapabilities';
import {
  EDGE_TTS_VOICES,
  normalizeTtsVoice,
  TtsEngine,
} from '../lib/ttsVoices';

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

export const TtsEngineView: React.FC = () => {
  const { locale, t } = useTranslation();
  const english = locale === 'en-US';
  // 让配置页所有说明与全局语言切换保持一致。
  const tr = (zh: string, en: string) => english ? en : zh;
  const [config, setConfig] = useState<ConfigDTO | null>(null);
  const [activeEngine, setActiveEngine] = useState<'edge' | 'openai'>('edge');
  const [showApiKey, setShowApiKey] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { saved, clearSaved, showSaved } = useSaveFeedback();
  const [testingTTS, setTestingTTS] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const remoteEnabled = !!config && (activeEngine === 'openai' || config.tts.type === 'openai');
  const baseURL = config?.tts.base_url || '';
  const apiKey = config?.tts.api_key || '';
  const model = config?.tts.model || '';
  const remoteModels = useRemoteModels(baseURL, apiKey, remoteEnabled);
  const modelConfirmed = remoteModels.hasModel(model);
  const [capabilityState, setCapabilityState] = useState<{
    baseURL: string; apiKey: string; model: string; value: TTSCapabilities | null; error: string;
  } | null>(null);
  const capabilityRequest = useRef(0);
  const capabilities = modelConfirmed && capabilityState?.baseURL === baseURL &&
    capabilityState.apiKey === apiKey && capabilityState.model === model ? capabilityState.value : null;
  const capabilityError = capabilityState?.baseURL === baseURL && capabilityState.apiKey === apiKey &&
    capabilityState.model === model ? capabilityState.error : '';

  useEffect(() => {
    const request = ++capabilityRequest.current;
    setCapabilityState(null);
    if (!modelConfirmed) return;
    api.getTTSCapabilities(baseURL, apiKey, model).then(value => {
      if (request === capabilityRequest.current) setCapabilityState({ baseURL, apiKey, model, value, error: '' });
    }).catch(error => {
      if (request === capabilityRequest.current) setCapabilityState({ baseURL, apiKey, model, value: null, error: error instanceof Error ? error.message : String(error) });
    });
    return () => { ++capabilityRequest.current; };
  }, [baseURL, apiKey, model, modelConfirmed]);
  const engineDetails = {
    edge: { name: 'Edge TTS', tag: tr('免 Key 推荐', 'Recommended · No Key'), desc: tr('微软语音服务，音色自然且无需单独配置密钥。', 'Microsoft speech service with natural voices and no API key setup.'), badge: tr('推荐', 'Recommended') },
    openai: { name: 'OpenAI TTS', tag: tr('兼容接口', 'Compatible API'), desc: tr('可连接 OpenAI 兼容的语音合成服务。', 'Connect to any OpenAI-compatible speech synthesis service.'), badge: undefined },
  };

  useEffect(() => {
    api.getConfig().then((cfg) => {
      const configuredType = cfg.tts?.type?.toLowerCase();
      const current: TtsEngine = configuredType === 'openai' ? 'openai' : 'edge';
      const patched: ConfigDTO = {
        ...cfg,
        pipeline: {
          ...cfg.pipeline,
          max_speed_factor: cfg.pipeline?.max_speed_factor ?? 1.2,
          tts_workers: cfg.pipeline?.tts_workers ?? 2,
        },
        tts: {
          ...cfg.tts,
          type: current,
          voice: current === 'openai' ? cfg.tts.openai_voice ?? cfg.tts.voice ?? '' : ttsVoiceForEngine(current, cfg.tts),
          model: cfg.tts?.model || '',
          language: cfg.tts?.language || 'Auto',
          instructions: cfg.tts?.instructions || '',
          base_url: cfg.tts?.base_url || 'https://api.openai.com/v1',
        },
      };
      setConfig(patched);
      setActiveEngine(current);
    }).catch(console.error);
  }, []);

  const validateRemoteTTS = (voice: string) => {
    if (!config || !modelConfirmed) return tr('请先获取并选择远程模型。', 'Fetch and select a remote model first.');
    if (!capabilities) return capabilityError || tr('请等待语音服务能力查询完成。', 'Wait for the speech service capabilities.');
    const error = validateTTSSelection(capabilities, { ...config.tts, voice });
    const messages = {
      voice: tr('请选择当前模型支持的音色。', 'Select a voice supported by the current model.'),
      instructions: tr('当前模型不支持情绪指令，请清空后保存。', 'This model does not support instructions. Clear them before saving.'),
      instructions_length: tr('情绪指令不能超过 4096 字符。', 'Instructions cannot exceed 4096 characters.'),
      language: tr('请选择当前模型支持的语言。', 'Select a language supported by the current model.'),
    };
    return error ? messages[error] : '';
  };

  const handleSave = async (engineToSet?: string) => {
    if (!config) return;
    clearSaved();
    const requestedType = engineToSet || config.tts.type || activeEngine;
    const targetType: TtsEngine = requestedType === 'openai' ? 'openai' : 'edge';
    const needsRemote = activeEngine === 'openai' || targetType === 'openai';
    const remoteVoice = activeEngine === 'openai' ? config.tts.voice : ttsVoiceForEngine('openai', config.tts);
    const error = needsRemote ? validateRemoteTTS(remoteVoice) : '';
    if (error) { setErrorMsg(error); return; }
    setLoading(true);
    setErrorMsg(null);
    try {
      const selectedVoice = activeEngine === 'openai' ? config.tts.voice : normalizeTtsVoice('edge', config.tts.voice);
      const updatedConfig = {
        ...config,
        tts: {
          ...config.tts,
          type: targetType,
          voice: selectedVoice,
          [activeEngine === 'openai' ? 'openai_voice' : 'edge_voice']: selectedVoice,
          protocol: needsRemote ? capabilities!.protocol : config.tts.protocol,
        },
      };

      const updates = configFormUpdates('tts', updatedConfig, activeEngine, targetType);
      if (activeEngine === 'openai' && updates.tts) {
        updates.tts.openai_voice = selectedVoice;
        if (targetType === 'openai') updates.tts.voice = selectedVoice;
      }
      await api.updateConfig(updates);

      setConfig(updatedConfig);
      showSaved();
    } catch (err) {
      setErrorMsg(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  const handleSetDefault = (engineId: 'edge' | 'openai') => {
    if (!config) return;
    void handleSave(engineId);
  };

  const handleSelectEngine = (engine: TtsEngine) => {
    setActiveEngine(engine);
    setConfig((current) => current ? {
      ...current,
      tts: {
        ...current.tts,
        [activeEngine === 'openai' ? 'openai_voice' : 'edge_voice']: activeEngine === 'openai' ? current.tts.voice : normalizeTtsVoice('edge', current.tts.voice),
        voice: engine === 'openai' ? current.tts.openai_voice ?? (activeEngine === 'openai' ? current.tts.voice : '') : ttsVoiceForEngine(engine, current.tts),
      },
    } : current);
  };

  // 统一试音：调用选定引擎朗读指定文本并触发播放。
  const handleTestTTS = async () => {
    if (!config) return;
    if (activeEngine === 'openai') {
      const error = validateRemoteTTS(config.tts.voice);
      if (error) {
        setTestResult({ success: false, message: error });
        return;
      }
    }
    setTestingTTS(true);
    setTestResult(null);
    try {
      const isEnglishVoice = config.tts.voice?.toLowerCase().startsWith('en-') || config.tts.voice?.toLowerCase().startsWith('en_');
      const text = isEnglishVoice ? 'Hello, I am using Lark Studio.' : '你好，我正在使用云雀工坊';
      let audioURL = '';
      if (activeEngine === 'edge') {
        audioURL = await api.testEdgeTTS(config.tts.voice || 'zh-CN-XiaoxiaoNeural', text);
      } else {
        audioURL = await api.testConfiguredTTS({ ...config.tts, type: 'openai', protocol: capabilities!.protocol }, text);
      }
      audioRef.current?.pause();
      audioRef.current = new Audio(audioURL);
      await audioRef.current.play();
      setTestResult({ success: true, message: locale === 'en-US' ? 'Audition is playing.' : '试音播放中' });
    } catch (err) {
      setTestResult({ success: false, message: err instanceof Error ? err.message : String(err) });
    } finally {
      setTestingTTS(false);
    }
  };

  if (!config) {
    return (
      <div className="h-screen flex items-center justify-center text-xs text-slate-400">
        {tr('正在读取语音合成配置...', 'Loading speech synthesis settings...')}
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
                {tr('语音合成引擎', 'Speech Synthesis Engine')}
              </h2>
              <span className="px-2 py-0.5 text-[11px] font-medium bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20 rounded-md">
                {tr('当前默认：', 'Current default: ')}{TTS_ENGINES.find(e => e.id === (config.tts.type || 'edge').toLowerCase())?.name || config.tts.type}
              </span>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {tr('配置视频配音与语音合成音色、并发线程及自动语速调整', 'Configure voices, concurrency, and automatic speech rate for dubbing.')}
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
                {tr('合成引擎列表', 'Synthesis Engines')}
              </div>
              {TTS_ENGINES.map((item) => {
                const isSelected = activeEngine === item.id;
                const isDefault = (config.tts.type || 'edge').toLowerCase() === item.id;

                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => handleSelectEngine(item.id)}
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

            <div className="p-3 bg-slate-50 dark:bg-white/[0.03] border border-slate-200/70 dark:border-white/5 rounded-xl text-[11px] text-slate-500 dark:text-slate-400 space-y-1">
              <div className="flex items-center gap-1 font-semibold text-slate-700 dark:text-slate-300">
                <ShieldCheck size={14} className="text-emerald-500" />
                <span>{tr('智能音画对齐', 'Audio and Video Alignment')}</span>
              </div>
              <p>{tr('系统内置动态调速算法，确保生成的译文配音与原视频片段精准对齐，无吞字、无拖沓。', 'Dynamic speed adjustment keeps translated speech aligned with each source video segment.')}</p>
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
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        {currentMeta?.desc}
                      </p>
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

              {/* Edge TTS 配置 */}
              {activeEngine === 'edge' && (
                <div className="space-y-4 animate-in fade-in">
                  <div>
                    <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('默认神经音色', 'Default Neural Voice')}
                    </label>
                    <div className="flex gap-2">
                      <select
                        value={config.tts.voice}
                        onChange={(e) =>
                          setConfig({
                            ...config,
                            tts: { ...config.tts, voice: e.target.value },
                          })
                        }
                        className="flex-1 h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      >
                        {EDGE_TTS_VOICES.map((voice) => (
                          <option key={voice.value} value={voice.value}>
                            {english ? voice.enLabel : voice.zhLabel}
                          </option>
                        ))}
                      </select>
                      <button
                        type="button"
                        onClick={handleTestTTS}
                        disabled={testingTTS}
                        className="px-4 h-10 rounded-xl bg-blue-600 hover:bg-blue-500 active:scale-95 text-white text-xs font-semibold flex items-center gap-1.5 transition-all shrink-0 shadow-sm shadow-blue-500/20 disabled:opacity-50"
                      >
                        {testingTTS ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} className="fill-current" />}
                        <span>{tr('试音', 'Audition')}</span>
                      </button>
                    </div>
                    {testResult && activeEngine === 'edge' && (
                      <p className={`text-xs mt-1.5 ${testResult.success ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'}`}>
                        {testResult.message}
                      </p>
                    )}
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('TTS 调速上限', 'TTS Speed Limit')}
                      </label>
                      <input
                        type="number"
                        step="0.05"
                        min="1.0"
                        max="2.0"
                        placeholder="1.2"
                        value={config.pipeline.max_speed_factor ?? 1.2}
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
                      <p className="text-[11px] text-slate-400 mt-1">{tr('推荐 1.2 ~ 1.3，译文超时时自动轻微倍速压制', '1.2–1.3 is recommended; speech speeds up slightly when a segment exceeds its duration.')}</p>
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('TTS 并发生成协程数', 'Concurrent TTS Workers')}
                      </label>
                      <input
                        type="number"
                        min={1}
                        max={6}
                        placeholder="2"
                        value={config.pipeline.tts_workers ?? 2}
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
                      <p className="text-[11px] text-slate-400 mt-1">{tr('默认 2 协程并发，兼顾速度与避免被服务端流控', 'Two workers by default balance speed with provider rate limits.')}</p>
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
                        {tr('访问密钥 API Key', 'API Key')}
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
                        {tr('TTS 模型', 'TTS Model')}
                      </label>
                      <RemoteModelSelect catalog={remoteModels} value={config.tts.model || ''}
                        hasAddress={!!config.tts.base_url?.trim()}
                        onChange={model => setConfig({ ...config, tts: { ...config.tts, model } })} />
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                        {tr('默认音色', 'Default Voice')}
                      </label>
                      <div className="flex gap-2">
                        {capabilities?.voices.length ? (
                          <select aria-label={tr('默认音色', 'Default Voice')}
                            value={capabilities.voices.some(voice => voice.id === config.tts.voice) ? config.tts.voice : ''}
                            onChange={event => setConfig({ ...config, tts: { ...config.tts, voice: event.target.value } })}
                            className="flex-1 min-w-0 h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white disabled:opacity-50">
                            <option value="">{tr('请选择音色', 'Select a voice')}</option>
                            {capabilities.voices.map(voice => <option key={voice.id} value={voice.id}>{voice.name || voice.id}</option>)}
                          </select>
                        ) : (
                          <input type="text" disabled={!capabilities || capabilities.voice_source !== 'manual'}
                            placeholder={tr('填写服务端支持的音色名', 'Enter a voice supported by the service')}
                            value={config.tts.voice || ''} maxLength={256}
                            onChange={event => setConfig({ ...config, tts: { ...config.tts, voice: event.target.value } })}
                            className="flex-1 min-w-0 h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white disabled:opacity-50" />
                        )}
                        <button
                          type="button"
                          onClick={handleTestTTS}
                          disabled={testingTTS}
                          className="px-4 h-10 rounded-xl bg-blue-600 hover:bg-blue-500 active:scale-95 text-white text-xs font-semibold flex items-center gap-1.5 transition-all shrink-0 shadow-sm shadow-blue-500/20 disabled:opacity-50"
                        >
                          {testingTTS ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} className="fill-current" />}
                          <span>{tr('试音', 'Audition')}</span>
                        </button>
                      </div>
                      <p className="text-[11px] text-slate-400 mt-1.5">
                        {capabilityError || (!modelConfirmed ? tr('请选择远程模型以获取音色与服务能力。', 'Select a remote model to fetch voices and capabilities.')
                          : !capabilities ? tr('正在查询音色与服务能力…', 'Loading voices and service capabilities…')
                            : capabilities.voice_source === 'qwen_builtin' ? tr('Qwen 内置音色，服务未返回音色列表。', 'Built-in Qwen voices; the service did not return a voice list.')
                              : capabilities.voice_source === 'manual' ? tr('服务未提供音色查询接口，请填写其支持的音色。', 'The service has no voice discovery endpoint. Enter a supported voice.')
                                : tr('音色列表来自当前服务。', 'The voice list comes from the current service.'))}
                      </p>
                    </div>
                  </div>

                  {capabilities?.protocol === 'mlx' && capabilities.languages.length > 0 && (
                    <div>
                      <label className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1.5">{tr('配音语言', 'Speech Language')}</label>
                      <select value={capabilities.languages.includes(config.tts.language || 'Auto') ? config.tts.language || 'Auto' : ''}
                        onChange={event => setConfig({ ...config, tts: { ...config.tts, language: event.target.value } })}
                        className="w-full h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white">
                        <option value="" disabled>{tr('请选择语言', 'Select a language')}</option>
                        {capabilities.languages.map(language => <option key={language} value={language}>{language === 'Auto' ? tr('自动 · 随任务目标语言', 'Auto · Follow the task target language') : language}</option>)}
                      </select>
                    </div>
                  )}

                  <div className="space-y-1.5">
                    <label className="block text-xs font-medium text-slate-700 dark:text-slate-300">{tr('情绪与语气指令', 'Emotion and Style Instructions')}</label>
                    {capabilities?.instructions ? (
                      <textarea rows={2} maxLength={4096} value={config.tts.instructions || ''}
                        onChange={event => setConfig({ ...config, tts: { ...config.tts, instructions: event.target.value } })}
                        placeholder={tr('例如：用温暖、平静的语气朗读。', 'For example: Read in a warm, calm voice.')}
                        className="w-full px-3 py-2 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white" />
                    ) : (
                      <p className="text-[11px] text-slate-400">
                        {tr('当前模型未启用情绪指令；Qwen 0.6B 通常不支持，请选择支持指令的 1.7B 模型。', 'Instructions are unavailable; Qwen 0.6B generally does not support them. Select a compatible 1.7B model.')}
                        {config.tts.instructions && <button type="button" onClick={() => setConfig({ ...config, tts: { ...config.tts, instructions: '' } })}
                          className="ml-2 text-blue-600">{tr('清空原指令', 'Clear saved instructions')}</button>}
                      </p>
                    )}
                  </div>


                  <div className="flex flex-wrap items-center gap-3 pt-1">
                    {testResult && activeEngine === 'openai' && (
                      <span className={`text-xs ${testResult.success ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'}`}>
                        {testResult.message}
                      </span>
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
