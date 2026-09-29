import React, { useEffect, useRef, useState } from 'react';
import { X } from 'lucide-react';
import { api } from '../lib/api';
import { UpdateInfo, UpdateStatus } from '../types';
import { useTranslation } from '../i18n';

const STATUS_POLL_MS = 500;

export const UpdateDialog: React.FC<{ update: UpdateInfo; onClose: () => void }> = ({ update, onClose }) => {
  const { t } = useTranslation();
  const dialog = useRef<HTMLDivElement>(null);
  const [busy, setBusy] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [status, setStatus] = useState<UpdateStatus>({ phase: 'idle', message: '' });
  const [error, setError] = useState('');

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.current?.focus();
    return () => previous?.focus();
  }, []);

  useEffect(() => {
    if (!installing) return;
    let active = true;
    let pending = false;
    const timer = setInterval(async () => {
      if (pending) return;
      pending = true;
      try {
        const next = await api.getUpdateStatus();
        if (active) setStatus(next);
      } catch (err) {
        if (active) console.warn('Update status check failed', err);
      } finally {
        pending = false;
      }
    }, STATUS_POLL_MS);
    return () => { active = false; clearInterval(timer); };
  }, [installing]);

  const install = async () => {
    setError('');
    setStatus({ phase: 'downloading', message: '' });
    setBusy(true);
    setInstalling(true);
    try {
      await api.installUpdate(update.version);
      setStatus({ phase: 'restarting', message: '' });
    } catch (err) {
      setError(String(err));
      setStatus({ phase: 'error', message: '' });
      setBusy(false);
      setInstalling(false);
    }
  };

  const ignore = async () => {
    setBusy(true);
    setError('');
    try {
      await api.ignoreUpdate(update.version);
      onClose();
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  };

  const onKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Escape' && !busy) onClose();
    if (event.key !== 'Tab') return;
    const buttons = dialog.current?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)');
    if (!buttons?.length) { event.preventDefault(); return; }
    const first = buttons[0];
    const last = buttons[buttons.length - 1];
    if (event.shiftKey && (document.activeElement === first || document.activeElement === dialog.current)) {
      event.preventDefault(); last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault(); first.focus();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/35 p-6" onMouseDown={(event) => { if (event.target === event.currentTarget && !busy) onClose(); }}>
      <div ref={dialog} tabIndex={-1} role="dialog" aria-modal="true" aria-labelledby="update-title" onKeyDown={onKeyDown} className="w-full max-w-lg rounded-2xl border border-slate-200 dark:border-white/10 bg-white dark:bg-[#1C1C1E] p-6 shadow-2xl outline-none select-text">
        <div className="flex items-start justify-between gap-4">
          <h2 id="update-title" className="text-lg font-bold">{t('update.title', { version: `v${update.version.replace(/^v/, '')}` })}</h2>
          <button disabled={busy} onClick={onClose} aria-label={t('common.close')} className="p-1 rounded-lg text-slate-400 hover:bg-slate-100 dark:hover:bg-white/10 disabled:opacity-30"><X size={18} /></button>
        </div>
        <div className="mt-4 max-h-64 overflow-y-auto whitespace-pre-wrap break-words text-sm leading-relaxed text-slate-700 dark:text-slate-300">{update.notes || t('update.noNotes')}</div>
        {!update.supported && <p className="mt-4 text-sm text-amber-600 dark:text-amber-400">{update.reason || t('update.unsupported')}</p>}
        {installing && <p role="status" aria-live="polite" className="mt-4 text-sm text-blue-600 dark:text-blue-400">{status.message || t(`update.${status.phase}`)}</p>}
        {error && <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400 break-words">{t('update.failed')}: {error}</p>}
        <div className="flex justify-end gap-3 mt-6">
          <button disabled={busy} onClick={ignore} className="px-4 py-2 text-sm rounded-lg bg-slate-100 dark:bg-white/10 disabled:opacity-50">{t('update.ignore')}</button>
          <button disabled={busy || !update.supported} onClick={install} className="px-4 py-2 text-sm font-semibold rounded-lg bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50">{t(error ? 'update.retry' : 'update.install')}</button>
        </div>
      </div>
    </div>
  );
};
