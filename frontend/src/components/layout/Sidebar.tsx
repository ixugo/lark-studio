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
} from 'lucide-react';

export type TabKey =
  | 'dashboard'
  | 'board'
  | 'list'
  | 'merge'
  | 'glossary'
  | 'settings';

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
  const navItems: { key: TabKey; label: string; icon: React.ReactNode; badge?: number }[] = [
    { key: 'dashboard', label: '工作台', icon: <LayoutDashboard size={18} /> },
    { key: 'board', label: '任务看板', icon: <Kanban size={18} />, badge: runningTasksCount > 0 ? runningTasksCount : undefined },
    { key: 'list', label: '任务列表', icon: <ListTodo size={18} /> },
    { key: 'merge', label: '字幕合成', icon: <Layers size={18} /> },
    { key: 'glossary', label: '术语词库', icon: <BookOpen size={18} /> },
    { key: 'settings', label: '全局设置', icon: <Settings size={18} /> },
  ];

  return (
    <aside className="w-56 h-screen flex flex-col justify-between border-r border-apple-border dark:border-apple-darkBorder bg-apple-sidebar dark:bg-apple-darkSidebar px-3 py-4 transition-colors">
      <div className="space-y-6">
        {/* 顶部标题区（预留 macOS 红黄绿交通灯安全高度与拖拽区） */}
        <div className="pt-8 px-2 flex items-center justify-between" style={{ WebkitAppRegion: 'drag' } as React.CSSProperties}>
          <div className="flex items-center space-x-2.5" style={{ WebkitAppRegion: 'no-drag' } as React.CSSProperties}>
            <div className="w-7 h-7 rounded-lg bg-apple-accent text-white flex items-center justify-center font-bold text-sm shadow-sm">
              V
            </div>
            <div>
              <h1 className="text-sm font-semibold tracking-tight text-apple-text dark:text-apple-darkText">
                VDub Studio
              </h1>
              <p className="text-[10px] text-apple-muted">Wails 3 桌面端</p>
            </div>
          </div>
        </div>

        {/* 导航菜单 */}
        <nav className="space-y-1">
          {navItems.map((item) => {
            const active = currentTab === item.key;
            return (
              <button
                key={item.key}
                onClick={() => onTabChange(item.key)}
                className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-xs font-medium transition-all ${
                  active
                    ? 'bg-apple-accent text-white shadow-sm'
                    : 'text-apple-text dark:text-apple-darkText hover:bg-black/5 dark:hover:bg-white/5'
                }`}
              >
                <div className="flex items-center space-x-2.5">
                  <span className={active ? 'text-white' : 'text-apple-muted'}>{item.icon}</span>
                  <span>{item.label}</span>
                </div>
                {item.badge !== undefined && (
                  <span
                    className={`text-[10px] px-1.5 py-0.2 rounded-full font-bold ${
                      active ? 'bg-white text-apple-accent' : 'bg-apple-accent text-white'
                    }`}
                  >
                    {item.badge}
                  </span>
                )}
              </button>
            );
          })}
        </nav>
      </div>

      {/* 底部控制区 */}
      <div className="pt-3 border-t border-apple-border dark:border-apple-darkBorder flex items-center justify-between px-2">
        <button
          onClick={onToggleTheme}
          className="p-1.5 rounded-lg text-apple-muted hover:text-apple-text dark:hover:text-apple-darkText hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
          title={isDark ? '切换浅色模式' : '切换深色模式'}
        >
          {isDark ? <Sun size={16} /> : <Moon size={16} />}
        </button>
        <span className="text-[11px] font-mono text-apple-muted">v1.0.0</span>
      </div>
    </aside>
  );
};
