import { afterEach, describe, expect, it, vi } from 'vitest';
import { api } from './api';
import { initialModelDownloadProgress, listenModelDownload } from './modelDownload';

afterEach(() => vi.restoreAllMocks());

describe('模型下载事件与进度', () => {
  it('匹配后端事件名和字段，能收到实际字节进度、完成与错误并取消监听', () => {
    const events = new Map<string, (value: unknown) => void>();
    const off = vi.fn();
    vi.spyOn(api, 'onEvent').mockImplementation((name, callback) => { events.set(name, callback); return off; });
    const handlers = { progress: vi.fn(), done: vi.fn(), error: vi.fn() };
    const unsubscribe = listenModelDownload(handlers);
    expect([...events.keys()]).toEqual(['model_download_progress', 'model_download_done', 'model_download_failed']);
    events.get('model_download_progress')!({ model: 'tiny', progress: 25, speed: '1 MB/s', downloaded: 1024 ** 2, total: 4 * 1024 ** 2 });
    expect(handlers.progress).toHaveBeenCalledWith('tiny', { percent: 25, speed: '1 MB/s', downloaded: '1.0 MiB', total: '4.0 MiB' });
    events.get('model_download_done')!({ model: 'tiny' });
    expect(handlers.done).toHaveBeenCalledWith('tiny');
    events.get('model_download_failed')!({ model: 'tiny', error: 'HTTP 503' });
    expect(handlers.error).toHaveBeenCalledWith('HTTP 503');
    unsubscribe();
    expect(off).toHaveBeenCalledTimes(3);
  });
  it('开始时不捏造速度和百分比；总大小未知时显示下载量', () => {
    expect(initialModelDownloadProgress).toEqual({ percent: null, speed: '', downloaded: '0 B', total: '' });
    const events = new Map<string, (value: unknown) => void>();
    vi.spyOn(api, 'onEvent').mockImplementation((name, callback) => { events.set(name, callback); return () => {}; });
    const progress = vi.fn();
    listenModelDownload({ progress, done: vi.fn(), error: vi.fn() });
    events.get('model_download_progress')!({ model: 'tiny', downloaded: 4096, total: -1, progress: 0 });
    expect(progress).toHaveBeenCalledWith('tiny', { percent: null, speed: '', downloaded: '4.0 KiB', total: '' });
  });
});
