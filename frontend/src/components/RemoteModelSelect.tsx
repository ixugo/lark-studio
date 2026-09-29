import React, { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../lib/api';
import { finishRemoteModels, hasRemoteModel, pendingRemoteModels } from '../lib/remoteModels';
import { useTranslation } from '../i18n';
import { RefreshButton } from './RefreshButton';

const CONNECTION_EDIT_DELAY_MS = 400;

export function useRemoteModels(baseURL: string, apiKey: string, enabled: boolean) {
  const [state, setState] = useState(() => pendingRemoteModels(0, '', ''));
  const requestId = useRef(0);
  const connection = useRef({ baseURL, apiKey, enabled });
  connection.current = { baseURL, apiKey, enabled };

  const refresh = useCallback(async () => {
    if (!enabled || !baseURL.trim()) return;
    const request = pendingRemoteModels(++requestId.current, baseURL, apiKey);
    setState(request);
    const isCurrentRequest = () => requestId.current === request.requestId && connection.current.enabled &&
      connection.current.baseURL === baseURL && connection.current.apiKey === apiKey;
    try {
      const models = await api.listRemoteModels(baseURL, apiKey);
      if (!isCurrentRequest()) return;
      setState(current => finishRemoteModels(current, request, models));
    } catch (error) {
      if (!isCurrentRequest()) return;
      const message = error instanceof Error ? error.message : String(error);
      setState(current => finishRemoteModels(current, request, [], message));
    }
  }, [baseURL, apiKey, enabled]);

  useEffect(() => {
    const id = ++requestId.current;
    setState(pendingRemoteModels(id, baseURL, apiKey));
    if (!enabled || !baseURL.trim()) return;
    const timer = setTimeout(() => { void refresh(); }, CONNECTION_EDIT_DELAY_MS);
    return () => { clearTimeout(timer); ++requestId.current; };
  }, [baseURL, apiKey, enabled, refresh]);

  // 输入发生变化时立即拒绝旧列表，不能等待下一次 effect 才失效。
  const current = state.baseURL === baseURL && state.apiKey === apiKey && enabled;
  return {
    models: current ? state.models : [],
    loading: enabled && !!baseURL.trim() && (!current || state.status === 'loading'),
    error: current ? state.error : '',
    refresh,
    hasModel: (model: string) => enabled && hasRemoteModel(state, connection.current.baseURL, connection.current.apiKey, model),
  };
}

export const RemoteModelSelect: React.FC<{
  catalog: ReturnType<typeof useRemoteModels>;
  value: string;
  onChange: (model: string) => void;
  hasAddress: boolean;
}> = ({ catalog, value, onChange, hasAddress }) => {
  const { locale } = useTranslation();
  const en = locale === 'en-US';
  const selected = catalog.hasModel(value) ? value : '';
  return (
    <div className="space-y-1.5">
      <div className="flex gap-2">
        <select aria-label={en ? 'Remote model' : '远程模型'} value={selected}
          onChange={event => onChange(event.target.value)} disabled={catalog.loading || catalog.models.length === 0}
          className="w-full min-w-0 h-10 px-3 rounded-xl border border-slate-200 dark:border-white/10 bg-slate-50 dark:bg-white/5 text-xs text-slate-800 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/30 disabled:opacity-60">
          <option value="">{catalog.loading ? (en ? 'Loading models…' : '正在获取模型…') : (en ? 'Select a remote model' : '请选择远程模型')}</option>
          {catalog.models.map(model => <option key={model.id} value={model.id}>{model.id}</option>)}
        </select>
        <RefreshButton onRefresh={catalog.refresh} title={en ? 'Refresh models' : '刷新模型'}
          className="px-3 rounded-xl border border-slate-200 dark:border-white/10 text-slate-500 hover:text-blue-600" />
      </div>
      <p role={catalog.error ? 'alert' : undefined} className={`text-[11px] ${catalog.error ? 'text-rose-600 dark:text-rose-400' : 'text-slate-400'}`}>
        {catalog.error || (!hasAddress ? (en ? 'Enter the API address first.' : '请先填写接口地址。')
          : !catalog.loading && catalog.models.length === 0 ? (en ? 'The service returned no models.' : '服务未返回可选模型。')
            : value && !selected && !catalog.loading ? (en ? 'The saved model is unavailable. Select another model.' : '原模型不在远程列表中，请重新选择。')
              : (en ? 'Only models returned by the current service can be selected.' : '仅能选择当前服务返回的模型。'))}
      </p>
    </div>
  );
};
