import React, { useEffect, useState, useCallback } from 'react';
import { Search, RotateCcw, Trash2, FolderOpen, Play, Pause } from 'lucide-react';
import { api } from '../lib/api';
import { Task } from '../types';

export const TaskListView: React.FC = () => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(true);

  const loadTasks = useCallback(async () => {
    try {
      const data = await api.listTasks();
      setTasks(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTasks();
  }, [loadTasks]);

  const filteredTasks = tasks.filter((t) =>
    (t.input_path || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
    (t.id || '').toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="h-screen flex flex-col overflow-y-auto px-8 py-6">
      <div className="max-w-5xl w-full mx-auto space-y-5">
        {/* 顶部搜索与操作 */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold tracking-tight text-apple-text dark:text-apple-darkText">
              任务总表
            </h2>
            <p className="text-xs text-apple-muted mt-1">清单化查看与批量管理全部音视频处理任务</p>
          </div>

          <div className="flex items-center space-x-3">
            <div className="relative">
              <Search size={14} className="absolute left-2.5 top-2.5 text-apple-muted" />
              <input
                type="text"
                placeholder="搜索任务或文件名..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-8 pr-3 py-1.5 text-xs bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-lg focus:outline-none focus:ring-1 focus:ring-apple-accent text-apple-text dark:text-apple-darkText w-56"
              />
            </div>
            <button
              onClick={loadTasks}
              className="p-1.5 rounded-lg border border-apple-border dark:border-apple-darkBorder hover:bg-black/5 dark:hover:bg-white/5 text-apple-muted"
            >
              <RotateCcw size={14} />
            </button>
          </div>
        </div>

        {/* 列表表格 */}
        <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl overflow-hidden">
          <table className="w-full text-left text-xs">
            <thead className="bg-black/5 dark:bg-white/5 text-apple-muted border-b border-apple-border dark:border-apple-darkBorder">
              <tr>
                <th className="py-2.5 px-4 font-semibold">文件名 / 路径</th>
                <th className="py-2.5 px-4 font-semibold">模式</th>
                <th className="py-2.5 px-4 font-semibold">状态</th>
                <th className="py-2.5 px-4 font-semibold">进度</th>
                <th className="py-2.5 px-4 font-semibold">翻译服务</th>
                <th className="py-2.5 px-4 font-semibold text-right">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-apple-border dark:divide-apple-darkBorder text-apple-text dark:text-apple-darkText">
              {loading ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-apple-muted">
                    正在加载...
                  </td>
                </tr>
              ) : filteredTasks.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-8 text-center text-apple-muted">
                    无匹配任务
                  </td>
                </tr>
              ) : (
                filteredTasks.map((t) => (
                  <tr key={t.id} className="hover:bg-black/5 dark:hover:bg-white/5 transition-colors">
                    <td className="py-3 px-4 max-w-xs truncate font-mono">
                      {t.input_path ? t.input_path.split(/[\\/]/).pop() : t.id}
                    </td>
                    <td className="py-3 px-4">
                      {t.mode === 1 ? '仅字幕' : t.mode === 2 ? '双语翻译' : '翻译配音'}
                    </td>
                    <td className="py-3 px-4">
                      <span className="font-semibold text-[11px]">
                        {t.status === 1 ? '进行中' : t.status === 2 ? '已暂停' : t.status === 3 ? '已完成' : t.status === 4 ? '失败' : '排队中'}
                      </span>
                    </td>
                    <td className="py-3 px-4 font-mono font-bold text-apple-accent">
                      {t.progress}%
                    </td>
                    <td className="py-3 px-4 text-apple-muted">
                      {t.translator}
                    </td>
                    <td className="py-3 px-4 text-right space-x-2">
                      <button
                        onClick={() => api.openInFileManager(t.output_dir)}
                        className="text-apple-muted hover:text-apple-text dark:hover:text-apple-darkText"
                        title="打开目录"
                      >
                        <FolderOpen size={14} />
                      </button>
                      {t.status === 1 ? (
                        <button
                          onClick={async () => {
                            await api.pauseTask(t.id);
                            loadTasks();
                          }}
                          className="text-amber-500 hover:opacity-75"
                          title="暂停"
                        >
                          <Pause size={14} />
                        </button>
                      ) : t.status === 2 ? (
                        <button
                          onClick={async () => {
                            await api.resumeTask(t.id);
                            loadTasks();
                          }}
                          className="text-apple-accent hover:opacity-75"
                          title="继续"
                        >
                          <Play size={14} />
                        </button>
                      ) : null}
                      <button
                        onClick={async () => {
                          if (confirm('确认删除？')) {
                            await api.deleteTask(t.id);
                            loadTasks();
                          }
                        }}
                        className="text-red-500 hover:opacity-75"
                        title="删除"
                      >
                        <Trash2 size={14} />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
