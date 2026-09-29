import React, { useState, useEffect, useRef } from 'react';
import { Sidebar, TabKey } from './components/layout/Sidebar';
import { DashboardView } from './views/DashboardView';
import { TaskBoardView } from './views/TaskBoardView';
import { SettingsView } from './views/SettingsView';
import { GlossaryView } from './views/GlossaryView';
import { SubtitleMergeView } from './views/SubtitleMergeView';
import { TranslationEngineView } from './views/TranslationEngineView';
import { TtsEngineView } from './views/TtsEngineView';
import { AsrEngineView } from './views/AsrEngineView';
import { AboutView } from './views/AboutView';
import { api } from './lib/api';
import { LanguageProvider, useTranslation } from './i18n';
import { UpdateDialog } from './components/UpdateDialog';
import { UpdateInfo } from './types';

export const AppContent: React.FC = () => {
  const { t } = useTranslation();
  const [update, setUpdate] = useState<UpdateInfo | null>(null);
  const [checkingUpdates, setCheckingUpdates] = useState(false);
  const [updateFeedback, setUpdateFeedback] = useState('');
  const startupCheck = useRef<Promise<UpdateInfo> | null>(null);
  const checkPending = useRef(false);
  const [currentTab, setCurrentTab] = useState<TabKey>('dashboard');
  const [isDark, setIsDark] = useState(false);
  const [runningCount, setRunningCount] = useState(0);

  useEffect(() => {
    if (!api.isWailsEnvironment()) return;
    // StrictMode 会重放 effect；复用同一次检查，由当前 effect 接收结果。
    startupCheck.current ??= api.checkForUpdates(false);
    let active = true;
    startupCheck.current.then((info) => {
      if (active && info.available) setUpdate(info);
    }).catch((err) => { if (active) console.warn('Startup update check failed', err); });
    return () => { active = false; };
  }, []);

  const checkUpdates = async () => {
    if (checkPending.current) return;
    if (!api.isWailsEnvironment()) { setUpdateFeedback(t('update.desktopOnly')); return; }
    checkPending.current = true;
    setCheckingUpdates(true);
    setUpdateFeedback('');
    try {
      const info = await api.checkForUpdates(true);
      if (info.available) setUpdate(info);
      else setUpdateFeedback(t('update.current'));
    } catch (err) {
      setUpdateFeedback(`${t('update.failed')}: ${String(err)}`);
    } finally {
      checkPending.current = false;
      setCheckingUpdates(false);
    }
  };

  // 初始化深浅色模式（默认跟随系统）
  useEffect(() => {
    const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
    setIsDark(prefersDark);
    if (prefersDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, []);

  const toggleTheme = () => {
    setIsDark((prev) => {
      const next = !prev;
      if (next) {
        document.documentElement.classList.add('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
      return next;
    });
  };

  // 轮询运行中任务数
  useEffect(() => {
    const checkRunning = async () => {
      try {
        const tasks = await api.listTasks();
        const running = tasks.filter((t) => t.status === 1).length;
        setRunningCount(running);
      } catch {
        // ignore
      }
    };
    checkRunning();
    const timer = setInterval(checkRunning, 4000);
    return () => clearInterval(timer);
  }, []);

  const renderContent = () => {
    switch (currentTab) {
      case 'dashboard':
        return null;
      case 'board':
        return <TaskBoardView />;
      case 'merge':
        return <SubtitleMergeView onTaskCreated={() => setCurrentTab('board')} />;
      case 'translation-engine':
        return <TranslationEngineView />;
      case 'tts-engine':
        return <TtsEngineView />;
      case 'asr-engine':
        return <AsrEngineView />;
      case 'glossary':
        return <GlossaryView />;
      case 'settings':
        return <SettingsView onNavigate={(tab) => setCurrentTab(tab as TabKey)} />;
      case 'about':
        return <AboutView onCheckUpdates={checkUpdates} checkingUpdates={checkingUpdates} updateFeedback={updateFeedback} />;
      default:
        return null;
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#F5F5F7] dark:bg-[#121215] text-[#1D1D1F] dark:text-[#F5F5F7] select-none">
      {update && <UpdateDialog update={update} onClose={() => setUpdate(null)} />}
      <Sidebar
        currentTab={currentTab}
        onTabChange={setCurrentTab}
        isDark={isDark}
        onToggleTheme={toggleTheme}
        runningTasksCount={runningCount}
      />
      <main className="flex-1 h-screen overflow-hidden bg-[#F5F5F7] dark:bg-[#121215] flex flex-col">
        {/* macOS 原生窗口拖拽区 */}
        <div className="h-7 w-full shrink-0 wails-drag" />
        <div className="flex-1 overflow-hidden">
          <div className={currentTab === 'dashboard' ? 'h-full' : 'hidden'}>
            <DashboardView
              active={currentTab === 'dashboard'}
              onTaskCreated={() => setCurrentTab('board')}
              onConfigureASR={() => setCurrentTab('asr-engine')}
            />
          </div>
          {currentTab !== 'dashboard' && renderContent()}
        </div>
      </main>
    </div>
  );
};

export const App: React.FC = () => {
  return (
    <LanguageProvider>
      <AppContent />
    </LanguageProvider>
  );
};
