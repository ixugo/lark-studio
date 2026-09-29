import React, { useEffect, useState, useCallback } from 'react';
import { Plus, Trash2, Search } from 'lucide-react';
import { api } from '../lib/api';
import { RefreshButton } from '../components/RefreshButton';
import { Term } from '../types';
import { useTranslation } from '../i18n';

export const GlossaryView: React.FC = () => {
  const { t, locale } = useTranslation();
  const english = locale === 'en-US';
  const [terms, setTerms] = useState<Term[]>([]);
  const [source, setSource] = useState('');
  const [target, setTarget] = useState('');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [deletingId, setDeletingId] = useState<number | null>(null);

  const loadTerms = useCallback(async () => {
    try {
      const data = await api.listTerms();
      setTerms(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTerms();
  }, [loadTerms]);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!source.trim() || !target.trim()) return;
    try {
      await api.saveTerm(source.trim(), target.trim());
      setSource('');
      setTarget('');
      loadTerms();
    } catch (err) {
      console.error('Failed to add term:', err);
    }
  };

  const handleDelete = async (id: number) => {
    // 乐观立即过滤
    setTerms((prev) => prev.filter((t) => t.id !== id));
    setDeletingId(null);
    try {
      await api.deleteTerm(id);
    } catch (err) {
      console.error('Failed to delete term:', err);
    } finally {
      loadTerms();
    }
  };

  const filteredTerms = terms.filter(
    (t) =>
      t.text.toLowerCase().includes(search.toLowerCase()) ||
      t.translation.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="h-screen flex flex-col overflow-y-auto px-8 py-6">
      <div className="max-w-4xl w-full mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold tracking-tight text-apple-text dark:text-apple-darkText">
              {english ? 'Glossary' : '术语锁定词库'}
            </h2>
            <p className="text-xs text-apple-muted mt-1">
              {english
                ? 'Lock key terms during translation to keep terminology consistent.'
                : '在翻译过程中锁定专业名词，确保翻译前后上下文精确统一'}
            </p>
          </div>
          <RefreshButton
            onRefresh={loadTerms}
            title={english ? 'Refresh list' : '刷新列表'}
            className="p-1.5 rounded-lg border border-apple-border dark:border-apple-darkBorder hover:bg-black/5 dark:hover:bg-white/5 text-apple-muted"
          />
        </div>

        {/* 新增术语表单卡片 */}
        <form
          onSubmit={handleAdd}
          className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl p-4 flex items-center space-x-3"
        >
          <input
            type="text"
            placeholder={english ? 'Source term (e.g. Agent)' : '原文词汇 (如 Agent)'}
            value={source}
            onChange={(e) => setSource(e.target.value)}
            className="flex-1 bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
          />
          <input
            type="text"
            placeholder={english ? 'Translation (e.g. agent)' : '目标译文 (如 智能体)'}
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            className="flex-1 bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg px-3 py-1.5 text-xs text-apple-text dark:text-apple-darkText focus:outline-none focus:ring-1 focus:ring-apple-accent"
          />
          <button
            type="submit"
            className="flex items-center space-x-1 bg-apple-accent hover:bg-apple-accentHover text-white px-4 py-1.5 rounded-lg text-xs font-semibold shrink-0"
          >
            <Plus size={14} />
            <span>{t('glossary.addTerm', english ? 'Add term' : '添加术语')}</span>
          </button>
        </form>

        {/* 词库列表 */}
        <div className="bg-apple-card dark:bg-apple-darkCard border border-apple-border dark:border-apple-darkBorder rounded-xl overflow-hidden">
          <div className="p-3 border-b border-apple-border dark:border-apple-darkBorder flex items-center justify-between">
            <span className="text-xs font-semibold text-apple-muted">
              {english ? `Terms (${filteredTerms.length})` : `当前收录 (${filteredTerms.length})`}
            </span>
            <div className="relative">
              <Search size={13} className="absolute left-2.5 top-2 text-apple-muted" />
              <input
                type="text"
                placeholder={english ? 'Search terms...' : '搜索术语...'}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-7 pr-2.5 py-1 text-xs bg-apple-bg dark:bg-apple-darkBg border border-apple-border dark:border-apple-darkBorder rounded-lg text-apple-text dark:text-apple-darkText w-40 focus:outline-none"
              />
            </div>
          </div>

          <table className="w-full text-left text-xs">
            <thead className="bg-black/5 dark:bg-white/5 text-apple-muted border-b border-apple-border dark:border-apple-darkBorder">
              <tr>
                <th className="py-2.5 px-4 font-semibold">{english ? 'Source term' : '原文词条'}</th>
                <th className="py-2.5 px-4 font-semibold">{english ? 'Fixed translation' : '锁定译文'}</th>
                <th className="py-2.5 px-4 font-semibold text-right">{english ? 'Actions' : '操作'}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-apple-border dark:divide-apple-darkBorder text-apple-text dark:text-apple-darkText">
              {loading ? (
                <tr>
                  <td colSpan={3} className="py-6 text-center text-apple-muted">
                    {english ? 'Loading...' : '正在加载...'}
                  </td>
                </tr>
              ) : filteredTerms.length === 0 ? (
                <tr>
                  <td colSpan={3} className="py-6 text-center text-apple-muted">
                    {english ? 'No terms yet' : '暂无术语'}
                  </td>
                </tr>
              ) : (
                filteredTerms.map((term) => (
                  <tr key={term.id} className="hover:bg-black/5 dark:hover:bg-white/5 transition-colors">
                    <td className="py-2.5 px-4 font-medium">{term.text}</td>
                    <td className="py-2.5 px-4 font-medium text-apple-accent">{term.translation}</td>
                    <td className="py-2.5 px-4 text-right">
                      {deletingId === term.id ? (
                        <span className="inline-flex items-center space-x-1.5">
                          <button
                            type="button"
                            onClick={() => handleDelete(term.id)}
                            className="text-rose-600 hover:text-rose-700 dark:text-rose-400 font-bold text-[11px] px-1.5 py-0.5 rounded bg-rose-50 dark:bg-rose-950/40"
                          >
                            {english ? 'Confirm' : '确认'}
                          </button>
                          <button
                            type="button"
                            onClick={() => setDeletingId(null)}
                            className="text-slate-400 hover:text-slate-600 text-[11px] px-1"
                          >
                            {english ? 'Cancel' : '取消'}
                          </button>
                        </span>
                      ) : (
                        <button
                          type="button"
                          onClick={() => setDeletingId(term.id)}
                          className="text-apple-muted hover:text-red-500 p-1"
                          title={english ? 'Delete term' : '删除词条'}
                        >
                          <Trash2 size={13} />
                        </button>
                      )}
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
