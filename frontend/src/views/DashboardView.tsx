import React, { useState, useEffect, useMemo, useRef } from 'react';
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
  BookmarkPlus,
  Loader2,
  AlertCircle,
} from 'lucide-react';
import { api } from '../lib/api';
import { BUILTIN_PRESETS, WorkflowPreset, ResourceType, detectResourceType, getPresetDisplay, recipeUnavailableReason, resolveWorkflowMode, subtitleContent } from '../lib/workflowRecipes';
export { detectResourceType } from '../lib/workflowRecipes';
import { useTranslation } from '../i18n';
import { EDGE_TTS_VOICES, normalizeTtsVoice, TtsEngine } from '../lib/ttsVoices';
import { whisperModelChoices, selectWhisperModel, configuredWhisperModel, WhisperModelChoice } from '../lib/whisperModels';
import { ASREngine, prepareASRSelection } from '../lib/asrSelection';
import { loadWorkbenchDraft, saveWorkbenchDraft } from '../lib/workbenchDraft';
import { configuredWorkbenchVoice, validateWorkbenchTTS } from '../lib/workbenchTTS';
import type { ConfigDTO, TTSCapabilities } from '../types';

declare global {
  interface Window {
    __onWailsFilesDropped?: (files: string[]) => void;
  }
}

interface DashboardViewProps {
  onTaskCreated: () => void;
  onConfigureASR: () => void;
  active?: boolean;
}

export function getResourceTypeName(type: ResourceType | null, t?: (k: string, def?: string) => string): string {
  switch (type) {
    case 'video': return t ? t('dashboard.resTypeVideo', '视频') : '视频';
    case 'audio': return t ? t('dashboard.resTypeAudio', '音频') : '音频';
    case 'text': return t ? t('dashboard.resTypeText', '文本/字幕') : '文本/字幕';
    default: return t ? t('dashboard.resTypeFile', '文件') : '文件';
  }
}

