import { describe, it, expect } from 'vitest';
import { api } from './api';

describe('API 客户端与数据映射测试', () => {
  it('应当正确获取配置与默认翻译引擎', async () => {
    const config = await api.getConfig();
    expect(config).toBeDefined();
    expect(config.llm.provider).toBe('openai');
    expect(config.llm.base_url).toContain('api.openai.com');
  });

  it('应当正确创建任务并校验输入参数', async () => {
    const task = await api.createTask({
      input_path: '/mock/video.mp4',
      target_lang: 'zh-CN',
      mode: 2,
      translator: 'openai',
    });

    expect(task).toBeDefined();
    expect(task.id).toBeDefined();
    expect(task.mode).toBe(2);
    expect(task.target_lang).toBe('zh-CN');
  });

  it('应当正确获取任务列表与日志', async () => {
    const tasks = await api.listTasks();
    expect(Array.isArray(tasks)).toBe(true);
    expect(tasks.length).toBeGreaterThan(0);

    const firstTask = tasks[0];
    const logs = await api.listTaskLogs(firstTask.id);
    expect(Array.isArray(logs)).toBe(true);
    expect(logs.length).toBeGreaterThan(0);
  });

  it('应当支持术语词库增删查', async () => {
    const initialTerms = await api.listTerms();
    expect(Array.isArray(initialTerms)).toBe(true);

    await api.saveTerm('Prompt', '提示词');

    const updatedTerms = await api.listTerms();
    const found = updatedTerms.some((t) => t.text === 'Prompt' && t.translation === '提示词');
    expect(found).toBe(true);
  });
});
