import React, { useState, useEffect } from 'react';
import { Sidebar, TabKey } from './components/layout/Sidebar';
import { DashboardView } from './views/DashboardView';
import { TaskBoardView } from './views/TaskBoardView';
import { TaskListView } from './views/TaskListView';
import { SettingsView } from './views/SettingsView';
import { GlossaryView } from './views/GlossaryView';
import { PlaceholderView } from './views/PlaceholderView';
import { api } from './lib/api';

export const App: React.FC = () => {
  const [currentTab, setCurrentTab] = useState<TabKey>('dashboard');
  const [isDark, setIsDark] = useState(false);
  const [runningCount, setRunningCount] = useState(0);

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
        return <DashboardView onTaskCreated={() => setCurrentTab('board')} />;
      case 'board':
        return <TaskBoardView />;
      case 'list':
        return <TaskListView />;
      case 'proofread':
        return <PlaceholderView type="proofread" />;
      case 'merge':
        return <PlaceholderView type="merge" />;
      case 'dubbing':
        return <PlaceholderView type="dubbing" />;
      case 'glossary':
        return <GlossaryView />;
      case 'settings':
        return <SettingsView />;
      default:
        return <DashboardView onTaskCreated={() => setCurrentTab('board')} />;
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-apple-bg dark:bg-apple-darkBg text-apple-text dark:text-apple-darkText select-none">
      <Sidebar
        currentTab={currentTab}
        onTabChange={setCurrentTab}
        isDark={isDark}
        onToggleTheme={toggleTheme}
        runningTasksCount={runningCount}
      />
      <main className="flex-1 h-screen overflow-hidden bg-apple-bg dark:bg-apple-darkBg">
        {renderContent()}
      </main>
    </div>
  );
};
