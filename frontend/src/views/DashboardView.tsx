import React, { useState, useEffect } from 'react';
import {
  UploadCloud,
  FileVideo,
  FileText,
  X,
  Play,
  CheckCircle2,
  Volume2,
  Languages,
  Film,
  Mic,
  ChevronRight,
  ChevronDown,
  Info,
  UserCheck,
  BookmarkPlus,
  Loader2,
  AlertCircle,
  Clock,
} from 'lucide-react';
import { api } from '../lib/api';
import { TaskMode } from '../types';

declare global {
  interface Window {
    __onWailsFilesDropped?: (files: string[]) => void;
  }
}

interface DashboardViewProps {
  onTaskCreated: () => void;
}

// 工作流快捷预设模板
interface WorkflowPreset {
  id: string;
  title: string;
  subtitle: string;
  badge: string;
  goals: {
    sub: boolean;
    translate: boolean;
    dub: boolean;
    video: boolean;
  };
}

const PRESETS: WorkflowPreset[] = [
  {
    id: 'dub_full',
    title: '视频 → 配音成片',
    subtitle: '听写、翻译、配音、合成一条龙，全自动出成品视频',
    badge: '全自动',
    goals: { sub: true, translate: true, dub: true, video: true },
  },
  {
    id: 'bilingual_sub',
    title: '视频 → 双语字幕',
    subtitle: '转写人声并翻译成目标语言，一步到位产出高精字幕',
    badge: '精细校对',
    goals: { sub: true, translate: true, dub: false, video: false },
  },
  {
    id: 'subtitle_only',
    title: '视频 → 原文字幕',
    subtitle: '只转写视频人声，不进行翻译，快速生成高精度 SRT / VTT',
    badge: '极速转写',
    goals: { sub: true, translate: false, dub: false, video: false },
  },
  {
    id: 'translate_existing',
    title: '翻译已有字幕',
    subtitle: '导入已有 SRT/ASS 字幕文件，结合上下文意译至指定语言',
    badge: '字幕翻译',
    goals: { sub: true, translate: true, dub: false, video: false },
  },
  {
    id: 'custom',
    title: '自定义智能流程',
    subtitle: '自主灵活开启或关闭各个处理流水线阶段',
    badge: '自由组合',
    goals: { sub: true, translate: true, dub: true, video: true },
  },
];

