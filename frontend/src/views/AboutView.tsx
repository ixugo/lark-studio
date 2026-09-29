import React, { useEffect, useState } from 'react';
import {
  ShieldCheck,
  Layers,
  Database,
  Github,
} from 'lucide-react';
import { api } from '../lib/api';
import { AppInfo } from '../types';
import { useTranslation } from '../i18n';

export const AboutView: React.FC = () => {
  const { t } = useTranslation();
  const [appInfo, setAppInfo] = useState<AppInfo>({
    app_name: 'Lark Studio',
    build_version: '0.1.0',
    platform: 'darwin',
    arch: 'arm64',
  });

  useEffect(() => {
    api.getAppInfo().then(setAppInfo).catch(console.error);
  }, []);

  return (
    <div className="h-full flex flex-col p-6 space-y-6 overflow-y-auto max-w-4xl mx-auto animate-in fade-in select-text">
      {/* 头部产品主卡片 */}
      <div className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-white/10 rounded-2xl p-6 sm:p-8 flex flex-col sm:flex-row items-center sm:items-start gap-6 shadow-sm">
        <div className="w-20 h-20 rounded-2xl overflow-hidden shrink-0 border border-slate-200 dark:border-white/10 shadow-md">
          <img src="/lark-logo.webp" alt={t('about.appName')} className="w-full h-full object-contain p-3" />
        </div>
        <div className="space-y-2 text-center sm:text-left flex-1">
          <div className="flex flex-wrap items-center justify-center sm:justify-start gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-white">
              {t('about.appName', 'Lark Studio')}
            </h1>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-mono font-bold bg-blue-50 text-blue-600 dark:bg-blue-500/15 dark:text-blue-400 border border-blue-200 dark:border-blue-500/30">
              v{appInfo.build_version.replace(/^v/, '')}
            </span>
          </div>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            {t('about.tagline', '本地优先 · 智能音视频全流程译制与配音工作台')}
          </p>
          <div className="flex flex-wrap items-center justify-center sm:justify-start gap-4 pt-2 text-xs text-slate-400 font-mono">
            <span>{t('about.platform', '平台')}: {appInfo.platform} ({appInfo.arch})</span>
            <span>•</span>
            <a
              href="https://github.com/ixugo/lark-studio"
              onClick={(event) => {
                event.preventDefault();
                api.openProjectWebsite().catch(console.error);
              }}
              target="_blank"
              className="inline-flex items-center gap-1.5 text-blue-600 hover:text-blue-500 dark:text-blue-400"
            >
              <Github size={13} />
              {t('about.github', 'GitHub 项目主页')}
            </a>
          </div>
        </div>
      </div>

      {/* 核心产品特性 */}
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <h2 className="text-xs font-bold text-slate-700 dark:text-slate-300 uppercase tracking-wider">
            {t('about.featuresTitle', '核心特性')}
          </h2>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3.5">
          <div className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-white/10 rounded-xl p-4 space-y-2">
            <div className="w-7 h-7 rounded-lg bg-blue-50 dark:bg-blue-500/15 text-blue-600 dark:text-blue-400 flex items-center justify-center">
              <Layers size={16} />
            </div>
            <h3 className="text-xs font-bold text-slate-800 dark:text-white">
              {t('about.feature1Title', '四大领域阶段闭环')}
            </h3>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 leading-relaxed">
              {t('about.feature1Desc', '听写转录、智能翻译、AI 配音与压制合成，无缝串联或自由组合。')}
            </p>
          </div>

          <div className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-white/10 rounded-xl p-4 space-y-2">
            <div className="w-7 h-7 rounded-lg bg-emerald-50 dark:bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 flex items-center justify-center">
              <ShieldCheck size={16} />
            </div>
            <h3 className="text-xs font-bold text-slate-800 dark:text-white">
              {t('about.feature2Title', '零损耗秒级合成')}
            </h3>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 leading-relaxed">
              {t('about.feature2Desc', '针对软字幕采用流拷贝复制算法，无需重编码，秒级完成封装出片。')}
            </p>
          </div>

          <div className="bg-white dark:bg-[#1C1C1E] border border-slate-200/90 dark:border-white/10 rounded-xl p-4 space-y-2">
            <div className="w-7 h-7 rounded-lg bg-purple-50 dark:bg-purple-500/15 text-purple-600 dark:text-purple-400 flex items-center justify-center">
              <Database size={16} />
            </div>
            <h3 className="text-xs font-bold text-slate-800 dark:text-white">
              {t('about.feature3Title', '本地持久化与数据安全')}
            </h3>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 leading-relaxed">
              {t('about.feature3Desc', '任务记录、配方数据与专业术语库均保存在本地 SQLite 数据库中，隐私安全可靠。')}
            </p>
          </div>
        </div>
      </div>

    </div>
  );
};
