export interface RemoteModel {
  id: string;
}

export interface RemoteModelState {
  requestId: number;
  baseURL: string;
  apiKey: string;
  status: 'loading' | 'ready' | 'error';
  models: RemoteModel[];
  error: string;
}

export function pendingRemoteModels(requestId: number, baseURL: string, apiKey: string): RemoteModelState {
  return { requestId, baseURL, apiKey, status: 'loading', models: [], error: '' };
}

export function finishRemoteModels(current: RemoteModelState, request: RemoteModelState, models: RemoteModel[], error = ''): RemoteModelState {
  if (current.requestId !== request.requestId || current.baseURL !== request.baseURL || current.apiKey !== request.apiKey) return current;
  return { ...request, models: error ? [] : models, status: error ? 'error' : 'ready', error };
}

export function hasRemoteModel(state: RemoteModelState, baseURL: string, apiKey: string, model: string): boolean {
  return state.status === 'ready' && state.baseURL === baseURL && state.apiKey === apiKey &&
    model !== '' && state.models.some(item => item.id === model);
}
