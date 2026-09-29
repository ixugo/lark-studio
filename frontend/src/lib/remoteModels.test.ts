import { describe, expect, it } from 'vitest';
import { finishRemoteModels, hasRemoteModel, pendingRemoteModels } from './remoteModels';

describe('远程模型选择与连接隔离', () => {
  it('保存的模型在当前服务返回确认前不能提交', () => {
    const pending = pendingRemoteModels(1, 'https://one.test/v1', 'key');
    expect(hasRemoteModel(pending, pending.baseURL, pending.apiKey, 'whisper-1')).toBe(false);
    const ready = finishRemoteModels(pending, pending, [{ id: 'whisper-1' }]);
    expect(hasRemoteModel(ready, ready.baseURL, ready.apiKey, 'whisper-1')).toBe(true);
    expect(hasRemoteModel(ready, ready.baseURL, ready.apiKey, 'invented-model')).toBe(false);
  });

  it('地址或密钥变化立即使旧列表失效', () => {
    const request = pendingRemoteModels(1, 'https://one.test/v1', 'key');
    const ready = finishRemoteModels(request, request, [{ id: 'model-a' }]);
    expect(hasRemoteModel(ready, 'https://two.test/v1', 'key', 'model-a')).toBe(false);
    expect(hasRemoteModel(ready, ready.baseURL, 'new-key', 'model-a')).toBe(false);
  });

  it('较旧请求返回时不能覆盖新服务的列表', () => {
    const oldRequest = pendingRemoteModels(1, 'https://one.test/v1', 'key');
    const newRequest = pendingRemoteModels(2, 'https://two.test/v1', 'key');
    const ready = finishRemoteModels(newRequest, newRequest, [{ id: 'model-b' }]);
    expect(finishRemoteModels(ready, oldRequest, [{ id: 'model-a' }])).toBe(ready);
    expect(finishRemoteModels(ready, oldRequest, [], '旧连接错误')).toBe(ready);
  });

  it('同一连接刷新后，较旧请求也不能恢复已删除的模型', () => {
    const oldRequest = pendingRemoteModels(1, 'https://one.test/v1', 'key');
    const refreshed = pendingRemoteModels(2, oldRequest.baseURL, oldRequest.apiKey);
    const ready = finishRemoteModels(refreshed, refreshed, [{ id: 'new-model' }]);
    const lateResult = finishRemoteModels(ready, oldRequest, [{ id: 'old-model' }]);
    expect(lateResult).toBe(ready);
    expect(hasRemoteModel(lateResult, ready.baseURL, ready.apiKey, 'old-model')).toBe(false);
  });

  it('刷新删除的模型、空列表或错误不允许继续使用旧选择', () => {
    const request = pendingRemoteModels(2, 'https://one.test/v1', 'key');
    for (const state of [finishRemoteModels(request, request, []), finishRemoteModels(request, request, [], 'HTTP 401')]) {
      expect(hasRemoteModel(state, state.baseURL, state.apiKey, 'old-model')).toBe(false);
    }
  });
});
