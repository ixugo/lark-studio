import { api } from './api';

export interface ModelDownloadProgress {
  percent: number | null;
  speed: string;
  downloaded: string;
  total: string;
}

export const initialModelDownloadProgress: ModelDownloadProgress = {
  percent: null, speed: '', downloaded: '0 B', total: '',
};

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B';
  const units = ['B', 'KiB', 'MiB', 'GiB'];
  const index = Math.min(3, Math.floor(Math.log(value) / Math.log(1024)));
  return `${(value / 1024 ** index).toFixed(index ? 1 : 0)} ${units[index]}`;
}

export function listenModelDownload(handlers: {
  progress: (model: string, progress: ModelDownloadProgress) => void;
  done: (model: string) => void;
  error: (message: string) => void;
}): () => void {
  const off = [
    api.onEvent('model_download_progress', (value) => {
      const data = value as { model?: string; progress?: number; speed?: string; downloaded?: number; total?: number };
      if (!data?.model) return;
      handlers.progress(data.model, {
        percent: (data.total || 0) > 0 ? Math.max(0, Math.min(100, Math.round(data.progress || 0))) : null,
        speed: data.speed || '',
        downloaded: formatBytes(data.downloaded || 0),
        total: (data.total || 0) > 0 ? formatBytes(data.total!) : '',
      });
    }),
    api.onEvent('model_download_done', (value) => {
      const data = value as { model?: string };
      if (data?.model) handlers.done(data.model);
    }),
    api.onEvent('model_download_failed', (value) => {
      const data = value as { error?: string };
      handlers.error(data?.error || '模型下载失败');
    }),
  ];
  return () => off.forEach(unsubscribe => unsubscribe());
}
