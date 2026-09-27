import React from 'react';
import {
  LayoutDashboard,
  Kanban,
  ListTodo,
  Layers,
  BookOpen,
  Settings,
  Sun,
  Moon,
  Languages,
  Volume2,
  Cpu,
  Globe,
  Info,
} from 'lucide-react';
import { useTranslation } from '../../i18n';
import { api } from '../../lib/api';

export type TabKey =
  | 'dashboard'
  | 'board'
  | 'list'
  | 'merge'
  | 'translation-engine'
  | 'tts-engine'
  | 'asr-engine'
  | 'glossary'
  | 'settings'
  | 'about';

interface SidebarProps {
  currentTab: TabKey;
  onTabChange: (tab: TabKey) => void;
  isDark: boolean;
  onToggleTheme: () => void;
  runningTasksCount: number;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onTabChange,
  isDark,
  onToggleTheme,
  runningTasksCount,
}) => {
  const { locale, toggleLocale, t } = useTranslation();
  const [appVersion, setAppVersion] = React.useState('0.1.0');

  React.useEffect(() => {
    api.getAppInfo().then((info) => {
      if (info?.build_version) {
        setAppVersion(info.build_version);
      }
    }).catch(() => {});
  }, []);

  const sections: {
    title: string;
    items: { key: TabKey; label: string; icon: React.ReactNode; badge?: number }[];
  }[] = [
    {
      title: t('sidebar.creationCenter'),
      items: [
        { key: 'dashboard', label: t('sidebar.workbench'), icon: <LayoutDashboard size={17} /> },
        { key: 'board', label: t('sidebar.taskBoard'), icon: <Kanban size={17} />, badge: runningTasksCount > 0 ? runningTasksCount : undefined },
        { key: 'merge', label: t('sidebar.subtitleMerge'), icon: <Layers size={17} /> },
      ],
    },
    {
      title: t('sidebar.engineCenter'),
      items: [
        { key: 'translation-engine', label: t('sidebar.translationEngine'), icon: <Languages size={17} /> },
        { key: 'tts-engine', label: t('sidebar.ttsEngine'), icon: <Volume2 size={17} /> },
        { key: 'asr-engine', label: t('sidebar.asrEngine'), icon: <Cpu size={17} /> },
      ],
    },
    {
      title: '系统与管理',
      items: [
        { key: 'glossary', label: t('sidebar.glossary'), icon: <BookOpen size={17} /> },
        { key: 'settings', label: t('sidebar.settings'), icon: <Settings size={17} /> },
        { key: 'about', label: t('sidebar.aboutSystem', '关于系统'), icon: <Info size={17} /> },
      ],
    },
  ];

  return (
    <aside className="w-56 h-screen flex flex-col justify-between border-r border-slate-200/80 dark:border-white/10 bg-white/70 dark:bg-[#18181B]/70 backdrop-blur-2xl px-3 py-4 transition-colors select-none">
      <div className="space-y-5">
        {/* 顶部标题区（预留 macOS 红黄绿交通灯安全高度与拖拽区） */}
        <div className="pt-7 px-2 flex items-center justify-between wails-drag">
          <div className="flex items-center space-x-2.5 wails-no-drag">
            <div className="w-7 h-7 rounded-lg overflow-hidden flex items-center justify-center border border-slate-200 dark:border-white/10">
              <img src="/lark-logo.svg" alt={t('common.appTitle')} className="w-full h-full object-contain" />
            </div>
            <div>
              <h1 className="text-sm font-semibold tracking-tight text-slate-800 dark:text-white">
                {t('common.appTitle')}
              </h1>
              <p className="text-[10px] text-slate-400">智能翻译与配音</p>
            </div>
          </div>
        </div>

        {/* 导航菜单分组 */}
        <nav className="space-y-4">
          {sections.map((sec, idx) => (
            <div key={idx} className="space-y-1">
              <div className="px-3 text-[10px] font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wider">
                {sec.title}
              </div>
              {sec.items.map((item) => {
                const active = currentTab === item.key;
                return (
                  <button
                    key={item.key}
                    type="button"
                    onClick={() => onTabChange(item.key)}
                    className={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition-all active:scale-[0.98] ${
                      active
                        ? 'bg-blue-600 text-white font-semibold'
                        : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-white/[0.06]'
                    }`}
                  >
                    <div className="flex items-center space-x-2.5">
                      <span className={active ? 'text-white' : 'text-slate-500 dark:text-slate-400'}>
                        {item.icon}
                      </span>
                      <span>{item.label}</span>
                    </div>
                    {item.badge !== undefined && (
                      <span
                        className={`text-[10px] px-1.5 py-0.2 rounded-full font-bold ${
                          active ? 'bg-white text-blue-600' : 'bg-blue-600 text-white'
                        }`}
                      >
                        {item.badge}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
          ))}
        </nav>
      </div>

      {/* 底部控制区 */}
      <div className="pt-3 border-t border-slate-200/80 dark:border-white/10 flex items-center justify-between px-2">
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            onClick={onToggleTheme}
            className="p-1.5 rounded-lg text-slate-500 hover:text-slate-800 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-white/5 transition-colors active:scale-95"
            title={isDark ? t('sidebar.lightMode') : t('sidebar.darkMode')}
          >
            {isDark ? <Sun size={15} /> : <Moon size={15} />}
          </button>
          <button
            type="button"
            onClick={toggleLocale}
            className="flex items-center gap-1 px-2 py-1 rounded-lg text-[11px] font-semibold text-slate-600 dark:text-slate-300 hover:text-blue-600 dark:hover:text-blue-400 hover:bg-slate-100 dark:hover:bg-white/5 transition-colors active:scale-95 border border-slate-200/60 dark:border-white/10"
            title={t('sidebar.switchLanguage')}
          >
            <Globe size={13} className="text-slate-400" />
            <span>{locale === 'zh-CN' ? '中' : 'EN'}</span>
          </button>
        </div>
        <span className="text-[11px] font-mono text-slate-400">v{appVersion.replace(/^v/, '')}</span>
      </div>
    </aside>
  );
};
