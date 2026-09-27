import React, { useState } from 'react';
import {
  Film,
  FileText,
  Layers,
  Upload,
  CheckCircle2,
  AlertCircle,
  Play,
} from 'lucide-react';
import { api } from '../lib/api';
import { useTranslation } from '../i18n';

interface SubtitleMergeViewProps {
  onTaskCreated?: () => void;
}

export const SubtitleMergeView: React.FC<SubtitleMergeViewProps> = ({ onTaskCreated }) => {
  const { t } = useTranslation();
  const [videoPath, setVideoPath] = useState<string>('');
  const [primarySubPath, setPrimarySubPath] = useState<string>('');
  const [secondarySubPath, setSecondarySubPath] = useState<string>('');
  const [isBilingual, setIsBilingual] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successToast, setSuccessToast] = useState<string | null>(null);

  // 选择视频文件
  const handlePickVideo = async () => {
    try {
      const files = await api.pickFiles({
        title: t('subtitleMerge.selectVideo', '选择待合成的视频文件'),
        multiple: false,
        extensions: ['mp4', 'mkv', 'mov', 'avi', 'webm', 'flv'],
      });
      if (files && files.length > 0) {
        setVideoPath(files[0]);
        setErrorMessage(null);
      }
    } catch (err) {
      console.error('选择视频失败:', err);
    }
  };

  // 选择主字幕文件
  const handlePickPrimarySub = async () => {
    try {
      const files = await api.pickFiles({
        title: t('subtitleMerge.selectSub', '选择字幕文件 (.srt / .vtt / .ass)'),
        multiple: false,
        extensions: ['srt', 'vtt', 'ass'],
      });
      if (files && files.length > 0) {
        setPrimarySubPath(files[0]);
        setErrorMessage(null);
      }
    } catch (err) {
      console.error('选择字幕失败:', err);
    }
  };

  // 选择副字幕文件（双语对照）
  const handlePickSecondarySub = async () => {
    try {
      const files = await api.pickFiles({
        title: t('subtitleMerge.secondarySub', '选择第二字幕文件 (原文字幕 .srt / .vtt / .ass)'),
        multiple: false,
        extensions: ['srt', 'vtt', 'ass'],
      });
      if (files && files.length > 0) {
        setSecondarySubPath(files[0]);
        setErrorMessage(null);
      }
    } catch (err) {
      console.error('选择副字幕失败:', err);
    }
  };

  // 开始合成
  const handleStartMerge = async () => {
    if (!videoPath.trim()) {
      setErrorMessage(t('subtitleMerge.needSelectFiles', '请先选择待压制合成的视频文件'));
      return;
    }
    if (!primarySubPath.trim()) {
      setErrorMessage(t('subtitleMerge.needSelectFiles', '请先选择需要压制的字幕文件'));
      return;
    }
    if (isBilingual && !secondarySubPath.trim()) {
      setErrorMessage(t('subtitleMerge.needSelectFiles', '已开启双语对照，请选择副字幕文件'));
      return;
    }

    setLoading(true);
    setErrorMessage(null);
    try {
      await api.mergeSubtitle({
        video_path: videoPath.trim(),
        primary_sub_path: primarySubPath.trim(),
        secondary_sub_path: isBilingual ? secondarySubPath.trim() : undefined,
        output_content: isBilingual ? 'bilingual' : 'source',
      });

      setSuccessToast(t('subtitleMerge.startMerge', '字幕合成任务已成功创建并提交流水线！'));
      setTimeout(() => {
        setSuccessToast(null);
        if (onTaskCreated) {
          onTaskCreated();
        }
      }, 1200);
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  const getFileName = (path: string) => path.split(/[\\/]/).pop() || path;

  return (
    <div
      className="h-screen flex flex-col justify-between overflow-y-auto px-8 py-6 select-none"
      data-file-drop-target="true"
    >
      <div className="max-w-4xl w-full mx-auto space-y-6 pt-4">
        {/* 标题区 */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-xl bg-blue-600 text-white flex items-center justify-center">
                <Layers className="w-4 h-4" />
              </div>
              <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
                {t('subtitleMerge.title', '独立字幕合成')}
              </h2>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1.5">
              {t('subtitleMerge.mainDesc', '将外部现成的字幕文件（.srt / .vtt / .ass）直接烧录压制进视频，快速渲染高画质成片')}
            </p>
          </div>
        </div>

        {/* 提示条 */}
        {successToast && (
          <div className="p-3.5 bg-emerald-500/10 border border-emerald-500/20 rounded-xl flex items-center space-x-2 text-xs text-emerald-600 dark:text-emerald-400">
            <CheckCircle2 size={16} />
            <span>{successToast}</span>
          </div>
        )}

        {errorMessage && (
          <div className="p-3.5 bg-rose-50 border border-rose-200 dark:bg-rose-500/10 dark:border-rose-500/30 rounded-xl flex items-center gap-2 text-xs text-rose-600 dark:text-rose-300">
            <AlertCircle size={16} />
            <span>{errorMessage}</span>
          </div>
        )}

        {/* 第一步：选择待合成视频 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="w-5 h-5 rounded-full bg-blue-600 text-white text-[11px] font-bold flex items-center justify-center">
                1
              </span>
              <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                {t('subtitleMerge.step1Title', '选择底板视频')}
              </h3>
            </div>
            {videoPath && (
              <button
                type="button"
                onClick={handlePickVideo}
                className="text-xs text-blue-600 hover:text-blue-500 font-medium"
              >
                {t('subtitleMerge.reselect', '重新选择')}
              </button>
            )}
          </div>

          {videoPath ? (
            <div className="p-3.5 bg-slate-50 dark:bg-white/[0.04] border border-slate-200 dark:border-white/10 rounded-xl flex items-center justify-between">
              <div className="flex items-center gap-3 truncate">
                <div className="w-8 h-8 rounded-lg bg-blue-600/10 text-blue-600 flex items-center justify-center shrink-0">
                  <Film className="w-4 h-4" />
                </div>
                <div className="truncate">
                  <div className="text-xs font-bold text-slate-800 dark:text-white truncate">
                    {getFileName(videoPath)}
                  </div>
                  <div className="text-[10px] text-slate-400 truncate mt-0.5 font-mono">
                    {videoPath}
                  </div>
                </div>
              </div>
              <span className="text-[11px] text-emerald-600 dark:text-emerald-400 font-semibold shrink-0 ml-2">
                {t('common.installed', '已就绪')}
              </span>
            </div>
          ) : (
            <div
              onClick={handlePickVideo}
              className="border-2 border-dashed border-slate-200 hover:border-blue-500 dark:border-white/10 dark:hover:border-blue-500/60 rounded-xl p-6 text-center cursor-pointer transition-colors bg-slate-50/50 hover:bg-blue-50/30 dark:bg-white/[0.02]"
            >
              <Upload className="w-6 h-6 text-slate-400 dark:text-slate-500 mx-auto mb-2" />
              <p className="text-xs font-bold text-slate-700 dark:text-slate-200">
                {t('subtitleMerge.step1Desc', '点击选择待合成的视频文件')}
              </p>
              <p className="text-[11px] text-slate-400 mt-1">
                {t('subtitleMerge.step1Format', '支持常见视频格式：MP4, MKV, MOV, AVI, WEBM, FLV')}
              </p>
            </div>
          )}
        </div>

        {/* 第二步：选择字幕文件 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="w-5 h-5 rounded-full bg-blue-600 text-white text-[11px] font-bold flex items-center justify-center">
                2
              </span>
              <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
                选择字幕文件
              </h3>
            </div>

            {/* 单字幕 / 双语对照 切换 */}
            <div className="flex items-center gap-1 bg-slate-100 dark:bg-white/10 p-0.5 rounded-xl border border-slate-200/80 dark:border-white/10 text-xs">
              <button
                type="button"
                onClick={() => setIsBilingual(false)}
                className={`px-3 py-1 rounded-lg font-medium transition-all ${
                  !isBilingual
                    ? 'bg-white dark:bg-[#2C2C2E] text-blue-600 dark:text-blue-400 font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-800'
                }`}
              >
                {t('subtitleMerge.singleSubTab', '单字幕压制')}
              </button>
              <button
                type="button"
                onClick={() => setIsBilingual(true)}
                className={`px-3 py-1 rounded-lg font-medium transition-all ${
                  isBilingual
                    ? 'bg-white dark:bg-[#2C2C2E] text-blue-600 dark:text-blue-400 font-semibold'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-800'
                }`}
              >
                {t('subtitleMerge.bilingualSubTab', '双语对照压制')}
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {/* 主字幕 */}
            <div className="space-y-2">
              <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block">
                {isBilingual ? t('subtitleMerge.primarySubBilingualLabel', '主要字幕 (显示在上层，如翻译译文)') : t('subtitleMerge.primarySubLabel', '字幕文件 (.srt / .vtt / .ass)')}
              </label>

              {primarySubPath ? (
                <div className="p-3 bg-slate-50 dark:bg-white/[0.04] border border-slate-200 dark:border-white/10 rounded-xl flex items-center justify-between">
                  <div className="flex items-center gap-2 truncate">
                    <FileText className="w-4 h-4 text-blue-600 shrink-0" />
                    <span className="text-xs font-bold text-slate-800 dark:text-white truncate">
                      {getFileName(primarySubPath)}
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={handlePickPrimarySub}
                    className="text-[11px] text-blue-600 hover:text-blue-500 font-medium shrink-0 ml-2"
                  >
                    {t('subtitleMerge.reselect', '重选')}
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  onClick={handlePickPrimarySub}
                  className="w-full h-14 border border-dashed border-slate-200 hover:border-blue-500 dark:border-white/15 rounded-xl flex items-center justify-center gap-2 text-xs font-semibold text-slate-600 hover:text-blue-600 dark:text-slate-300 transition-colors bg-slate-50/50 dark:bg-white/[0.02]"
                >
                  <Upload className="w-4 h-4" />
                  <span>{t('subtitleMerge.selectPrimaryBtn', '选择主字幕文件')}</span>
                </button>
              )}
            </div>

            {/* 副字幕 (仅双语模式启用) */}
            {isBilingual && (
              <div className="space-y-2 animate-in fade-in duration-150">
                <label className="text-[11px] font-semibold text-slate-600 dark:text-slate-400 block">
                  {t('subtitleMerge.secondarySubLabel', '次要字幕 (显示在下层，如原文原声)')}
                </label>

                {secondarySubPath ? (
                  <div className="p-3 bg-slate-50 dark:bg-white/[0.04] border border-slate-200 dark:border-white/10 rounded-xl flex items-center justify-between">
                    <div className="flex items-center gap-2 truncate">
                      <FileText className="w-4 h-4 text-slate-500 shrink-0" />
                      <span className="text-xs font-bold text-slate-800 dark:text-white truncate">
                        {getFileName(secondarySubPath)}
                      </span>
                    </div>
                    <button
                      type="button"
                      onClick={handlePickSecondarySub}
                      className="text-[11px] text-blue-600 hover:text-blue-500 font-medium shrink-0 ml-2"
                    >
                      {t('subtitleMerge.reselect', '重选')}
                    </button>
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={handlePickSecondarySub}
                    className="w-full h-14 border border-dashed border-slate-200 hover:border-blue-500 dark:border-white/15 rounded-xl flex items-center justify-center gap-2 text-xs font-semibold text-slate-600 hover:text-blue-600 dark:text-slate-300 transition-colors bg-slate-50/50 dark:bg-white/[0.02]"
                  >
                    <Upload className="w-4 h-4" />
                    <span>{t('subtitleMerge.selectSecondaryBtn', '选择副字幕文件')}</span>
                  </button>
                )}
              </div>
            )}
          </div>
        </div>

        {/* 第三步：合成配置与输出 */}
        <div className="bg-white dark:bg-[#1C1C1E] rounded-2xl border border-slate-200/90 dark:border-[#2C2C2E] p-5 space-y-4">
          <div className="flex items-center gap-2">
            <span className="w-5 h-5 rounded-full bg-blue-600 text-white text-[11px] font-bold flex items-center justify-center">
              3
            </span>
            <h3 className="text-xs font-bold text-slate-800 dark:text-slate-200 uppercase tracking-wider">
              {t('subtitleMerge.step3Title', '输出目录与压制设置')}
            </h3>
          </div>

          <div className="space-y-3">
            <div className="p-3.5 bg-slate-50 dark:bg-white/[0.02] border border-slate-200/80 dark:border-white/5 rounded-xl text-xs space-y-1.5">
              <div className="flex items-center gap-2 font-semibold text-slate-700 dark:text-slate-200">
                <CheckCircle2 className="w-4 h-4 text-emerald-500 shrink-0" />
                <span>{t('subtitleMerge.step3Box1Title', '自动归档至统一任务目录')}</span>
              </div>
              <p className="text-slate-500 dark:text-slate-400 text-[11px] leading-relaxed">
                {t('subtitleMerge.step3Box1Desc', '成片将固定生成至任务工作区目录，避免路径混乱与文件覆盖。压制完成后可在【任务看板】中随时双击或点击直接在系统访达/资源管理器中定位查看。')}
              </p>
            </div>

          </div>
        </div>
      </div>

      {/* 底部动作悬浮底栏 (圆角胶囊形态) */}
      <div className="sticky bottom-4 mx-auto max-w-4xl w-full z-30 px-4 mt-6">
        <div className="bg-white/90 dark:bg-[#202024]/90 backdrop-blur-2xl border border-slate-200/90 dark:border-white/10 rounded-2xl px-6 py-3 flex items-center justify-between">
          <div className="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-2">
            <span>{t('subtitleMerge.readyStatus', '准备状态:')}</span>
            {videoPath && primarySubPath ? (
              <span className="text-emerald-600 dark:text-emerald-400 font-semibold flex items-center gap-1">
                <CheckCircle2 className="w-3.5 h-3.5" /> {t('subtitleMerge.readyToMerge', '素材已齐备，可立即合成')}
              </span>
            ) : (
              <span className="text-amber-500 font-medium">{t('subtitleMerge.needSelectFiles', '请先选定视频及对应字幕')}</span>
            )}
          </div>

          <button
            type="button"
            onClick={handleStartMerge}
            disabled={loading || !videoPath || !primarySubPath}
            className={`px-6 py-2.5 rounded-xl text-xs font-bold text-white transition-all flex items-center gap-2 active:scale-[0.98] ${
              loading || !videoPath || !primarySubPath
                ? 'bg-slate-300 dark:bg-white/20 cursor-not-allowed text-slate-500'
                : 'bg-blue-600 hover:bg-blue-500'
            }`}
          >
            {loading ? (
              <span>{t('subtitleMerge.merging', '正在压制合成...')}</span>
            ) : (
              <>
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>{t('subtitleMerge.startMerge', '开始压制合成')}</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
