import React from 'react';
import { Sparkles, Layers, Mic, FileEdit } from 'lucide-react';

interface PlaceholderViewProps {
  type: 'proofread' | 'merge' | 'dubbing';
}

export const PlaceholderView: React.FC<PlaceholderViewProps> = ({ type }) => {
  const meta = {
    proofread: {
      title: '字幕校对与时间轴协同',
      desc: '支持双语对照、分秒级时间轴微调与音视频播放器跳帧联动编辑',
      icon: <FileEdit size={32} className="text-apple-accent" />,
    },
    merge: {
      title: '成片字幕渲染与合成',
      desc: 'GPU/CPU 硬件加速将 ASS/SRT 字体样式烧录入成片，或直接压入独立音轨容器',
      icon: <Layers size={32} className="text-indigo-500" />,
    },
    dubbing: {
      title: '智能配音与声音克隆',
      desc: '零样本声音克隆、跨语种声线保持与情绪音调微调（结合 MuseTalk 口型同步）',
      icon: <Mic size={32} className="text-emerald-500" />,
    },
  }[type];

  return (
    <div className="h-screen flex items-center justify-center px-8">
      <div className="max-w-md text-center space-y-4">
        <div className="w-16 h-16 rounded-2xl bg-black/5 dark:bg-white/5 border border-slate-200/60 dark:border-white/10 flex items-center justify-center mx-auto">
          {meta.icon}
        </div>
        <div>
          <h2 className="text-base font-bold text-apple-text dark:text-apple-darkText">
            {meta.title}
          </h2>
          <p className="text-xs text-apple-muted mt-1 leading-relaxed">
            {meta.desc}
          </p>
        </div>
        <div className="inline-flex items-center space-x-1.5 px-3 py-1 rounded-full bg-apple-accent/10 text-apple-accent text-[11px] font-semibold">
          <Sparkles size={12} />
          <span>流水线后台全自动化执行中</span>
        </div>
      </div>
    </div>
  );
};