export const DashboardView: React.FC<DashboardViewProps> = ({ onTaskCreated }) => {
  // 选中的快捷预设
  const [activePreset, setActivePreset] = useState<string>('dub_full');

  // 第一步：放入的文件列表
  const [selectedFiles, setSelectedFiles] = useState<string[]>([]);
  const [isDragging, setIsDragging] = useState(false);

  // 第二步：我要得到 (四阶段目标核心开关)
  const [doSub, setDoSub] = useState(true);
  const [doTranslate, setDoTranslate] = useState(true);
  const [doDub, setDoDub] = useState(true);
  const [doVideo, setDoVideo] = useState(true);

  // 子阶段 1：字幕与翻译配置
  const [whisperModel, setWhisperModel] = useState('Whisper.cpp large-v3-turbo');
  const [videoLang, setVideoLang] = useState('auto');
  const [targetLang, setTargetLang] = useState('zh-CN');
  const [translateService, setTranslateService] = useState<'openai' | 'local' | 'bing' | 'deeplx'>('openai');
  const [outputContent, setOutputContent] = useState('bilingual');

  // 子阶段 2：AI 配音配置
  const [ttsEngine, setTtsEngine] = useState<'edge' | 'local' | 'openai' | 'elevenlabs'>('edge');
  const [ttsVoice, setTtsVoice] = useState('zh-CN-XiaoxiaoNeural');
  const [speechRate, setSpeechRate] = useState<number>(1.1);

  // 子阶段 3：成品视频压制配置
  const [subtitleOutput, setSubtitleOutput] = useState('burn');
  const [subtitleStyle, setSubtitleStyle] = useState('经典白字黑边');
  const [videoQuality, setVideoQuality] = useState('原画质');
  const [encodeMethod, setEncodeMethod] = useState('Apple VideoToolbox');

  // 子阶段 4：人工把关开关
  const [proofread, setProofread] = useState(true);
  const [ttsConfirm, setTtsConfirm] = useState(false);

  // 配方弹窗与提交状态
  const [recipeName, setRecipeName] = useState('');
  const [showRecipeModal, setShowRecipeModal] = useState(false);
  const [recipeSavedToast, setRecipeSavedToast] = useState(false);
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // 路径直接输入态
  const [manualPath, setManualPath] = useState('');
  const [showManualInput, setShowManualInput] = useState(false);

  // 统一文件添加助手
  const appendFiles = (incoming: unknown) => {
    let paths: string[] = [];
    if (incoming instanceof FileList) {
      for (let i = 0; i < incoming.length; i++) {
        const f = incoming[i];
        const p = (f as unknown as { path?: string }).path || f.name;
        if (p) paths.push(p);
      }
    } else if (Array.isArray(incoming)) {
      for (const item of incoming) {
        if (typeof item === 'string') {
          paths.push(item);
        } else if (item && typeof item === 'object') {
          const p = (item as unknown as { path?: string }).path || (item as File).name;
          if (p) paths.push(p);
        }
      }
    } else if (incoming && typeof incoming === 'object') {
      const obj = incoming as Record<string, unknown>;
      if (Array.isArray(obj.filenames)) {
        paths = obj.filenames as string[];
      } else if (Array.isArray(obj.data)) {
        paths = obj.data as string[];
      }
    }
    if (paths.length > 0) {
      setSelectedFiles((prev) => Array.from(new Set([...prev, ...paths])));
    }
  };

  // 全局与原生窗口拖拽双轨监听
  useEffect(() => {
    const handleWailsDrop = (data: unknown) => appendFiles(data);
    window.__onWailsFilesDropped = (files: string[]) => appendFiles(files);
    const unbind1 = api.onEvent('files-dropped', handleWailsDrop);
    const unbind2 = api.onEvent('common:WindowFilesDropped', handleWailsDrop);

    const handleWindowDragOver = (e: DragEvent) => {
      e.preventDefault();
      setIsDragging(true);
    };

    const handleWindowDragLeave = (e: DragEvent) => {
      if (!e.relatedTarget) {
        setIsDragging(false);
      }
    };

    const handleWindowDrop = (e: DragEvent) => {
      e.preventDefault();
      setIsDragging(false);
      if (e.dataTransfer && e.dataTransfer.files.length > 0) {
        appendFiles(e.dataTransfer.files);
      }
    };

    window.addEventListener('dragover', handleWindowDragOver);
    window.addEventListener('dragleave', handleWindowDragLeave);
    window.addEventListener('drop', handleWindowDrop);

    return () => {
      delete window.__onWailsFilesDropped;
      unbind1();
      unbind2();
      window.removeEventListener('dragover', handleWindowDragOver);
      window.removeEventListener('dragleave', handleWindowDragLeave);
      window.removeEventListener('drop', handleWindowDrop);
    };
  }, []);

  // 切换预设模板
  const handleSelectPreset = (preset: WorkflowPreset) => {
    setActivePreset(preset.id);
    setDoSub(preset.goals.sub);
    setDoTranslate(preset.goals.translate);
    setDoDub(preset.goals.dub);
    setDoVideo(preset.goals.video);
  };

  // 目标开关联动逻辑
  const handleToggleSub = () => {
    setDoSub((prev) => !prev);
    setActivePreset('custom');
  };

  const handleToggleTranslate = () => {
    setDoTranslate((prev) => {
      const next = !prev;
      if (!next) {
        setDoDub(false);
        setDoVideo(false);
      }
      return next;
    });
    setActivePreset('custom');
  };

  const handleToggleDub = () => {
    setDoDub((prev) => {
      const next = !prev;
      if (next && !doTranslate) {
        setDoTranslate(true);
      }
      if (!next) {
        setDoVideo(false);
      }
      return next;
    });
    setActivePreset('custom');
  };

  const handleToggleVideo = () => {
    setDoVideo((prev) => {
      const next = !prev;
      if (next) {
        if (!doDub) setDoDub(true);
        if (!doTranslate) setDoTranslate(true);
      }
      return next;
    });
    setActivePreset('custom');
  };

  // 原生选择文件
  const handlePickFiles = async () => {
    try {
      const files = await api.pickFiles();
      if (files && files.length > 0) {
        appendFiles(files);
      }
    } catch (err) {
      console.error('Pick files error:', err);
    }
  };

  const handleRemoveFile = (index: number) => {
    setSelectedFiles((prev) => prev.filter((_, i) => i !== index));
  };

  const resolvedMode = (): TaskMode => {
    if (doVideo && doDub) return 3;
    if (doTranslate) return 2;
    return 1;
  };

  // 提交并创建任务
  const handleStartTask = async () => {
    if (selectedFiles.length === 0) {
      await handlePickFiles();
      return;
    }

    setLoading(true);
    setErrorMessage(null);

    const mode = resolvedMode();
    const finalSubtitleOutput = doVideo ? subtitleOutput : 'file';
    const finalTranslator = translateService === 'local' ? 'openai' : translateService;

    try {
      if (selectedFiles.length === 1) {
        await api.createTask({
          input_path: selectedFiles[0],
          mode,
          source_lang: videoLang,
          target_lang: targetLang,
          translator: finalTranslator,
          output_content: outputContent,
          tts_engine: ttsEngine,
          tts_voice: ttsVoice,
          speech_rate: speechRate,
          subtitle_output: finalSubtitleOutput,
        });
      } else {
        await api.batchCreateTasks(selectedFiles, {
          input_path: '',
          mode,
          source_lang: videoLang,
          target_lang: targetLang,
          translator: finalTranslator,
          output_content: outputContent,
          tts_engine: ttsEngine,
          tts_voice: ttsVoice,
          speech_rate: speechRate,
          subtitle_output: finalSubtitleOutput,
        });
      }

      onTaskCreated();
    } catch (err: unknown) {
      console.error('Failed to create task:', err);
      setErrorMessage((err as Error).message || '创建任务失败，请检查音视频格式及后端配置');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveRecipe = () => {
    if (!recipeName.trim()) return;
    setShowRecipeModal(false);
    setRecipeSavedToast(true);
    setTimeout(() => setRecipeSavedToast(false), 2500);
  };

  const hasFiles = selectedFiles.length > 0;

  return (
    <div className="flex flex-col h-full w-full overflow-hidden bg-slate-50/60 dark:bg-[#121215]" data-file-drop-target="true">
      {/* 上半部：主内容滚动区域（底部不再受任何悬浮条遮挡） */}
      <div className="flex-1 overflow-y-auto px-7 py-6 space-y-6">
        {/* 顶部标题与快捷模板 */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <div>
              <h1 className="text-xl font-bold text-slate-900 dark:text-white tracking-tight flex items-center gap-2">
                <span>开始创作</span>
                <span className="text-[11px] font-semibold px-2 py-0.5 rounded-full bg-blue-50 text-blue-600 border border-blue-200 dark:bg-blue-500/15 dark:text-blue-300 dark:border-blue-500/25">
                  AI 智能流水线
                </span>
              </h1>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">选择推荐配方或自由组合阶段，将原声视频全自动转化为多语种成片</p>
            </div>
          </div>

          {/* 快捷模板网格：统摄为克制清朗的 Apple 风格 */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
            {PRESETS.map((preset) => {
              const isSelected = activePreset === preset.id;
              return (
                <button
                  key={preset.id}
                  type="button"
                  onClick={() => handleSelectPreset(preset)}
                  className={`text-left rounded-2xl p-4 border transition-all relative overflow-hidden group ${
                    isSelected
                      ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600'
                      : 'bg-white hover:bg-slate-50 dark:bg-[#1C1C1E] dark:hover:bg-[#252528] border-slate-200 dark:border-[#2C2C2E]'
                  }`}
                >
                  <div className="flex items-center justify-between mb-2">
                    <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${
                      isSelected
                        ? 'bg-blue-600 text-white border-blue-600'
                        : 'bg-slate-100 text-slate-600 border-slate-200 dark:bg-white/10 dark:text-slate-300 dark:border-white/10'
                    }`}>
                      {preset.badge}
                    </span>
                    {isSelected && <CheckCircle2 className="w-4 h-4 text-blue-600 dark:text-blue-400" />}
                  </div>
                  <h3 className={`text-sm font-bold transition-colors ${
                    isSelected ? 'text-blue-900 dark:text-white' : 'text-slate-800 dark:text-slate-100 group-hover:text-blue-600 dark:group-hover:text-blue-400'
                  }`}>
                    {preset.title}
                  </h3>
                  <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1 line-clamp-2 leading-relaxed">
                    {preset.subtitle}
                  </p>
                </button>
              );
            })}
          </div>
        </div>

        {/* 第一步：放入文件 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2.5">
              <span className="w-6 h-6 rounded-full bg-blue-600 text-white text-xs font-bold flex items-center justify-center shadow-md shadow-blue-500/20">
                1
              </span>
              <div>
                <h2 className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">第一步 · 放入待处理文件</h2>
                <p className="text-[11px] text-slate-500 dark:text-slate-400">支持音视频及字幕文件，放入多个即可启动批量流水线</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setShowManualInput((prev) => !prev)}
                className="text-xs text-slate-500 hover:text-blue-600 dark:hover:text-blue-400 px-2.5 py-1 rounded-lg hover:bg-blue-50 dark:hover:bg-blue-500/10 transition-colors"
              >
                {showManualInput ? '隐藏输入框' : '手动输入/粘贴路径'}
              </button>
              {hasFiles && (
                <button
                  type="button"
                  onClick={() => setSelectedFiles([])}
                  className="text-xs text-slate-500 hover:text-rose-600 dark:hover:text-rose-400 px-2.5 py-1 rounded-lg hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-colors"
                >
                  清空列表
                </button>
              )}
              <button
                type="button"
                onClick={handlePickFiles}
                className="text-xs font-semibold px-3 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-white border border-slate-200 dark:border-white/10 transition-all flex items-center gap-1.5 shadow-sm"
              >
                <UploadCloud className="w-3.5 h-3.5" />
                <span>选择文件</span>
              </button>
            </div>
          </div>

          {/* 手动路径输入折叠区 */}
          {showManualInput && (
            <div className="mb-4 p-3 bg-slate-50 dark:bg-white/[0.03] border border-slate-200 dark:border-white/10 rounded-xl space-y-2 animate-in fade-in">
              <div className="flex gap-2">
                <input
                  type="text"
                  placeholder="粘贴音视频绝对路径，例如：/path/to/demo.mp4"
                  value={manualPath}
                  onChange={(e) => setManualPath(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && manualPath.trim()) {
                      appendFiles([manualPath.trim()]);
                      setManualPath('');
                    }
                  }}
                  className="flex-1 h-9 bg-white dark:bg-slate-900 border border-slate-200 dark:border-white/15 rounded-lg px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/50"
                />
                <button
                  type="button"
                  onClick={() => {
                    if (manualPath.trim()) {
                      appendFiles([manualPath.trim()]);
                      setManualPath('');
                    }
                  }}
                  className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold rounded-lg transition-colors shadow-sm"
                >
                  添加
                </button>
              </div>
            </div>
          )}

          {/* 已放入的文件列表标签 */}
          {hasFiles && (
            <div className="mb-4 max-h-48 overflow-y-auto space-y-2 pr-1">
              {selectedFiles.map((file, idx) => {
                const fileName = file.split('/').pop() || file;
                const isSub = fileName.endsWith('.srt') || fileName.endsWith('.ass') || fileName.endsWith('.vtt');
                return (
                  <div
                    key={idx}
                    className="flex items-center justify-between bg-slate-50 dark:bg-white/[0.04] border border-slate-200 dark:border-white/10 rounded-xl px-3.5 py-2.5 group hover:border-blue-400/50 transition-all"
                  >
                    <div className="flex items-center gap-2.5 min-w-0 pr-2">
                      <div className="w-7 h-7 rounded-lg bg-blue-50 dark:bg-blue-500/20 text-blue-600 dark:text-blue-400 flex items-center justify-center shrink-0">
                        {isSub ? <FileText className="w-4 h-4" /> : <FileVideo className="w-4 h-4" />}
                      </div>
                      <div className="truncate">
                        <p className="text-xs font-semibold text-slate-800 dark:text-slate-200 truncate">{fileName}</p>
                        <p className="text-[10px] text-slate-400 truncate">{file}</p>
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleRemoveFile(idx)}
                      className="w-6 h-6 rounded-lg text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 flex items-center justify-center transition-colors shrink-0"
                      title="移除此文件"
                    >
                      <X className="w-3.5 h-3.5" />
                    </button>
                  </div>
                );
              })}
            </div>
          )}

          {/* 拖拽放置虚线区域 */}
          <div
            data-file-drop-target="true"
            onDragOver={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setIsDragging(true);
            }}
            onDragLeave={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setIsDragging(false);
            }}
            onDrop={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setIsDragging(false);
              if (e.dataTransfer && e.dataTransfer.files.length > 0) {
                appendFiles(e.dataTransfer.files);
              }
            }}
            onClick={handlePickFiles}
            className={`border-2 border-dashed rounded-xl py-8 px-4 text-center cursor-pointer transition-all ${
              isDragging
                ? 'border-blue-500 bg-blue-50/80 dark:bg-blue-500/10 scale-[0.99]'
                : 'border-slate-300 dark:border-white/15 hover:border-blue-500/70 bg-slate-50/50 hover:bg-slate-50 dark:bg-white/[0.02] dark:hover:bg-white/[0.04]'
            }`}
          >
            <UploadCloud className="w-8 h-8 text-blue-600 dark:text-blue-400 mx-auto mb-2 opacity-85" />
            <p className="text-xs font-semibold text-slate-800 dark:text-slate-200">
              把视频或字幕文件拖到这里，也可以点击浏览选择
            </p>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1">
              支持 MP4, MKV, MOV, MP3, WAV, AAC, SRT, ASS · 同时放入视频与字幕将自动同名配对跳过听写
            </p>
          </div>
        </div>

        {/* 第二步：我要得到 (统摄 Apple 极简蓝风格) */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <span className="w-6 h-6 rounded-full bg-blue-600 text-white text-xs font-bold flex items-center justify-center shadow-md shadow-blue-500/20">
                2
              </span>
              <div>
                <h2 className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">第二步 · 我要得到</h2>
                <p className="text-[11px] text-slate-500 dark:text-slate-400">勾选所需产物阶段，下方将动态呈现对应的子配置参数</p>
              </div>
            </div>
          </div>

          {/* 四阶段目标卡片横排：告别杂色，统一高级 Apple 风格 */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {/* 目标 1: 字幕文件 */}
            <div
              onClick={handleToggleSub}
              className={`p-3.5 rounded-xl border cursor-pointer transition-all relative flex flex-col justify-between h-24 ${
                doSub
                  ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600'
                  : 'bg-slate-50/50 hover:bg-slate-100/80 dark:bg-white/[0.02] border-slate-200/80 dark:border-white/10 opacity-75 hover:opacity-100'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className={`w-7 h-7 rounded-lg flex items-center justify-center ${
                  doSub ? 'bg-blue-600 text-white' : 'bg-slate-200 dark:bg-white/10 text-slate-500 dark:text-slate-400'
                }`}>
                  <FileText className="w-4 h-4" />
                </div>
                <div className={`w-4 h-4 rounded-full flex items-center justify-center transition-all ${
                  doSub ? 'bg-blue-600 text-white' : 'border border-slate-300 dark:border-slate-600 bg-transparent'
                }`}>
                  {doSub && <CheckCircle2 className="w-3.5 h-3.5 stroke-[2.5]" />}
                </div>
              </div>
              <div>
                <div className={`text-xs font-bold ${doSub ? 'text-blue-950 dark:text-white' : 'text-slate-700 dark:text-slate-300'}`}>
                  字幕文件
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">视频任务必产出字幕</div>
              </div>
            </div>

            {/* 目标 2: 翻译字幕 */}
            <div
              onClick={handleToggleTranslate}
              className={`p-3.5 rounded-xl border cursor-pointer transition-all relative flex flex-col justify-between h-24 ${
                doTranslate
                  ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600'
                  : 'bg-slate-50/50 hover:bg-slate-100/80 dark:bg-white/[0.02] border-slate-200/80 dark:border-white/10 opacity-75 hover:opacity-100'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className={`w-7 h-7 rounded-lg flex items-center justify-center ${
                  doTranslate ? 'bg-blue-600 text-white' : 'bg-slate-200 dark:bg-white/10 text-slate-500 dark:text-slate-400'
                }`}>
                  <Languages className="w-4 h-4" />
                </div>
                <div className={`w-4 h-4 rounded-full flex items-center justify-center transition-all ${
                  doTranslate ? 'bg-blue-600 text-white' : 'border border-slate-300 dark:border-slate-600 bg-transparent'
                }`}>
                  {doTranslate && <CheckCircle2 className="w-3.5 h-3.5 stroke-[2.5]" />}
                </div>
              </div>
              <div>
                <div className={`text-xs font-bold ${doTranslate ? 'text-blue-950 dark:text-white' : 'text-slate-700 dark:text-slate-300'}`}>
                  翻译字幕
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">把字幕翻译成目标语言</div>
              </div>
            </div>

            {/* 目标 3: AI 配音 */}
            <div
              onClick={handleToggleDub}
              className={`p-3.5 rounded-xl border cursor-pointer transition-all relative flex flex-col justify-between h-24 ${
                doDub
                  ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600'
                  : 'bg-slate-50/50 hover:bg-slate-100/80 dark:bg-white/[0.02] border-slate-200/80 dark:border-white/10 opacity-75 hover:opacity-100'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className={`w-7 h-7 rounded-lg flex items-center justify-center ${
                  doDub ? 'bg-blue-600 text-white' : 'bg-slate-200 dark:bg-white/10 text-slate-500 dark:text-slate-400'
                }`}>
                  <Mic className="w-4 h-4" />
                </div>
                <div className={`w-4 h-4 rounded-full flex items-center justify-center transition-all ${
                  doDub ? 'bg-blue-600 text-white' : 'border border-slate-300 dark:border-slate-600 bg-transparent'
                }`}>
                  {doDub && <CheckCircle2 className="w-3.5 h-3.5 stroke-[2.5]" />}
                </div>
              </div>
              <div>
                <div className={`text-xs font-bold ${doDub ? 'text-blue-950 dark:text-white' : 'text-slate-700 dark:text-slate-300'}`}>
                  AI 配音
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">用 AI 声音朗读字幕</div>
              </div>
            </div>

            {/* 目标 4: 成品视频 */}
            <div
              onClick={handleToggleVideo}
              className={`p-3.5 rounded-xl border cursor-pointer transition-all relative flex flex-col justify-between h-24 ${
                doVideo
                  ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600'
                  : 'bg-slate-50/50 hover:bg-slate-100/80 dark:bg-white/[0.02] border-slate-200/80 dark:border-white/10 opacity-75 hover:opacity-100'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className={`w-7 h-7 rounded-lg flex items-center justify-center ${
                  doVideo ? 'bg-blue-600 text-white' : 'bg-slate-200 dark:bg-white/10 text-slate-500 dark:text-slate-400'
                }`}>
                  <Film className="w-4 h-4" />
                </div>
                <div className={`w-4 h-4 rounded-full flex items-center justify-center transition-all ${
                  doVideo ? 'bg-blue-600 text-white' : 'border border-slate-300 dark:border-slate-600 bg-transparent'
                }`}>
                  {doVideo && <CheckCircle2 className="w-3.5 h-3.5 stroke-[2.5]" />}
                </div>
              </div>
              <div>
                <div className={`text-xs font-bold ${doVideo ? 'text-blue-950 dark:text-white' : 'text-slate-700 dark:text-slate-300'}`}>
                  成品视频
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">字幕(和配音)合成进视频</div>
              </div>
            </div>
          </div>

          {/* 动态处理流程拓扑条：统一为克制经典的蓝灰格调 */}
          <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.02] border border-slate-200/80 dark:border-white/5 flex flex-wrap items-center gap-2 text-xs">
            <span className="text-slate-500 dark:text-slate-400 font-medium">当前处理流程:</span>
            <div className="flex flex-wrap items-center gap-1.5">
              <span className={`px-2.5 py-1 rounded-full text-[11px] font-mono flex items-center gap-1 border ${
                doSub
                  ? 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/40'
                  : 'bg-slate-100 text-slate-400 border-slate-200 dark:bg-white/5 dark:text-slate-500 dark:border-white/5'
              }`}>
                <Mic className="w-3 h-3" /> 听写转录
              </span>

              <ChevronRight className={`w-3.5 h-3.5 ${doTranslate ? 'text-blue-500' : 'text-slate-300 dark:text-slate-700'}`} />

              <span className={`px-2.5 py-1 rounded-full text-[11px] font-mono flex items-center gap-1 border ${
                doTranslate
                  ? 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/40'
                  : 'bg-slate-100 text-slate-400 border-slate-200 dark:bg-white/5 dark:text-slate-500 dark:border-white/5'
              }`}>
                <Languages className="w-3 h-3" /> 智能翻译
              </span>

              <ChevronRight className={`w-3.5 h-3.5 ${doDub ? 'text-blue-500' : 'text-slate-300 dark:text-slate-700'}`} />

              <span className={`px-2.5 py-1 rounded-full text-[11px] font-mono flex items-center gap-1 border ${
                doDub
                  ? 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/40'
                  : 'bg-slate-100 text-slate-400 border-slate-200 dark:bg-white/5 dark:text-slate-500 dark:border-white/5'
              }`}>
                <Volume2 className="w-3 h-3" /> 语音合成
              </span>

              <ChevronRight className={`w-3.5 h-3.5 ${doVideo ? 'text-blue-500' : 'text-slate-300 dark:text-slate-700'}`} />

              <span className={`px-2.5 py-1 rounded-full text-[11px] font-mono flex items-center gap-1 border ${
                doVideo
                  ? 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/40'
                  : 'bg-slate-100 text-slate-400 border-slate-200 dark:bg-white/5 dark:text-slate-500 dark:border-white/5'
              }`}>
                <Film className="w-3 h-3" /> 压制合成
              </span>
            </div>
          </div>
        </div>

        {/* 阶段 1：字幕与翻译设置 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 shadow-sm space-y-4">
          <div className="flex items-center gap-2">
            <Languages className="w-4 h-4 text-blue-600 dark:text-blue-400" />
            <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
              字幕与翻译设置
            </h3>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            {/* 语音模型 */}
            <div>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">语音模型</label>
              <div className="relative">
                <select
                  value={whisperModel}
                  onChange={(e) => setWhisperModel(e.target.value)}
                  className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                >
                  <option value="Whisper.cpp large-v3-turbo">Whisper.cpp large-v3-turbo (推荐)</option>
                  <option value="Whisper large-v3">Whisper large-v3 (标准质量)</option>
                  <option value="Whisper medium">Whisper medium (平衡)</option>
                  <option value="SenseVoice Small">SenseVoice Small (超快)</option>
                </select>
                <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
              </div>
            </div>

            {/* 视频源语言 */}
            <div>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">视频源语言</label>
              <div className="relative">
                <select
                  value={videoLang}
                  onChange={(e) => setVideoLang(e.target.value)}
                  className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                >
                  <option value="auto">自动检测 (Auto Detect)</option>
                  <option value="en">英语 (English)</option>
                  <option value="zh">中文 (Chinese)</option>
                  <option value="ja">日语 (Japanese)</option>
                  <option value="ko">韩语 (Korean)</option>
                  <option value="de">德语 (German)</option>
                  <option value="fr">法语 (French)</option>
                </select>
                <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
              </div>
            </div>

            {/* 翻译目标语言 */}
            <div className={doTranslate ? 'opacity-100' : 'opacity-40 pointer-events-none'}>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                翻译目标语言 {!doTranslate && '(未启用翻译)'}
              </label>
              <div className="relative">
                <select
                  value={targetLang}
                  onChange={(e) => setTargetLang(e.target.value)}
                  disabled={!doTranslate}
                  className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                >
                  <option value="zh-CN">中文 (简体)</option>
                  <option value="zh-TW">中文 (繁体)</option>
                  <option value="en">英语 (English)</option>
                  <option value="ja">日语 (Japanese)</option>
                  <option value="ko">韩语 (Korean)</option>
                  <option value="de">德语 (German)</option>
                  <option value="fr">法语 (French)</option>
                </select>
                <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
              </div>
            </div>

            {/* 翻译服务商 */}
            <div className={doTranslate ? 'opacity-100' : 'opacity-40 pointer-events-none'}>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                翻译服务商 {!doTranslate && '(未启用翻译)'}
              </label>
              <div className="relative">
                <select
                  value={translateService}
                  onChange={(e) => setTranslateService(e.target.value as 'openai' | 'local' | 'bing' | 'deeplx')}
                  disabled={!doTranslate}
                  className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                >
                  <option value="openai">OpenAI / GPT-4o (推荐)</option>
                  <option value="local">Local LLM (本地 Ollama/Qwen)</option>
                  <option value="bing">必应翻译 (免费免配置)</option>
                  <option value="deeplx">DeepLX 翻译引擎</option>
                </select>
                <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
              </div>
            </div>

            {/* 输出内容模式 */}
            {doTranslate && (
              <div className="sm:col-span-2 lg:col-span-4 pt-1">
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">字幕输出内容模式</label>
                <div className="flex flex-wrap gap-2">
                  {[
                    { id: 'bilingual', label: '双语对照 (上译下原 推荐)' },
                    { id: 'target', label: '仅输出翻译字幕' },
                    { id: 'source', label: '仅输出原文字幕' },
                  ].map((item) => (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => setOutputContent(item.id)}
                      className={`px-3.5 py-1.5 rounded-xl text-xs font-semibold border transition-all ${
                        outputContent === item.id
                          ? 'bg-blue-600 text-white border-blue-600 shadow-sm'
                          : 'bg-slate-100 hover:bg-slate-200/80 dark:bg-white/[0.04] text-slate-600 dark:text-slate-300 border-slate-200 dark:border-white/10'
                      }`}
                    >
                      {item.label}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 阶段 2：AI 配音设置 */}
        {doDub && (
          <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 shadow-sm space-y-4 animate-in fade-in duration-200">
            <div className="flex items-center gap-2">
              <Volume2 className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                AI 语音合成与配音设置
              </h3>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              {/* 配音引擎 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">配音引擎</label>
                <div className="relative">
                  <select
                    value={ttsEngine}
                    onChange={(e) => setTtsEngine(e.target.value as 'edge' | 'local' | 'openai' | 'elevenlabs')}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                  >
                    <option value="edge">EdgeTTS (免费免配置 · 推荐)</option>
                    <option value="local">Local (ChatTTS / CosyVoice)</option>
                    <option value="openai">OpenAI TTS-1-HD</option>
                    <option value="elevenlabs">ElevenLabs (专业级)</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 音色选择 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">声音音色</label>
                <div className="relative">
                  <select
                    value={ttsVoice}
                    onChange={(e) => setTtsVoice(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                  >
                    <option value="zh-CN-YunxiNeural">云希 (经典纪录片/解说男声)</option>
                    <option value="zh-CN-XiaoxiaoNeural">晓晓 (自然温柔女声)</option>
                    <option value="zh-CN-YunjianNeural">云健 (沉稳专业男声)</option>
                    <option value="en-US-JennyNeural">Jenny (标准美语女声)</option>
                    <option value="1234">自定义音色克隆: 1234</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 语速倍率 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">语速倍率</label>
                <div className="relative">
                  <select
                    value={`${speechRate.toFixed(1)}x`}
                    onChange={(e) => setSpeechRate(parseFloat(e.target.value))}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                  >
                    <option value="0.8x">0.8x (慢速稳重)</option>
                    <option value="0.9x">0.9x (微慢)</option>
                    <option value="1.0x">1.0x (原速)</option>
                    <option value="1.1x">1.1x (适度微快 · 推荐)</option>
                    <option value="1.2x">1.2x (快节奏)</option>
                    <option value="1.3x">1.3x (极速)</option>
                    <option value="1.5x">1.5x (超高速)</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-white/[0.02] p-3 rounded-xl border border-slate-200/80 dark:border-white/5">
              <Info className="w-4 h-4 text-blue-600 dark:text-blue-400 shrink-0" />
              <span>已启用 Edge 免配置高质量引擎；如切换至云端商业配音将按字符计费，费用取决于最终字幕文本量。</span>
            </div>
          </div>
        )}

        {/* 阶段 3：成品视频压制设置 */}
        {doVideo && (
          <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 shadow-sm space-y-4 animate-in fade-in duration-200">
            <div className="flex items-center gap-2">
              <Film className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                成品视频压制设置
              </h3>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
              {/* 字幕方式 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">字幕方式</label>
                <div className="relative">
                  <select
                    value={subtitleOutput}
                    onChange={(e) => setSubtitleOutput(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                  >
                    <option value="burn">烧录进画面 (硬字幕 · 推荐)</option>
                    <option value="file">封装软字幕轨 (可开关)</option>
                    <option value="none">无字幕仅替换配音</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 字幕样式 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">字幕样式</label>
                <div className="relative">
                  <select
                    value={subtitleStyle}
                    onChange={(e) => setSubtitleStyle(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                  >
                    <option value="经典白字黑边">经典白字黑边</option>
                    <option value="Apple 毛玻璃底条">Apple 毛玻璃底条</option>
                    <option value="现代鲜黄高对比">现代鲜黄高对比</option>
                    <option value="电影黑底居中">电影黑底居中</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 导出画质 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">导出画质</label>
                <div className="relative">
                  <select
                    value={videoQuality}
                    onChange={(e) => setVideoQuality(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                  >
                    <option value="原画质">原画质 (Bitrate Copy)</option>
                    <option value="4K 超高清">4K 超高清 (2160P)</option>
                    <option value="1080P 高清">1080P 高清 (推荐)</option>
                    <option value="720P 标清">720P 标清</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 编码方式 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">编码方式</label>
                <div className="relative">
                  <select
                    value={encodeMethod}
                    onChange={(e) => setEncodeMethod(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors shadow-sm"
                  >
                    <option value="Apple VideoToolbox">Apple VideoToolbox (硬件加速)</option>
                    <option value="CPU">CPU libx264 (纯软解)</option>
                    <option value="HEVC / H.265">HEVC / H.265 (高压缩比)</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>
            </div>

            <div className="text-xs text-slate-500 dark:text-slate-400 flex items-center justify-between">
              <span>配音完成后将自动降低原视频原声进行智能混音。样式可在合成工作台微调并保存为「我的样式」。</span>
            </div>
          </div>
        )}

        {/* 阶段 4：人工把关 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 shadow-sm">
          <div className="flex items-center gap-2 mb-4">
            <UserCheck className="w-4 h-4 text-blue-600 dark:text-blue-400" />
            <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
              人工把关与确认 (可选)
            </h3>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* 开关 1: 字幕校对 */}
            <div className="flex items-center justify-between p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-200/80 dark:border-white/[0.08]">
              <div>
                <div className="text-xs font-bold text-slate-800 dark:text-white flex items-center gap-1.5">
                  <span>字幕校对</span>
                  <span className="text-[10px] text-blue-600 bg-blue-50 px-1.5 py-0.5 rounded-full border border-blue-200 dark:bg-blue-500/15 dark:text-blue-400 dark:border-blue-500/20">
                    推荐开启
                  </span>
                </div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">字幕完成后先停下等你检查，再进行配音/合成</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={proofread}
                  onChange={(e) => setProofread(e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-slate-200 dark:bg-white/20 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 shadow-inner" />
              </label>
            </div>

            {/* 开关 2: 配音确认 */}
            <div className="flex items-center justify-between p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.03] border border-slate-200/80 dark:border-white/[0.08]">
              <div>
                <div className="text-xs font-bold text-slate-800 dark:text-white">配音确认</div>
                <div className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5">配音完成后先停下试听确认，满意后再压制合成</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={ttsConfirm}
                  onChange={(e) => setTtsConfirm(e.target.checked)}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-slate-200 dark:bg-white/20 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 shadow-inner" />
              </label>
            </div>
          </div>

          <div className="mt-3 text-[11px] text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5 text-slate-400" />
            <span>到点系统将发送系统通知；校对/确认后点击「放行」自动继续，若全关则后台全自动极速产出。</span>
          </div>
        </div>

        {/* 错误提示浮条 */}
        {errorMessage && (
          <div className="p-3.5 rounded-xl bg-rose-50 border border-rose-200 dark:bg-rose-500/10 dark:border-rose-500/30 flex items-center gap-2.5 text-xs text-rose-600 dark:text-rose-300">
            <AlertCircle className="w-4 h-4 text-rose-500 shrink-0" />
            <span>{errorMessage}</span>
          </div>
        )}
      </div>

      {/* 下半部：独立固定操作底栏（绝不遮挡主内容滚动，左右呼应） */}
      <div className="shrink-0 bg-white/95 dark:bg-[#1C1C1E]/95 backdrop-blur-2xl border-t border-slate-200/90 dark:border-[#2C2C2E] px-7 py-3.5 flex items-center justify-between shadow-sm z-20">
        <div className="flex items-center gap-2.5">
          <div className={`w-2.5 h-2.5 rounded-full ${hasFiles ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'}`} />
          <span className="text-xs font-semibold text-slate-700 dark:text-slate-200">
            {hasFiles
              ? `已就绪：已选择 ${selectedFiles.length} 个文件 · 目标模式: ${
                  doVideo ? '成品视频' : doDub ? 'AI配音' : doTranslate ? '双语字幕' : '提取字幕'
                }`
              : '请先拖入或选择待处理音视频文件'}
          </span>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            type="button"
            onClick={() => setShowRecipeModal(true)}
            className="px-3.5 py-2 rounded-xl bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-slate-200 text-xs font-semibold border border-slate-200 dark:border-white/10 transition-all flex items-center gap-1.5 shadow-sm"
          >
            <BookmarkPlus className="w-3.5 h-3.5" />
            <span>保存为配方</span>
          </button>

          <button
            type="button"
            onClick={handleStartTask}
            disabled={loading}
            className={`px-5 py-2 rounded-xl text-xs font-bold text-white shadow-md transition-all flex items-center gap-2 ${
              loading
                ? 'bg-blue-600/50 cursor-not-allowed'
                : 'bg-blue-600 hover:bg-blue-500 active:scale-95 shadow-blue-500/25'
            }`}
          >
            {loading ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                <span>正在提交任务...</span>
              </>
            ) : (
              <>
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>{hasFiles ? `开始处理 (${selectedFiles.length}个文件)` : '选择文件并启动'}</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* 保存配方对话框 */}
      {showRecipeModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
          <div className="w-96 bg-white dark:bg-slate-900 border border-slate-200 dark:border-white/15 rounded-2xl p-5 shadow-2xl space-y-4">
            <h3 className="text-sm font-bold text-slate-900 dark:text-white">保存当前配置为配方</h3>
            <p className="text-xs text-slate-500 dark:text-slate-400">配方将保存语言、模型、音色与压制参数，方便下次一键调用。</p>
            <input
              type="text"
              placeholder="请输入配方名称，例如：科技视频译制配方"
              value={recipeName}
              onChange={(e) => setRecipeName(e.target.value)}
              className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/50"
            />
            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowRecipeModal(false)}
                className="px-3 py-1.5 rounded-xl text-xs font-medium text-slate-600 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-white/10 transition-colors"
              >
                取消
              </button>
              <button
                type="button"
                onClick={handleSaveRecipe}
                disabled={!recipeName.trim()}
                className="px-4 py-1.5 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50 transition-colors shadow-sm"
              >
                保存配方
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 配方保存成功 Toast */}
      {recipeSavedToast && (
        <div className="fixed top-6 right-6 z-50 bg-emerald-600 text-white text-xs px-4 py-2.5 rounded-xl shadow-xl flex items-center gap-2 animate-in fade-in">
          <CheckCircle2 className="w-4 h-4" />
          <span>配方已保存，可在顶部快捷模板中再次选用</span>
        </div>
      )}
    </div>
  );
};
