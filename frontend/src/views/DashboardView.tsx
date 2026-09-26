import React, { useState } from 'react';
import {
  UploadCloud,
  FileVideo,
  X,
  Play,
  Settings2,
  CheckCircle2,
  Sparkles,
  Volume2,
  Languages,
  Loader2,
  AlertCircle,
} from 'lucide-react';
import { api } from '../lib/api';
import { TaskMode } from '../types';

interface DashboardViewProps {
  onTaskCreated: () => void;
}

export const DashboardView: React.FC<DashboardViewProps> = ({ onTaskCreated }) => {
  const [selectedFiles, setSelectedFiles] = useState<string[]>([]);
  const [mode, setMode] = useState<TaskMode>(3);
  const [sourceLang, setSourceLang] = useState('auto');
  const [targetLang, setTargetLang] = useState('zh-CN');
  
  // 翻译引擎：严格对齐后端合法枚举，自建服务走 openai 规范
  const [translator, setTranslator] = useState<'bing' | 'openai' | 'deeplx'>('openai');
  const [outputContent, setOutputContent] = useState('bilingual');
  
  // 配音与音色
  const [ttsEngine, setTtsEngine] = useState<'edge' | 'openai'>('edge');
  const [ttsVoice, setTtsVoice] = useState('zh-CN-XiaoxiaoNeural');
  const [speechRate, setSpeechRate] = useState(1.1);
  const [subtitleOutput, setSubtitleOutput] = useState('burn');

  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // 原生选择文件
  const handlePickFiles = async () => {
    try {
      const files = await api.pickFiles();
      if (files && files.length > 0) {
        setSelectedFiles((prev) => Array.from(new Set([...prev, ...files])));
      }
    } catch (err) {
      console.error('Pick files error:', err);
    }
  };

  const handleRemoveFile = (index: number) => {
    setSelectedFiles((prev) => prev.filter((_, i) => i !== index));
  };

  // 提交创建任务
  const handleStartTask = async () => {
    if (selectedFiles.length === 0) {
      await handlePickFiles();
      return;
    }

    setLoading(true);
    setErrorMessage(null);

    try {
      if (selectedFiles.length === 1) {
        await api.createTask({
          input_path: selectedFiles[0],
          mode,
          source_lang: sourceLang,
          target_lang: targetLang,
          translator, // 传递规范的 openai/bing/deeplx
          output_content: outputContent,
          tts_engine: ttsEngine,
          tts_voice: ttsVoice,
          speech_rate: speechRate,
          subtitle_output: subtitleOutput,
        });
      } else {
        await api.batchCreateTasks(selectedFiles, {
          input_path: '',
          mode,
          source_lang: sourceLang,
          target_lang: targetLang,
          translator,
          output_content: outputContent,
          tts_engine: ttsEngine,
          tts_voice: ttsVoice,
          speech_rate: speechRate,
          subtitle_output: subtitleOutput,
        });
      }
      setSelectedFiles([]);
      onTaskCreated();
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6">
      <div className="max-w-4xl w-full mx-auto space-y-6">
        {/* 顶部标题 */}
        <div>
          <h2 className="text-xl font-bold tracking-tight text-apple-text dark:text-apple-darkText">
            音视频工作台
          </h2>
          <p className="text-xs text-apple-muted mt-1">
            智能语音识别、多语言高质量翻译与全自动声音配音流水线
          </p>
        </div>

        {/* 错误提示条 */}
        {errorMessage && (
          <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-xl flex items-center justify-between text-xs text-red-500 animate-fadeIn">
            <div className="flex items-center space-x-2">
              <AlertCircle size={16} />
              <span>创建任务遇到问题：{errorMessage}</span>
            </div>
            <button onClick={() => setErrorMessage(null)} className="hover:opacity-75">
              <X size={14} />
            </button>
          </div>
        )}

        {/* 文件拖拽 / 拾取区 */}
        <div
          data-file-drop-target="true"
          onClick={handlePickFiles}
          className="border-2 border-dashed border-apple-border dark:border-apple-darkBorder hover:border-apple-accent dark:hover:border-apple-accent rounded-2xl p-6 flex flex-col items-center justify-center cursor-pointer transition-all bg-apple-card dark:bg-apple-darkCard hover:shadow-sm"
        >
          <div className="w-12 h-12 rounded-full bg-apple-accent/10 text-apple-accent flex items-center justify-center mb-3">
            <UploadCloud size={24} />
          </div>
          <p className="text-sm font-semibold text-apple-text dark:text-apple-darkText">
            点击选择或将音视频文件拖拽至此
          </p>
          <p className="text-xs text-apple-muted mt-1">
            支持 MP4, MKV, MOV, WebM, AVI 视频及 MP3, WAV 等音频
          </p>
        </div>

        {/* 已选文件列表卡片 */}
        {selectedFiles.length > 0 && (
          <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-4 shadow-sm space-y-2">
            <div className="flex items-center justify-between text-xs font-semibold text-apple-muted px-1">
              <span>待处理文件 ({selectedFiles.length})</span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setSelectedFiles([]);
                }}
                className="text-red-500 hover:underline"
              >
                清空
              </button>
            </div>
            <div className="max-h-36 overflow-y-auto space-y-1.5 pr-1">
              {selectedFiles.map((file, idx) => (
                <div
                  key={idx}
                  className="flex items-center justify-between bg-apple-bg dark:bg-apple-darkBg px-3 py-2 rounded-lg text-xs"
                >
                  <div className="flex items-center space-x-2 truncate pr-2">
                    <FileVideo size={15} className="text-apple-accent shrink-0" />
                    <span className="truncate font-mono text-apple-text dark:text-apple-darkText">
                      {file}
                    </span>
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleRemoveFile(idx);
                    }}
                    className="text-apple-muted hover:text-red-500 p-0.5"
                  >
                    <X size={13} />
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* 模式选择 */}
        <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-5 shadow-sm space-y-3">
          <label className="text-xs font-semibold text-apple-muted flex items-center space-x-1.5">
            <Sparkles size={14} className="text-apple-accent" />
            <span>处理流程模式</span>
          </label>
          <div className="grid grid-cols-3 gap-3">
            {[
              { id: 1 as TaskMode, label: '仅提取字幕', desc: 'Whisper 语音识别生成中英时间轴' },
              { id: 2 as TaskMode, label: '双语字幕翻译', desc: '智能识别并翻译为高质量对照字幕' },
              { id: 3 as TaskMode, label: '完整翻译与配音', desc: '识别 + 翻译 + 语音合成 + 成品压制' },
            ].map((item) => {
              const active = mode === item.id;
              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => setMode(item.id)}
                  className={`p-3 rounded-xl border text-left transition-all ${
                    active
                      ? 'border-apple-accent bg-apple-accent/5 ring-1 ring-apple-accent'
                      : 'border-apple-border dark:border-apple-darkBorder hover:bg-black/5 dark:hover:bg-white/5'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-apple-text dark:text-apple-darkText">
                      {item.label}
                    </span>
                    {active && <CheckCircle2 size={14} className="text-apple-accent" />}
                  </div>
                  <p className="text-[11px] text-apple-muted mt-1 leading-relaxed">
                    {item.desc}
                  </p>
                </button>
              );
            })}
          </div>
        </div>

        {/* 翻译与语言配置 */}
        <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-5 shadow-sm space-y-4">
          <label className="text-xs font-semibold text-apple-muted flex items-center space-x-1.5">
            <Languages size={14} className="text-apple-accent" />
            <span>语言与翻译模型</span>
          </label>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <span className="text-[11px] text-apple-muted block mb-1.5">原语言</span>
              <select
                value={sourceLang}
                onChange={(e) => setSourceLang(e.target.value)}
                className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-2.5 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
              >
                <option value="auto">自动识别 (Auto)</option>
                <option value="en">英语 (English)</option>
                <option value="zh">中文 (Chinese)</option>
                <option value="ja">日语 (Japanese)</option>
                <option value="ko">韩语 (Korean)</option>
                <option value="de">德语 (German)</option>
                <option value="fr">法语 (French)</option>
              </select>
            </div>

            <div>
              <span className="text-[11px] text-apple-muted block mb-1.5">翻译目标语言</span>
              <select
                value={targetLang}
                onChange={(e) => setTargetLang(e.target.value)}
                className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-2.5 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
              >
                <option value="zh-CN">简体中文 (zh-CN)</option>
                <option value="en">英语 (en)</option>
                <option value="ja">日语 (ja)</option>
                <option value="ko">韩语 (ko)</option>
                <option value="de">德语 (de)</option>
                <option value="fr">法语 (fr)</option>
              </select>
            </div>

            <div>
              <span className="text-[11px] text-apple-muted block mb-1.5">翻译服务引擎</span>
              <select
                value={translator}
                onChange={(e) => setTranslator(e.target.value as 'bing' | 'openai' | 'deeplx')}
                className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-2.5 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
              >
                <option value="openai">自建 / OpenAI 兼容接口</option>
                <option value="bing">必应免费翻译 (免 Key)</option>
                <option value="deeplx">DeepLX 自建接口</option>
              </select>
            </div>
          </div>
        </div>

        {/* 配音与成品设置（仅在 mode >= 2 或 mode === 3 时展示） */}
        {mode >= 2 && (
          <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-5 shadow-sm space-y-4">
            <label className="text-xs font-semibold text-apple-muted flex items-center space-x-1.5">
              <Volume2 size={14} className="text-apple-accent" />
              <span>配音与合成配置</span>
            </label>

            <div className="grid grid-cols-3 gap-4">
              <div>
                <span className="text-[11px] text-apple-muted block mb-1.5">配音引擎</span>
                <select
                  value={ttsEngine}
                  onChange={(e) => setTtsEngine(e.target.value as 'edge' | 'openai')}
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-2.5 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                >
                  <option value="edge">Edge-TTS (免费高质量)</option>
                  <option value="openai">OpenAI TTS</option>
                </select>
              </div>

              <div>
                <span className="text-[11px] text-apple-muted block mb-1.5">配音音色</span>
                <select
                  value={ttsVoice}
                  onChange={(e) => setTtsVoice(e.target.value)}
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-2.5 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                >
                  {ttsEngine === 'edge' ? (
                    <>
                      <option value="zh-CN-XiaoxiaoNeural">晓晓 (女声·温柔亲切)</option>
                      <option value="zh-CN-YunxiNeural">云希 (男声·阳光解说)</option>
                      <option value="zh-CN-YunjianNeural">云健 (男声·沉稳影视)</option>
                      <option value="zh-CN-XiaoyiNeural">晓伊 (女声·知性清晰)</option>
                    </>
                  ) : (
                    <>
                      <option value="alloy">Alloy (平衡通用)</option>
                      <option value="echo">Echo (温暖自然)</option>
                      <option value="nova">Nova (活泼热情)</option>
                      <option value="onyx">Onyx (深沉磁性)</option>
                    </>
                  )}
                </select>
              </div>

              <div>
                <div className="flex justify-between items-center mb-1.5">
                  <span className="text-[11px] text-apple-muted">语速倍率</span>
                  <span className="text-[11px] font-mono text-apple-accent font-bold">
                    {speechRate}x
                  </span>
                </div>
                <input
                  type="range"
                  min="0.8"
                  max="1.5"
                  step="0.05"
                  value={speechRate}
                  onChange={(e) => setSpeechRate(parseFloat(e.target.value))}
                  className="w-full accent-apple-accent"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4 pt-2 border-t border-apple-border dark:border-apple-darkBorder">
              <div>
                <span className="text-[11px] text-apple-muted block mb-1.5">字幕输出方式</span>
                <select
                  value={subtitleOutput}
                  onChange={(e) => setSubtitleOutput(e.target.value)}
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-2.5 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                >
                  <option value="burn">烧录硬字幕到成片 (Burn)</option>
                  <option value="file">导出独立 SRT 字幕文件</option>
                  <option value="none">不输出字幕</option>
                </select>
              </div>

              <div>
                <span className="text-[11px] text-apple-muted block mb-1.5">字幕排版内容</span>
                <select
                  value={outputContent}
                  onChange={(e) => setOutputContent(e.target.value)}
                  className="w-full bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-2.5 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
                >
                  <option value="bilingual">双语对照字幕</option>
                  <option value="translated">仅译文字幕</option>
                  <option value="source">仅原文字幕</option>
                </select>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* 底部固定操作栏 */}
      <div className="max-w-4xl w-full mx-auto pt-4 border-t border-apple-border dark:border-apple-darkBorder flex items-center justify-between">
        <div className="flex items-center space-x-2 text-xs text-apple-muted">
          <Settings2 size={15} />
          <span>已就绪 · 单击开始即可全自动执行流水线</span>
        </div>

        <button
          onClick={handleStartTask}
          disabled={loading}
          className="flex items-center space-x-2 bg-apple-accent hover:bg-apple-accentHover disabled:opacity-50 text-white px-6 py-2 rounded-xl text-xs font-semibold shadow-sm transition-all"
        >
          {loading ? (
            <>
              <Loader2 size={15} className="animate-spin" />
              <span>正在创建任务...</span>
            </>
          ) : (
            <>
              <Play size={15} />
              <span>{selectedFiles.length > 0 ? `开始处理 (${selectedFiles.length})` : '选择文件并处理'}</span>
            </>
          )}
        </button>
      </div>
    </div>
  );
};
