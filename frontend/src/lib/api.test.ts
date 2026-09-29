import { describe, it, expect, vi, afterEach } from 'vitest';
import { api } from './api';

describe('API 客户端与数据映射测试', () => {
  it('应当正确获取配置与默认翻译引擎', async () => {
    const config = await api.getConfig();
    expect(config).toBeDefined();
    expect(config.llm.provider).toBe('bing');
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


describe('桌面试听错误传递', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('保留包含 not found 的后端错误，不误报为浏览器运行', async () => {
    const error = new Error('Edge TTS 合成失败: executable file not found in $PATH');
    const byName = vi.fn().mockRejectedValue(error);
    vi.stubGlobal('window', { wails: { Call: { ByName: byName } } });
    await expect(api.testEdgeTTS('zh-CN-XiaoxiaoNeural', '你好')).rejects.toBe(error);
    expect(byName).toHaveBeenCalledTimes(1);
  });

  it('桌面接口返回可播放音频地址', async () => {
    const audio = 'data:audio/mpeg;base64,dGVzdA==';
    vi.stubGlobal('window', { wails: { Call: { ByName: vi.fn().mockResolvedValue(audio) } } });
    await expect(api.testEdgeTTS('zh-CN-XiaoxiaoNeural', '你好')).resolves.toBe(audio);
    await expect(api.testOpenAITTS('https://test.local', '', 'tts-1', 'alloy', 'hello')).resolves.toBe(audio);
  });
});

it('桌面连接未就绪时禁止用演示配置或演示任务兜底', async () => {
  vi.stubGlobal('window', { location: { protocol: 'wails:' } });
  try {
    await expect(api.getConfig()).rejects.toThrow('未就绪');
    await expect(api.createTask({ input_path: '/video.mp4', mode: 2 })).rejects.toThrow('未就绪');
  } finally {
    vi.unstubAllGlobals();
  }
});