export const DashboardView: React.FC<DashboardViewProps> = ({ onTaskCreated, onConfigureASR, active = true }) => {
  const { t, locale } = useTranslation();
  const english = locale === 'en-US';
  const initialDraft = useMemo(loadWorkbenchDraft, []);
  // 用户自定义配方列表（持久化于 SQLite 数据库）
  const [customPresets, setCustomPresets] = useState<WorkflowPreset[]>([]);

  // 挂载时拉取服务端 SQLite 持久化的配方
  useEffect(() => {
    const fetchRecipes = async () => {
      try {
        const list = await api.listRecipes();
        if (list && list.length > 0) {
          setCustomPresets(
            list.map((r) => ({
              id: r.id,
              title: r.title,
              subtitle: r.subtitle,
              badge: r.badge || t('dashboard.badgeMyRecipe', '我的配方'),
              isCustom: true,
              goals: {
                sub: r.do_sub,
                translate: r.do_translate,
                dub: r.do_dub,
                video: r.do_video,
              },
              config: {
                sourceLang: r.source_lang,
              whisperModel: r.whisper_model,
              translateService: r.translate_service,
              ttsEngine: r.tts_engine as TtsEngine,
              targetLang: r.target_lang,
                ttsVoice: r.tts_voice,
                speechRate: r.speech_rate,
                subtitleOutput: r.subtitle_output,
              },
            }))
          );
        }
      } catch (err) {
        console.error('Failed to load recipes from SQLite database:', err);
      }
    };
    fetchRecipes();
  }, [t]);

  // 选中的快捷预设
  const [activePreset, setActivePreset] = useState<string>(initialDraft.activePreset || 'dub_full');

  // 第一步：放入的文件列表
  const [selectedFiles, setSelectedFiles] = useState<string[]>(initialDraft.selectedFiles || []);
  const [isDragging, setIsDragging] = useState(false);

  // 第二步：我要得到 (四阶段目标核心开关)
  const [doSub, setDoSub] = useState(initialDraft.doSub ?? true);
  const [doTranslate, setDoTranslate] = useState(initialDraft.doTranslate ?? true);
  const [doDub, setDoDub] = useState(initialDraft.doDub ?? true);
  const [doVideo, setDoVideo] = useState(initialDraft.doVideo ?? true);

  // 子阶段 1：字幕与翻译配置
  const [asrEngine, setAsrEngine] = useState<ASREngine>(initialDraft.asrEngine ?? '');
  const [asrWarning, setAsrWarning] = useState<string | null>(null);
  const [asrRemoteModel, setAsrRemoteModel] = useState('');
  const [whisperModel, setWhisperModel] = useState(initialDraft.whisperModel || '');
  const [downloadedWhisperModels, setDownloadedWhisperModels] = useState<WhisperModelChoice[]>([]);
  const [loadingModels, setLoadingModels] = useState(true);
  const activation = useRef({ active, version: 0 });
  if (activation.current.active !== active) activation.current = { active, version: activation.current.version + 1 };
  const [ttsConfigSnapshot, setTtsConfigSnapshot] = useState<{ version: number; value: ConfigDTO['tts'] } | null>(null);
  const ttsConfig = active && ttsConfigSnapshot?.version === activation.current.version ? ttsConfigSnapshot.value : null;
  const [ttsConfigError, setTtsConfigError] = useState('');
  const initializedDefaults = useRef(false);
  const explicitlySelectedTtsEngine = useRef(!!initialDraft.ttsEngine);
  const engineVoices = useRef<Partial<Record<TtsEngine, string>>>(initialDraft.ttsEngine && initialDraft.ttsVoice
    ? { [initialDraft.ttsEngine]: initialDraft.ttsVoice } : {});

  useEffect(() => {
    if (!active) return;
    let mounted = true;
    Promise.all([
      api.getConfig().catch(() => null),
      api.listWhisperModels().catch(() => []),
    ]).then(([config, items]) => {
      if (!mounted) return;
      const configuredModel = configuredWhisperModel(config?.pipeline);
      const configuredEngine = config?.pipeline?.whisper_mode;
      setAsrEngine((current) => current || (configuredEngine === 'openai' || configuredEngine === 'whisper-cpp' ? configuredEngine : ''));
      setAsrRemoteModel(config?.pipeline?.asr_model || '');
      const ready = whisperModelChoices(items || [], configuredModel);
      setDownloadedWhisperModels(ready);
      setLoadingModels(false);
      setWhisperModel((selection) => selectWhisperModel(selection, configuredModel, ready));
    }).finally(() => {
      if (mounted) setLoadingModels(false);
    });
    return () => { mounted = false; };
  }, [active]);

  useEffect(() => {
    setTtsConfigSnapshot(null);
    setTtsConfigError('');
    if (!active) return;
    let mounted = true;
    const version = activation.current.version;
    api.getConfig().then((config) => {
      if (!mounted) return;
      setTtsConfigSnapshot({ version, value: config.tts });
      if (!initializedDefaults.current && !initialDraft.targetLang && config.pipeline?.default_target_lang) {
        setTargetLang(config.pipeline.default_target_lang);
      }
      if (!initializedDefaults.current && !initialDraft.translateService && ['google', 'bing', 'openai'].includes(config.llm?.provider)) {
        setTranslateService(config.llm.provider as 'google' | 'bing' | 'openai');
      }
      const configuredTtsEngine = ['edge', 'openai'].includes(config.tts?.type)
        ? config.tts.type as TtsEngine
        : 'edge';
      const preferredTtsEngine = initializedDefaults.current || explicitlySelectedTtsEngine.current
        ? selectedTtsEngine.current : configuredTtsEngine;
      if (!initializedDefaults.current && !explicitlySelectedTtsEngine.current) setTtsEngine(configuredTtsEngine);
      setTtsVoice(current => {
        const voice = current || configuredWorkbenchVoice(preferredTtsEngine, config.tts);
        engineVoices.current[preferredTtsEngine] = voice;
        return voice;
      });
      initializedDefaults.current = true;
    }).catch(error => {
      if (mounted) setTtsConfigError(error instanceof Error ? error.message : String(error));
    });
    return () => { mounted = false; };
  }, [active]);

  const [videoLang, setVideoLang] = useState(initialDraft.videoLang || 'auto');
  const [targetLang, setTargetLang] = useState(initialDraft.targetLang || 'zh-CN');
  const [translateService, setTranslateService] = useState<'openai' | 'local' | 'bing' | 'google'>(initialDraft.translateService || 'bing');
  const outputContent = subtitleContent(doTranslate);

  // 子阶段 2：AI 配音配置
  const [ttsEngine, setTtsEngine] = useState<TtsEngine>(initialDraft.ttsEngine || 'edge');
  const [ttsVoice, setTtsVoice] = useState(initialDraft.ttsVoice || '');
  const [speechRate, setSpeechRate] = useState<number>(initialDraft.speechRate ?? 1.0);
  const selectedTtsEngine = useRef(ttsEngine);
  selectedTtsEngine.current = ttsEngine;
  const [ttsCapabilityState, setTtsCapabilityState] = useState<{
    config: ConfigDTO['tts']; value: TTSCapabilities | null; error: string;
  } | null>(null);
  const ttsCapabilities = active && ttsEngine === 'openai' && ttsCapabilityState?.config === ttsConfig
    ? ttsCapabilityState.value : null;
  const ttsCapabilityError = ttsCapabilityState?.config === ttsConfig ? ttsCapabilityState?.error || '' : '';

  useEffect(() => {
    setTtsCapabilityState(null);
    if (!active || !doDub || ttsEngine !== 'openai' || !ttsConfig) return;
    let mounted = true;
    api.getTTSCapabilities(ttsConfig.base_url || '', ttsConfig.api_key || '', ttsConfig.model || '').then(value => {
      if (mounted) setTtsCapabilityState({ config: ttsConfig, value, error: '' });
    }).catch(error => {
      if (mounted) setTtsCapabilityState({ config: ttsConfig, value: null, error: error instanceof Error ? error.message : String(error) });
    });
    return () => { mounted = false; };
  }, [active, doDub, ttsEngine, ttsConfig]);

  const handleTtsVoiceChange = (voice: string) => {
    engineVoices.current[ttsEngine] = voice;
    setTtsVoice(voice);
  };

  const handleTtsEngineChange = (engine: TtsEngine) => {
    explicitlySelectedTtsEngine.current = true;
    engineVoices.current[ttsEngine] = ttsVoice;
    setTtsEngine(engine);
    setTtsVoice(engineVoices.current[engine] ?? (ttsConfig ? configuredWorkbenchVoice(engine, ttsConfig) : ''));
  };

  // 子阶段 3：成品视频压制配置
  const [subtitleOutput, setSubtitleOutput] = useState(initialDraft.subtitleOutput || 'soft');
  const [subtitleStyle, setSubtitleStyle] = useState(initialDraft.subtitleStyle || '经典白字黑边');
  const [videoQuality, setVideoQuality] = useState(initialDraft.videoQuality || '原画质(推荐)');
  const [encodeMethod, setEncodeMethod] = useState(initialDraft.encodeMethod || '默认(推荐)');

  // 配方弹窗与提交状态
  const [recipeName, setRecipeName] = useState('');
  const [recipeSaveError, setRecipeSaveError] = useState<string | null>(null);
  const [savingRecipe, setSavingRecipe] = useState(false);
  const [showRecipeModal, setShowRecipeModal] = useState(false);
  const [recipeSavedToast, setRecipeSavedToast] = useState(false);
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    saveWorkbenchDraft({
      selectedFiles,
      activePreset,
      doSub,
      doTranslate,
      doDub,
      doVideo,
      asrEngine, whisperModel,
      videoLang,
      targetLang,
      translateService,
      outputContent,
      ttsEngine,
      ttsVoice,
      speechRate,
      subtitleOutput,
      subtitleStyle,
      videoQuality,
      encodeMethod,
    });
  }, [
    selectedFiles, activePreset, doSub, doTranslate, doDub, doVideo,
    asrEngine, whisperModel, videoLang, targetLang, translateService, outputContent,
    ttsEngine, ttsVoice, speechRate, subtitleOutput, subtitleStyle,
    videoQuality, encodeMethod,
  ]);

  const currentResourceType = useMemo<ResourceType | null>(() => {
    if (selectedFiles.length === 0) return null;
    return detectResourceType(selectedFiles[0]);
  }, [selectedFiles]);

  // 统一文件添加助手（同质资源校验，严禁视频、音频、文本混搭）
  const appendFiles = (incoming: unknown) => {
    let rawPaths: string[] = [];
    if (incoming instanceof FileList) {
      for (let i = 0; i < incoming.length; i++) {
        const f = incoming[i];
        const p = (f as unknown as { path?: string }).path || f.name;
        if (p) rawPaths.push(p);
      }
    } else if (Array.isArray(incoming)) {
      for (const item of incoming) {
        if (typeof item === 'string') {
          rawPaths.push(item);
        } else if (item && typeof item === 'object') {
          const p = (item as unknown as { path?: string }).path || (item as File).name;
          if (p) rawPaths.push(p);
        }
      }
    } else if (incoming && typeof incoming === 'object') {
      const obj = incoming as Record<string, unknown>;
      if (Array.isArray(obj.filenames)) {
        rawPaths = obj.filenames as string[];
      } else if (Array.isArray(obj.data)) {
        rawPaths = obj.data as string[];
      }
    }

    if (rawPaths.length === 0) return;

    // 过滤出系统支持的三大资源类型
    const validItems = rawPaths
      .map((p) => ({ path: p, type: detectResourceType(p) }))
      .filter((item): item is { path: string; type: ResourceType } => item.type !== null);

    if (validItems.length === 0) {
      setErrorMessage(t('dashboard.dropZoneSubtitle', '未检测到支持的文件格式（支持 MP4/MKV等视频、MP3/WAV等音频、TXT/SRT等文本）'));
      return;
    }

    // 确定目标资源类型
    let targetType = currentResourceType;
    if (!targetType) {
      targetType = validItems[0].type;
      const mismatchedCount = validItems.filter((it) => it.type !== targetType).length;
      if (mismatchedCount > 0) {
        setErrorMessage(
          `已自动按首个文件确定为【${getResourceTypeName(targetType, t)}】资源，已忽略 ${mismatchedCount} 个异类文件（禁止视频/音频/文本混合上传）`
        );
      }
    } else {
      const mismatched = validItems.filter((it) => it.type !== targetType);
      if (mismatched.length > 0) {
        setErrorMessage(
          `上传资源类型不一致！当前为【${getResourceTypeName(targetType, t)}】资源，已自动过滤 ${mismatched.length} 个非同类文件`
        );
      }
    }

    const matchedPaths = validItems.filter((it) => it.type === targetType).map((it) => it.path);
    if (matchedPaths.length > 0) {
      setSelectedFiles((prev) => Array.from(new Set([...prev, ...matchedPaths])));
    }
  };

  // 全局与原生窗口拖拽双轨监听
  useEffect(() => {
    if (!active) {
      setIsDragging(false);
      return;
    }
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
  }, [active, currentResourceType]);

  // 根据当前所选文件类型自适应调整阶段开关与参数限制
  useEffect(() => {
    if (currentResourceType === 'video') {
      // 视频类资源：强制锁定 1.0x 原速，避免全局倍速破坏镜头画面与台词意境对应
      setSpeechRate(1.0);
      setDoSub(true);
      if (!doSub) setActivePreset('custom');
    } else if (currentResourceType === 'text') {
      // 文本类资源：纯配音模式
      setDoSub(false);
      setDoTranslate(false);
      setDoDub(true);
      setDoVideo(false);
      setActivePreset('text_dub');
    } else if (currentResourceType === 'audio') {
      // 音频类资源先听写，且没有可合成的视频画面
      setDoSub(true);
      setDoVideo(false);
      if (!doSub || doVideo) {
        setActivePreset('custom');
      }
    }
  }, [currentResourceType]);

  // 切换预设模板
  const handleSelectPreset = (preset: WorkflowPreset) => {
    const unavailable = recipeUnavailableReason({ do_sub: preset.goals.sub, do_video: preset.goals.video }, currentResourceType);
    if (unavailable) {
      setErrorMessage(english ? 'This recipe is incompatible with the input. Video and audio require transcription; text has no video picture.' : unavailable);
      return;
    }
    setActivePreset(preset.id);
    setDoSub(currentResourceType === 'text' ? false : preset.goals.sub);
    setDoTranslate(preset.goals.translate);
    setDoDub(preset.goals.dub);
    setDoVideo(currentResourceType === 'text' || currentResourceType === 'audio' ? false : preset.goals.video);
    if (preset.config) {
      if (preset.config.sourceLang) setVideoLang(preset.config.sourceLang);
      if (preset.config.whisperModel) setWhisperModel(preset.config.whisperModel);
      if (preset.config.translateService) setTranslateService((preset.config.translateService === 'local' ? 'openai' : preset.config.translateService) as 'bing' | 'google' | 'openai');
      if (preset.config.ttsEngine) handleTtsEngineChange(preset.config.ttsEngine);
      if (preset.config.targetLang) setTargetLang(preset.config.targetLang);
      if (preset.config.ttsVoice) {
        const engine = preset.config.ttsEngine || ttsEngine;
        const voice = engine === 'openai' ? preset.config.ttsVoice : normalizeTtsVoice('edge', preset.config.ttsVoice);
        engineVoices.current[engine] = voice;
        setTtsVoice(voice);
      }
      if (preset.config.speechRate) setSpeechRate(currentResourceType === 'video' ? 1.0 : preset.config.speechRate);
      if (preset.config.subtitleOutput) setSubtitleOutput(preset.config.subtitleOutput);
    }
  };

  // 目标开关联动逻辑（严格对齐合法资源组合）
  const handleToggleSub = () => {
    if (currentResourceType === 'text') {
      setErrorMessage(t('dashboard.stepWhisperDescText', '纯文本资源无需执行语音听写'));
      return;
    }
    // 视频和音频资源必须依赖听写转录
    if (currentResourceType === 'video' || currentResourceType === 'audio') {
      return;
    }
    setDoSub((prev) => !prev);
    setActivePreset('custom');
  };

  const handleToggleTranslate = () => {
    setDoTranslate((prev) => {
      const next = !prev;
      // 视频资源下：如果不翻译，且未开配音，则自动开启配音与成片，进入原文配音模式
      if (currentResourceType === 'video' && !next && !doDub) {
        setDoDub(true);
        setDoVideo(true);
      }
      return next;
    });
    setActivePreset('custom');
  };

  const handleToggleDub = () => {
    setDoDub((prev) => {
      const next = !prev;
      if (!next) {
        // 关闭配音时：若是视频，成片自动关闭，自动保持字幕翻译模式
        setDoVideo(false);
        if (currentResourceType === 'video') {
          setDoTranslate(true);
        }
      } else {
        // 开启配音时：若是视频，自动联动开启压制成片
        if (currentResourceType === 'video') {
          setDoVideo(true);
        }
      }
      return next;
    });
    setActivePreset('custom');
  };

  const handleToggleVideo = () => {
    if (currentResourceType === 'text') {
      setErrorMessage(t('dashboard.stepBurnDescDisabled', '纯文本资源无画面，仅支持文本翻译或朗读配音'));
      return;
    }
    setDoVideo((prev) => {
      const next = !prev;
      if (next) {
        // 开启成片，必须有配音
        if (!doDub) setDoDub(true);
      } else {
        // 关闭成片，若是视频则落入仅双语/译文字幕模式
        if (currentResourceType === 'video') {
          setDoDub(false);
          setDoTranslate(true);
        }
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

  const resolvedMode = () => resolveWorkflowMode({ do_sub: doSub, do_translate: doTranslate, do_dub: doDub, do_video: doVideo });

  // 提交并创建任务
  const handleStartTask = async () => {
    if (selectedFiles.length === 0 && !doSub) {
      await handlePickFiles();
      return;
    }

    setLoading(true);
    setErrorMessage(null);

    const mode = resolvedMode();
    const finalSubtitleOutput = doVideo ? subtitleOutput : 'file';
    const finalTranslator = translateService === 'local' ? 'openai' : translateService;
    const preset = [...BUILTIN_PRESETS, ...customPresets].find(item => item.id === activePreset) || BUILTIN_PRESETS.find(item => item.id === 'custom')!;
    const recipeName = getPresetDisplay(preset, t).title;

    try {
      const unavailable = recipeUnavailableReason({ do_sub: doSub, do_video: doVideo }, currentResourceType);
      if (unavailable) throw new Error(unavailable);
      if (doSub) {
        try {
          await prepareASRSelection(asrEngine, whisperModel);
        } catch (err) {
          setAsrWarning((err as Error).message);
          return;
        }
      }
      if (selectedFiles.length === 0) {
        await handlePickFiles();
        return;
      }
      const selectedVoice = ttsVoice || (ttsConfig ? configuredWorkbenchVoice(ttsEngine, ttsConfig) : '');
      if (doDub && ttsEngine === 'openai') {
        if (!ttsConfig) throw new Error(ttsConfigError || (english ? 'Wait for the speech settings to load.' : '请等待语音合成配置加载完成。'));
        const error = validateWorkbenchTTS(ttsCapabilities, ttsConfig, selectedVoice);
        const messages = {
          capabilities: ttsCapabilityError || (english ? 'Wait for the speech capabilities to load.' : '请等待语音服务能力查询完成。'),
          voice: english ? 'The draft or recipe voice is unavailable. Select a voice from the current service.' : '草稿或配方音色不在当前服务清单中，请重新选择音色。',
          instructions: english ? 'This model does not support the configured instructions.' : '当前模型不支持所配置的情绪指令，请修改语音设置。',
          instructions_length: english ? 'Instructions cannot exceed 4096 characters.' : '情绪指令不能超过 4096 字符，请修改语音设置。',
          language: english ? 'The configured language is unavailable.' : '当前模型不支持所配置的配音语言，请修改语音设置。',
        };
        if (error) throw new Error(messages[error]);
      }
      if (selectedFiles.length === 1) {
        await api.createTask({
          input_path: selectedFiles[0],
          recipe_name: recipeName,
          mode,
          source_lang: videoLang,
          target_lang: targetLang,
          translator: finalTranslator,
          output_content: outputContent,
          tts_engine: ttsEngine,
          tts_voice: selectedVoice,
          speech_rate: speechRate,
          subtitle_output: finalSubtitleOutput,
        });
      } else {
        await api.batchCreateTasks(selectedFiles, {
          input_path: '',
          recipe_name: recipeName,
          mode,
          source_lang: videoLang,
          target_lang: targetLang,
          translator: finalTranslator,
          output_content: outputContent,
          tts_engine: ttsEngine,
          tts_voice: selectedVoice,
          speech_rate: speechRate,
          subtitle_output: finalSubtitleOutput,
        });
      }

      setSelectedFiles([]);
      onTaskCreated();
    } catch (err: unknown) {
      console.error('Failed to create task:', err);
      setErrorMessage((err as Error).message || '创建任务失败，请检查音视频格式及后端配置');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveRecipe = async () => {
    if (!recipeName.trim() || savingRecipe) return;
    setRecipeSaveError(null);
    setSavingRecipe(true);
    const subtitleParts: string[] = [];
    if (doSub) subtitleParts.push(t('dashboard.stepWhisper', '听写转录'));
    if (doTranslate) subtitleParts.push(t('dashboard.stepTranslate', '翻译字幕'));
    if (doDub) subtitleParts.push(t('dashboard.stepDub', 'AI 配音'));
    if (doVideo) subtitleParts.push(t('dashboard.stepBurn', '压制合成'));

    const newRecipe: WorkflowPreset = {
      id: `rcp_${Date.now()}`,
      title: recipeName.trim(),
      subtitle: subtitleParts.join(' · ') || t('dashboard.presetCustom', '自定义智能流程'),
      badge: t('dashboard.badgeMyRecipe', '我的配方'),
      isCustom: true,
      goals: { sub: doSub, translate: doTranslate, dub: doDub, video: doVideo },
      config: {
        targetLang,
        ttsVoice,
        speechRate,
        subtitleOutput,
      },
    };

    try {
      await api.saveRecipe({
        id: newRecipe.id,
        title: newRecipe.title,
        subtitle: newRecipe.subtitle,
        badge: newRecipe.badge,
        is_custom: true,
        do_sub: doSub,
        do_translate: doTranslate,
        do_dub: doDub,
        do_video: doVideo,
        target_lang: targetLang,
        source_lang: videoLang,
        whisper_model: whisperModel,
        translate_service: translateService,
        tts_engine: ttsEngine,
        tts_voice: ttsVoice,
        speech_rate: speechRate,
        subtitle_output: subtitleOutput,
        subtitle_style: subtitleStyle,
        video_quality: videoQuality,
      });

      const serverList = await api.listRecipes();
      if (serverList && serverList.length > 0) {
        setCustomPresets(
          serverList.map((r) => ({
            id: r.id,
            title: r.title,
            subtitle: r.subtitle,
            badge: r.badge || t('dashboard.badgeMyRecipe', '我的配方'),
            isCustom: true,
            goals: { sub: r.do_sub, translate: r.do_translate, dub: r.do_dub, video: r.do_video },
            config: {
              sourceLang: r.source_lang,
              whisperModel: r.whisper_model,
              translateService: r.translate_service,
              ttsEngine: r.tts_engine as TtsEngine,
              targetLang: r.target_lang,
              ttsVoice: r.tts_voice,
              speechRate: r.speech_rate,
              subtitleOutput: r.subtitle_output,
            },
          }))
        );
      } else {
        setCustomPresets((prev) => [newRecipe, ...prev]);
      }
    } catch (err) {
      setRecipeSaveError(err instanceof Error ? err.message : String(err));
      return;
    } finally {
      setSavingRecipe(false);
    }

    setActivePreset(newRecipe.id);
    setRecipeName('');
    setShowRecipeModal(false);
    setRecipeSavedToast(true);
    setTimeout(() => setRecipeSavedToast(false), 2500);
  };

  const handleDeleteCustomPreset = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    try {
      await api.deleteRecipe(id);
    } catch (err) {
      console.error('Failed to delete recipe from database:', err);
    }
    setCustomPresets((prev) => prev.filter((p) => p.id !== id));
    if (activePreset === id) {
      setActivePreset('dub_full');
    }
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
                <span>{t('dashboard.startCreation', '开始创作')}</span>
              </h1>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                {t('dashboard.mainDesc', '选择推荐配方或自由组合阶段，将原声媒体全自动转化为多语种优质内容')}
              </p>
            </div>
          </div>

          {/* 快捷模板网格：统摄为克制清朗的 Apple 风格，动态融合内置与自定义配方 */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
            {[...BUILTIN_PRESETS, ...customPresets].map((preset) => {
              const isSelected = activePreset === preset.id;
              const display = getPresetDisplay(preset, t);
              const unavailable = recipeUnavailableReason({ do_sub: preset.goals.sub, do_video: preset.goals.video }, currentResourceType);
              return (
                <div
                  key={preset.id}
                  onClick={() => handleSelectPreset(preset)}
                  aria-disabled={!!unavailable}
                  title={unavailable || display.subtitle}
                  className={`text-left rounded-2xl p-4 border transition-all relative overflow-hidden group ${unavailable ? 'opacity-45 cursor-not-allowed' : 'cursor-pointer'} ${
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
                      {display.badge}
                    </span>
                    <div className="flex items-center space-x-1">
                      {preset.isCustom && (
                        <button
                          type="button"
                          onClick={(e) => handleDeleteCustomPreset(e, preset.id)}
                          className="opacity-0 group-hover:opacity-100 p-1 rounded-md text-slate-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-opacity"
                          title={t('dashboard.deleteRecipeTip', '删除此自定义配方')}
                        >
                          <X size={12} />
                        </button>
                      )}
                      {isSelected && <CheckCircle2 className="w-4 h-4 text-blue-600 dark:text-blue-400 shrink-0" />}
                    </div>
                  </div>
                  <h3 className={`text-sm font-bold transition-colors ${
                    isSelected ? 'text-blue-900 dark:text-white' : 'text-slate-800 dark:text-slate-100 group-hover:text-blue-600 dark:group-hover:text-blue-400'
                  }`}>
                    {display.title}
                  </h3>
                  <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1 line-clamp-2 leading-relaxed">
                    {display.subtitle}
                  </p>
                </div>
              );
            })}
          </div>
        </div>

        {/* 第一步：放入文件 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2.5">
              <span className="w-6 h-6 rounded-full bg-blue-600 text-white text-xs font-bold flex items-center justify-center">
                1
              </span>
              <div>
                <h2 className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">
                  {t('dashboard.step1Title', '第一步 · 放入待处理文件')}
                </h2>
                <p className="text-[11px] text-slate-500 dark:text-slate-400">
                  {t('dashboard.step1Desc', '支持音视频及字幕文本，放入多个即可批量排队处理')}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {hasFiles && (
                <button
                  type="button"
                  onClick={() => setSelectedFiles([])}
                  className="text-xs text-slate-500 hover:text-rose-600 dark:hover:text-rose-400 px-2.5 py-1 rounded-lg hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-colors"
                >
                  {t('dashboard.clearList', '清空列表')}
                </button>
              )}
              <button
                type="button"
                onClick={handlePickFiles}
                className="text-xs font-semibold px-3 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-white border border-slate-200 dark:border-white/10 transition-all flex items-center gap-1.5"
              >
                <UploadCloud className="w-3.5 h-3.5" />
                <span>{t('dashboard.browseFiles', '选择本地文件')}</span>
              </button>
            </div>
          </div>

          {/* 已放入的文件列表标签 */}
          {hasFiles && (
            <div className={`mb-4 max-h-48 overflow-y-auto pr-1 ${selectedFiles.length > 5 ? 'flex flex-wrap gap-2' : 'space-y-2'}`}>
              {selectedFiles.map((file, idx) => {
                const fileName = file.split('/').pop() || file;
                const compactFiles = selectedFiles.length > 5;
                const nameChars = [...fileName];
                const displayName = nameChars.length > 35 ? `${nameChars.slice(0, 35).join('')}…` : fileName;
                const isSub = fileName.endsWith('.srt') || fileName.endsWith('.ass') || fileName.endsWith('.vtt');
                return (
                  <div
                    key={idx}
                    title={compactFiles ? fileName : undefined}
                    style={compactFiles ? { width: `${Math.min(35, Math.max(12, nameChars.length)) + 7}ch` } : undefined}
                    className={`flex items-center justify-between bg-slate-50 dark:bg-white/[0.04] border border-slate-200 dark:border-white/10 rounded-xl group hover:border-blue-400/50 transition-all ${compactFiles ? 'max-w-full px-2.5 py-2' : 'px-3.5 py-2.5'}`}
                  >
                    <div className={`flex items-center min-w-0 ${compactFiles ? 'gap-2 pr-1' : 'gap-2.5 pr-2'}`}>
                      <div className={`${compactFiles ? 'w-5 h-5 rounded-md' : 'w-7 h-7 rounded-lg'} bg-blue-50 dark:bg-blue-500/20 text-blue-600 dark:text-blue-400 flex items-center justify-center shrink-0`}>
                        {isSub ? <FileText className={compactFiles ? 'w-3 h-3' : 'w-4 h-4'} /> : <FileVideo className={compactFiles ? 'w-3 h-3' : 'w-4 h-4'} />}
                      </div>
                      <div className="min-w-0 truncate">
                        <p className="text-xs font-semibold text-slate-800 dark:text-slate-200 truncate">{compactFiles ? displayName : fileName}</p>
                        {!compactFiles && <p className="text-[10px] text-slate-400 truncate">{file}</p>}
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleRemoveFile(idx)}
                      className="w-6 h-6 rounded-lg text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 flex items-center justify-center transition-colors shrink-0"
                      title={t('dashboard.removeFile', '移除此文件')}
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
              {t('dashboard.dropZoneTitle', '点击或拖拽音视频及文本文件至此处')}
            </p>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1">
              {t('dashboard.dropZoneSubtitle', '支持 MP4, MKV, MOV, MP3, WAV, TXT, SRT 等格式，自动智能识别类型')}
            </p>
          </div>
        </div>

        {/* 第二步：我要得到 (统摄 Apple 极简蓝风格) */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <span className="w-6 h-6 rounded-full bg-blue-600 text-white text-xs font-bold flex items-center justify-center">
                2
              </span>
              <div>
                <h2 className="text-sm font-bold text-slate-900 dark:text-white tracking-tight">
                  {t('dashboard.step2Title', '第二步 · 创作期望产物')}
                </h2>
                <p className="text-[11px] text-slate-500 dark:text-slate-400">
                  {t('dashboard.step2Desc', '勾选所需产物阶段，下方将动态呈现对应的子配置参数')}
                </p>
              </div>
            </div>

            {currentResourceType === 'text' && (
              <span className="px-2.5 py-1 rounded-full text-[11px] font-semibold bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20 flex items-center gap-1 animate-in fade-in">
                <FileText className="w-3 h-3" />
                {t('dashboard.modeText', '纯文本资源模式 (仅翻译/配音)')}
              </span>
            )}
            {currentResourceType === 'audio' && (
              <span className="px-2.5 py-1 rounded-full text-[11px] font-semibold bg-indigo-500/10 text-indigo-600 dark:text-indigo-400 border border-indigo-500/20 flex items-center gap-1 animate-in fade-in">
                <Volume2 className="w-3 h-3" />
                {t('dashboard.modeAudio', '音频资源模式 (无画面不可成片)')}
              </span>
            )}
            {currentResourceType === 'video' && (
              <span className="px-2.5 py-1 rounded-full text-[11px] font-semibold bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 flex items-center gap-1 animate-in fade-in">
                <Film className="w-3 h-3" />
                {t('dashboard.modeVideo', '视频资源模式 (全流程)')}
              </span>
            )}
          </div>

          {/* 四阶段目标卡片横排：根据资源类型精准受控 */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {/* 目标 1: 听写转录 */}
            <div
              onClick={handleToggleSub}
              className={`p-3.5 rounded-xl border transition-all relative flex flex-col justify-between h-24 ${
                currentResourceType === 'text'
                  ? 'opacity-40 cursor-not-allowed bg-slate-50/50 dark:bg-white/[0.02] border-slate-200/50'
                  : doSub
                  ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600 cursor-pointer'
                  : 'bg-slate-50/50 hover:bg-slate-100/80 dark:bg-white/[0.02] border-slate-200/80 dark:border-white/10 opacity-75 hover:opacity-100 cursor-pointer'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className={`w-7 h-7 rounded-lg flex items-center justify-center ${
                  doSub ? 'bg-blue-600 text-white' : 'bg-slate-200 dark:bg-white/10 text-slate-500 dark:text-slate-400'
                }`}>
                  <Mic className="w-4 h-4" />
                </div>
                <div className={`w-4 h-4 rounded-full flex items-center justify-center transition-all ${
                  doSub ? 'bg-blue-600 text-white' : 'border border-slate-300 dark:border-slate-600 bg-transparent'
                }`}>
                  {doSub && <CheckCircle2 className="w-3.5 h-3.5 stroke-[2.5]" />}
                </div>
              </div>
              <div>
                <div className={`text-xs font-bold ${doSub ? 'text-blue-950 dark:text-white' : 'text-slate-700 dark:text-slate-300'}`}>
                  {t('dashboard.stepWhisper', '听写转录')}
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">
                  {currentResourceType === 'text' ? t('dashboard.stepWhisperDescText') : t('dashboard.stepWhisperDescVideo')}
                </div>
              </div>
            </div>

            {/* 目标 2: 翻译字幕 / 文本翻译 */}
            <div
              onClick={handleToggleTranslate}
              className={`p-3.5 rounded-xl border transition-all relative flex flex-col justify-between h-24 ${
                doTranslate
                  ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600 cursor-pointer'
                  : 'bg-slate-50/50 hover:bg-slate-100/80 dark:bg-white/[0.02] border-slate-200/80 dark:border-white/10 opacity-75 hover:opacity-100 cursor-pointer'
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
                  {currentResourceType === 'text' ? t('dashboard.stepTranslateText', '智能翻译') : t('dashboard.stepTranslate', '翻译字幕')}
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">
                  {currentResourceType === 'text' ? t('dashboard.stepTranslateDescText') : t('dashboard.stepTranslateDescVideo')}
                </div>
              </div>
            </div>

            {/* 目标 3: AI 配音 */}
            <div
              onClick={handleToggleDub}
              className={`p-3.5 rounded-xl border transition-all relative flex flex-col justify-between h-24 ${
                doDub
                  ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600 cursor-pointer'
                  : 'bg-slate-50/50 hover:bg-slate-100/80 dark:bg-white/[0.02] border-slate-200/80 dark:border-white/10 opacity-75 hover:opacity-100 cursor-pointer'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className={`w-7 h-7 rounded-lg flex items-center justify-center ${
                  doDub ? 'bg-blue-600 text-white' : 'bg-slate-200 dark:bg-white/10 text-slate-500 dark:text-slate-400'
                }`}>
                  <Volume2 className="w-4 h-4" />
                </div>
                <div className={`w-4 h-4 rounded-full flex items-center justify-center transition-all ${
                  doDub ? 'bg-blue-600 text-white' : 'border border-slate-300 dark:border-slate-600 bg-transparent'
                }`}>
                  {doDub && <CheckCircle2 className="w-3.5 h-3.5 stroke-[2.5]" />}
                </div>
              </div>
              <div>
                <div className={`text-xs font-bold ${doDub ? 'text-blue-950 dark:text-white' : 'text-slate-700 dark:text-slate-300'}`}>
                  {t('dashboard.stepDub', 'AI 配音')} {currentResourceType === 'text' && <span className="text-[10px] text-blue-600 dark:text-blue-400 font-normal ml-1">({t('dashboard.requiredBadge', '必选')})</span>}
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">
                  {currentResourceType === 'text' ? t('dashboard.stepDubDescText') : t('dashboard.stepDubDescVideo')}
                </div>
              </div>
            </div>

            {/* 目标 4: 压制合成 */}
            <div
              onClick={handleToggleVideo}
              className={`p-3.5 rounded-xl border transition-all relative flex flex-col justify-between h-24 ${
                currentResourceType === 'text' || currentResourceType === 'audio'
                  ? 'opacity-40 cursor-not-allowed bg-slate-50/50 dark:bg-white/[0.02] border-slate-200/50'
                  : doVideo
                  ? 'bg-blue-50/40 dark:bg-blue-500/10 border-blue-600 cursor-pointer'
                  : 'bg-slate-50/50 hover:bg-slate-100/80 dark:bg-white/[0.02] border-slate-200/80 dark:border-white/10 opacity-75 hover:opacity-100 cursor-pointer'
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
                  {t('dashboard.stepBurn', '压制合成')}
                </div>
                <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">
                  {currentResourceType === 'text' || currentResourceType === 'audio' ? t('dashboard.stepBurnDescDisabled') : t('dashboard.stepBurnDescVideo')}
                </div>
              </div>
            </div>
          </div>

          {/* 动态处理流程拓扑条：统一为克制经典的蓝灰格调 */}
          <div className="p-3.5 rounded-xl bg-slate-50 dark:bg-white/[0.02] border border-slate-200/80 dark:border-white/5 flex flex-wrap items-center gap-2 text-xs">
            <span className="text-slate-500 dark:text-slate-400 font-medium">
              {t('dashboard.currentPipeline', '当前处理流程:')}
            </span>
            <div className="flex flex-wrap items-center gap-1.5">
              <span className={`px-2.5 py-1 rounded-full text-[11px] font-mono flex items-center gap-1 border ${
                doSub
                  ? 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/40'
                  : 'bg-slate-100 text-slate-400 border-slate-200 dark:bg-white/5 dark:text-slate-500 dark:border-white/5'
              }`}>
                <Mic className="w-3 h-3" /> {t('dashboard.stepWhisper', '听写转录')}
              </span>

              <ChevronRight className={`w-3.5 h-3.5 ${doTranslate ? 'text-blue-500' : 'text-slate-300 dark:text-slate-700'}`} />

              <span className={`px-2.5 py-1 rounded-full text-[11px] font-mono flex items-center gap-1 border ${
                doTranslate
                  ? 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/40'
                  : 'bg-slate-100 text-slate-400 border-slate-200 dark:bg-white/5 dark:text-slate-500 dark:border-white/5'
              }`}>
                <Languages className="w-3 h-3" /> {t('dashboard.stepTranslate', '翻译字幕')}
              </span>

              <ChevronRight className={`w-3.5 h-3.5 ${doDub ? 'text-blue-500' : 'text-slate-300 dark:text-slate-700'}`} />

              <span className={`px-2.5 py-1 rounded-full text-[11px] font-mono flex items-center gap-1 border ${
                doDub
                  ? 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/40'
                  : 'bg-slate-100 text-slate-400 border-slate-200 dark:bg-white/5 dark:text-slate-500 dark:border-white/5'
              }`}>
                <Volume2 className="w-3 h-3" /> {t('dashboard.stepDub', 'AI 配音')}
              </span>

              <ChevronRight className={`w-3.5 h-3.5 ${doVideo ? 'text-blue-500' : 'text-slate-300 dark:text-slate-700'}`} />

              <span className={`px-2.5 py-1 rounded-full text-[11px] font-mono flex items-center gap-1 border ${
                doVideo
                  ? 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-500/20 dark:text-blue-300 dark:border-blue-500/40'
                  : 'bg-slate-100 text-slate-400 border-slate-200 dark:bg-white/5 dark:text-slate-500 dark:border-white/5'
              }`}>
                <Film className="w-3 h-3" /> {t('dashboard.stepBurn', '压制合成')}
              </span>
            </div>
          </div>
        </div>

        {/* 阶段 1：字幕设置 */}
        {(doSub || doTranslate) && (
          <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 space-y-4 animate-in fade-in duration-200">
            <div className="flex items-center gap-2">
              <Languages className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                {t('dashboard.sectionSubtitle', '字幕设置')}
              </h3>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            {doSub && <div>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                {t('dashboard.asrEngine', '语音识别引擎')}
              </label>
              <select value={asrEngine} onChange={(e) => setAsrEngine(e.target.value as ASREngine)}
                className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white">
                <option value="">{t('dashboard.chooseAsrEngine', '请选择识别引擎')}</option>
                <option value="whisper-cpp">Whisper.cpp</option>
                <option value="openai">{t('asrEngine.openaiCompat', 'OpenAI 兼容')}</option>
              </select>
            </div>}
            {doSub && asrEngine === 'openai' && <div>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                {t('dashboard.whisperModel', '语音识别模型')}
              </label>
              <button type="button" onClick={onConfigureASR} className="w-full h-10 text-left bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white">
                {asrRemoteModel || t('dashboard.configureAsr', '前往语音识别设置')}
              </button>
            </div>}
            {/* 语音模型 */}
            {doSub && asrEngine === 'whisper-cpp' && <div>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                {t('dashboard.whisperModel', '语音识别模型')}
              </label>
              <div className="relative">
                <select
                  value={whisperModel}
                  onChange={(e) => setWhisperModel(e.target.value)}
                  disabled={downloadedWhisperModels.length === 0}
                  className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {downloadedWhisperModels.length === 0 ? (
                    <option value="">
                      {loadingModels
                        ? t('dashboard.loadingModels', '正在加载可用模型...')
                        : t('dashboard.noDownloadedModels', '暂无已下载模型 (请前往语音引擎下载)')}
                    </option>
                  ) : (
                    downloadedWhisperModels.map((model) => (
                      <option key={model.path} value={model.path} title={model.path}>
                        {model.custom ? `${t('dashboard.customModel', '自定义')} · ` : ''}{model.name}
                      </option>
                    ))
                  )}
                </select>
                <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
              </div>
            </div>}

            {/* 视频源语言 */}
            <div>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                {t('dashboard.sourceLang', '视频源语言')}
              </label>
              <div className="relative">
                <select
                  value={videoLang}
                  onChange={(e) => setVideoLang(e.target.value)}
                  className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                >
                  <option value="auto">{english ? 'Auto Detect' : t('dashboard.autoDetect', '自动检测')}</option>
                  <option value="en">English</option>
                  <option value="zh">{english ? 'Chinese (Simplified)' : '简体中文'}</option>
                  <option value="ja">{english ? 'Japanese' : '日本語'}</option>
                  <option value="ko">{english ? 'Korean' : '한국어'}</option>
                  <option value="de">{english ? 'German' : 'Deutsch'}</option>
                  <option value="fr">{english ? 'French' : 'Français'}</option>
                </select>
                <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
              </div>
            </div>

            {doSub && <p className="sm:col-span-2 lg:col-span-4 text-xs text-slate-500">{english ? 'Transcription always saves source subtitles.' : '听写转录固定提取并保存原文字幕。'}</p>}
          </div>
        </div>
        )}

        {/* 翻译设置 */}
        {doTranslate && (
          <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 space-y-4 animate-in fade-in duration-200">
            <div className="flex items-center gap-2">
              <Languages className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                {t('dashboard.sectionTranslation', '翻译设置')}
              </h3>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            {/* 翻译引擎 */}
            <div>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                {t('dashboard.translateService', '翻译引擎')}
              </label>
              <div className="relative">
                <select
                  value={translateService === 'local' ? 'openai' : translateService}
                  onChange={(e) => setTranslateService(e.target.value as 'bing' | 'google' | 'openai')}
                  className="w-full h-9 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-lg pl-3 pr-9 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                >
                  <option value="bing">{english ? 'Bing Translator' : '必应翻译'}</option>
                  <option value="google">{english ? 'Google Translate' : 'Google 翻译'}</option>
                  <option value="openai">{english ? 'OpenAI Compatible' : 'OpenAI 兼容'}</option>
                </select>
                <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-2.5 pointer-events-none" />
              </div>
            </div>

            {/* 翻译目标语言 */}
            <div>
              <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                {t('dashboard.targetLang', '翻译目标语言')}
              </label>
              <div className="relative">
                <select
                  value={targetLang}
                  onChange={(e) => setTargetLang(e.target.value)}
                  className="w-full h-9 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-lg pl-3 pr-9 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                >
                  <option value="zh-CN">{english ? 'Chinese (Simplified)' : '简体中文'}</option>
                  <option value="zh-TW">{english ? 'Chinese (Traditional)' : '繁體中文'}</option>
                  <option value="en">English</option>
                  <option value="ja">{english ? 'Japanese' : '日本語'}</option>
                  <option value="ko">{english ? 'Korean' : '한국어'}</option>
                  <option value="de">German</option>
                  <option value="fr">French</option>
                </select>
                <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-2.5 pointer-events-none" />
              </div>
            </div>


            </div>
            <p className="text-xs text-slate-500">{english ? 'Translation always saves translated subtitles; source subtitles are retained.' : '翻译固定输出翻译字幕，原文字幕同时保留。'}</p>
          </div>
        )}

        {/* 阶段 2：AI 配音设置 */}
        {doDub && (
          <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 space-y-4 animate-in fade-in duration-200">
            <div className="flex items-center gap-2">
              <Volume2 className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                {t('dashboard.sectionDub', 'AI 语音合成与配音设置')}
              </h3>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              {/* 配音引擎 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                  {t('dashboard.ttsEngine', '配音引擎')}
                </label>
                <div className="relative">
                  <select
                    value={ttsEngine}
                    onChange={(e) => handleTtsEngineChange(e.target.value as TtsEngine)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                  >
                    <option value="edge">Edge TTS</option>
                    <option value="openai">{t('dashboard.openaiTTS', 'OpenAI 兼容 TTS')}</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 音色选择 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                  {t('dashboard.ttsVoice', '声音音色')}
                </label>
                <div className="relative">
                  {ttsEngine === 'edge' ? (
                    <>
                      <select
                        value={ttsVoice}
                        onChange={(e) => handleTtsVoiceChange(e.target.value)}
                        className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                      >
                        {EDGE_TTS_VOICES.map((voice) => (
                          <option key={voice.value} value={voice.value}>
                            {english ? voice.enLabel : voice.zhLabel}
                          </option>
                        ))}
                      </select>
                      <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                    </>
                  ) : ttsCapabilities?.voices.length ? (
                    <select aria-label={english ? 'Speech voice' : '声音音色'}
                      value={ttsCapabilities.voices.some(voice => voice.id === ttsVoice) ? ttsVoice : ''}
                      onChange={event => handleTtsVoiceChange(event.target.value)}
                      className="w-full h-10 bg-slate-50 dark:bg-white/[0.06] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40">
                      <option value="">{english ? 'Select a voice' : '请选择音色'}</option>
                      {ttsCapabilities.voices.map(voice => <option key={voice.id} value={voice.id}>{voice.name || voice.id}</option>)}
                    </select>
                  ) : (
                    <input
                      type="text"
                      value={ttsVoice}
                      onChange={(e) => handleTtsVoiceChange(e.target.value)}
                      disabled={!ttsCapabilities || ttsCapabilities.voice_source !== 'manual'}
                      maxLength={256}
                      placeholder={english ? 'Enter a voice supported by the service' : '填写服务端支持的音色名'}
                      className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                    />
                  )}
                </div>
                {ttsEngine === 'openai' && (
                  <p className="text-[11px] text-slate-500 mt-1.5" role={ttsConfigError || ttsCapabilityError ? 'alert' : undefined}>
                    {ttsConfigError || ttsCapabilityError || (!ttsCapabilities
                      ? (english ? 'Loading the current speech service capabilities…' : '正在读取当前语音服务能力…')
                      : ttsCapabilities.voices.length > 0 && ttsVoice && !ttsCapabilities.voices.some(voice => voice.id === ttsVoice)
                          ? (english ? `The draft or recipe voice “${ttsVoice}” is unavailable. Select a voice again.` : `草稿或配方音色“${ttsVoice}”不可用，请重新选择。`)
                          : ttsCapabilities.voice_source === 'qwen_builtin'
                            ? (english ? 'Built-in Qwen voices; the service did not return a voice list.' : 'Qwen 内置音色，服务未返回音色列表。')
                            : ttsCapabilities.voice_source === 'remote'
                              ? (english ? 'Choose a voice returned by the current service.' : '请选择当前服务返回的音色。')
                              : (english ? 'No voice discovery endpoint; enter a supported voice.' : '服务未提供音色查询接口，请填写其支持的音色。'))}
                  </p>
                )}
              </div>

              {/* 语速倍率 */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400">
                    {t('dashboard.speechRate', '配音语速')}
                  </label>
                  {currentResourceType === 'video' && (
                    <span className="text-[10px] text-blue-600 dark:text-blue-400 font-medium">
                      {t('dashboard.rateLockedTip', '视频原画已锁定 1.0x')}
                    </span>
                  )}
                </div>
                <div className="relative">
                  <select
                    value={`${speechRate.toFixed(1)}x`}
                    disabled={currentResourceType === 'video'}
                    onChange={(e) => setSpeechRate(parseFloat(e.target.value))}
                    className={`w-full h-10 border rounded-xl px-3 text-[13px] appearance-none transition-colors ${
                      currentResourceType === 'video'
                        ? 'bg-slate-100 dark:bg-white/[0.03] text-slate-400 dark:text-slate-500 border-slate-200 dark:border-white/10 cursor-not-allowed'
                        : 'bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] text-slate-800 dark:text-white border-slate-200 dark:border-white/15 cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40'
                    }`}
                  >
                    <option value="1.0x">{english ? '1.0x (Recommended)' : '1.0x (推荐)'}</option>
                    <option value="0.9x">{english ? '0.9x (Slightly slower)' : '0.9x (微慢)'}</option>
                    <option value="0.8x">{english ? '0.8x (Slow)' : '0.8x (慢速稳重)'}</option>
                    <option value="1.1x">{english ? '1.1x (Slightly faster)' : '1.1x (适度微快)'}</option>
                    <option value="1.2x">{english ? '1.2x (Fast)' : '1.2x (快节奏)'}</option>
                    <option value="1.3x">{english ? '1.3x (Very fast)' : '1.3x (极速)'}</option>
                    <option value="1.5x">{english ? '1.5x (Maximum)' : '1.5x (超高速)'}</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>
            </div>

          </div>
        )}

        {/* 阶段 3：成品视频压制设置 */}
        {doVideo && (
          <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 space-y-4 animate-in fade-in duration-200">
            <div className="flex items-center gap-2">
              <Film className="w-4 h-4 text-blue-600 dark:text-blue-400" />
              <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                {t('dashboard.sectionVideo', '成品视频压制设置')}
              </h3>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
              {/* 字幕方式 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                  {t('dashboard.subtitleMethod', '字幕压制方式')}
                </label>
                <div className="relative">
                  <select
                    value={subtitleOutput}
                    onChange={(e) => setSubtitleOutput(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                  >
                    <option value="none">{t('dashboard.subNone', '无字幕')}</option>
                    <option value="soft">{t('dashboard.subSoft', '输出软字幕')}</option>
                    <option value="burn">{t('dashboard.subBurn', '将字幕编码到视频里面')}</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 字幕样式 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                  {t('dashboard.subtitleStyle', '字幕样式')}
                </label>
                <div className="relative">
                  <select
                    value={subtitleStyle}
                    onChange={(e) => setSubtitleStyle(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                  >
                    <option value="经典白字黑边">{english ? 'Classic white with black outline' : t('dashboard.styleClassic', '经典白字黑边')}</option>
                    <option value="Apple 毛玻璃底条">{english ? 'Apple frosted glass' : t('dashboard.styleGlass', 'Apple 毛玻璃底条')}</option>
                    <option value="现代鲜黄高对比">{english ? 'High-contrast yellow' : t('dashboard.styleYellow', '现代鲜黄高对比')}</option>
                    <option value="电影黑底居中">{english ? 'Cinematic centered black' : t('dashboard.styleCinema', '电影黑底居中')}</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 导出画质 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                  {t('dashboard.videoQuality', '导出画质')}
                </label>
                <div className="relative">
                  <select
                    value={videoQuality}
                    onChange={(e) => setVideoQuality(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                  >
                    <option value="原画质(推荐)">{t('dashboard.qualityOriginal', '原画质(推荐)')}</option>
                    <option value="4K 超清">{t('dashboard.quality4k', '4K 超清')}</option>
                    <option value="1080P 高清">{t('dashboard.quality1080p', '1080P 高清')}</option>
                    <option value="720P 标清">{t('dashboard.quality720p', '720P 标清')}</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>

              {/* 编码方式 */}
              <div>
                <label className="block text-[11px] font-semibold text-slate-600 dark:text-slate-400 mb-1.5">
                  {t('dashboard.encodeMethod', '编码方式')}
                </label>
                <div className="relative">
                  <select
                    value={encodeMethod}
                    onChange={(e) => setEncodeMethod(e.target.value)}
                    className="w-full h-10 bg-slate-50 hover:bg-slate-100/80 dark:bg-white/[0.06] dark:hover:bg-white/[0.09] border border-slate-200 dark:border-white/15 rounded-xl px-3 text-[13px] text-slate-800 dark:text-white appearance-none cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-colors"
                  >
                    <option value="默认(推荐)">{t('dashboard.codecDefault', '默认(推荐)')}</option>
                    <option value="H.264">H.264</option>
                    <option value="H.265">H.265</option>
                    <option value="AV1">AV1</option>
                  </select>
                  <ChevronDown className="w-4 h-4 text-slate-400 absolute right-3 top-3 pointer-events-none" />
                </div>
              </div>
            </div>

            <div className="text-xs text-slate-500 dark:text-slate-400 flex items-center justify-between">
              <span>{t('dashboard.mixTip', '配音完成后将自动降低原视频原声进行智能混音。')}</span>
            </div>
          </div>
        )}

        {/* 错误提示浮条 */}
        {errorMessage && (
          <div className="p-3.5 rounded-xl bg-rose-50 border border-rose-200 dark:bg-rose-500/10 dark:border-rose-500/30 flex items-center gap-2.5 text-xs text-rose-600 dark:text-rose-300">
            <AlertCircle className="w-4 h-4 text-rose-500 shrink-0" />
            <span>{errorMessage}</span>
          </div>
        )}
      </div>

      {/* 下半部：独立固定操作底栏（绝不遮挡主内容滚动，左右呼应） */}
      <div className="shrink-0 bg-white/95 dark:bg-[#1C1C1E]/95 backdrop-blur-2xl border-t border-slate-200/90 dark:border-[#2C2C2E] px-7 py-3.5 flex items-center justify-between z-20">
        <div className="flex items-center gap-2.5">
          <div className={`w-2.5 h-2.5 rounded-full ${hasFiles ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'}`} />
          <span className="text-xs font-semibold text-slate-700 dark:text-slate-200">
            {hasFiles
              ? t('dashboard.readyPrefix', {
                  count: selectedFiles.length,
                  type: getResourceTypeName(currentResourceType, t),
                  goal:
                    currentResourceType === 'text' || (doDub && !doTranslate && !doSub)
                      ? t('dashboard.goalAudiobook', 'AI朗读配音(小说/纯文本)')
                      : doVideo
                      ? t('dashboard.goalVideoProduct', '成品视频')
                      : doDub
                      ? t('dashboard.goalVoiceover', 'AI配音')
                      : doTranslate
                      ? t('dashboard.goalBilingualSub', '双语字幕')
                      : t('dashboard.goalTranscript', '提取字幕'),
                })
              : t('dashboard.noFileTip', '请拖入或选择同类型资源（视频、音频或纯文本）')}
          </span>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            type="button"
            onClick={() => setShowRecipeModal(true)}
            className="px-3.5 py-2 rounded-xl bg-slate-100 hover:bg-slate-200/80 dark:bg-white/10 dark:hover:bg-white/15 text-slate-700 dark:text-slate-200 text-xs font-semibold border border-slate-200 dark:border-white/10 transition-all flex items-center gap-1.5"
          >
            <BookmarkPlus className="w-3.5 h-3.5" />
            <span>{t('dashboard.saveRecipe', '保存为配方')}</span>
          </button>

          <button
            type="button"
            onClick={handleStartTask}
            disabled={loading}
            className={`px-5 py-2 rounded-xl text-xs font-bold text-white transition-all flex items-center gap-2 ${
              loading
                ? 'bg-blue-600/50 cursor-not-allowed'
                : 'bg-blue-600 hover:bg-blue-500 active:scale-95'
            }`}
          >
            {loading ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                <span>{t('dashboard.processing', '任务创建提交中...')}</span>
              </>
            ) : (
              <>
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>{hasFiles ? `${t('dashboard.startProcess', '立即开始处理')} (${selectedFiles.length})` : t('dashboard.browseAndStart', '选择文件并启动')}</span>
              </>
            )}
          </button>
        </div>
      </div>

      {asrWarning && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
          <div role="alertdialog" aria-modal="true" aria-labelledby="asr-warning-title" className="w-96 bg-white dark:bg-slate-900 border border-slate-200 dark:border-white/15 rounded-2xl p-5 space-y-4">
            <h3 id="asr-warning-title" className="text-sm font-bold text-slate-900 dark:text-white">{t('dashboard.asrNotReady', '语音识别尚未就绪')}</h3>
            <p className="text-sm text-slate-600 dark:text-slate-300">{asrWarning}</p>
            <div className="flex justify-end gap-2">
              <button type="button" onClick={() => setAsrWarning(null)} className="px-3 py-2 text-sm">{t('common.cancel', '取消')}</button>
              <button type="button" onClick={() => { setAsrWarning(null); onConfigureASR(); }} className="px-4 py-2 rounded-xl text-sm font-bold bg-blue-600 text-white">
                {asrEngine === 'whisper-cpp' ? t('dashboard.downloadAsrModel', '去下载模型') : t('dashboard.configureAsr', '前往语音识别设置')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 保存配方对话框 */}
      {showRecipeModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
          <div className="w-96 bg-white dark:bg-slate-900 border border-slate-200 dark:border-white/15 rounded-2xl p-5 space-y-4">
            <h3 className="text-sm font-bold text-slate-900 dark:text-white">
              {t('dashboard.recipeModalTitle', '保存当前配置为配方')}
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t('dashboard.recipeModalDesc', '配方将持久化保存至服务端数据库，方便下次一键直接调用。')}
            </p>
            {recipeSaveError && <p role="alert" className="text-xs text-rose-600">{recipeSaveError}</p>}
            <input
              type="text"
              placeholder={t('dashboard.recipeNamePlaceholder', '请输入配方名称，例如：科技视频精配方案')}
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
                {t('common.cancel', '取消')}
              </button>
              <button
                type="button"
                onClick={handleSaveRecipe}
                disabled={!recipeName.trim() || savingRecipe}
                className="px-4 py-1.5 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50 transition-colors"
              >
                {t('dashboard.saveRecipeBtn', '保存配方')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 配方保存成功 Toast */}
      {recipeSavedToast && (
        <div className="fixed top-6 right-6 z-50 bg-emerald-600 text-white text-xs px-4 py-2.5 rounded-xl border border-emerald-500/30 flex items-center gap-2 animate-in fade-in">
          <CheckCircle2 className="w-4 h-4" />
          <span>{t('dashboard.recipeSavedSuccess', '配方已成功保存至数据库，可在顶部模板中随时调用！')}</span>
        </div>
      )}
    </div>
  );
};
