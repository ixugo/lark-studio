import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { LanguageProvider } from '../i18n';
import { UpdateDialog } from './UpdateDialog';

const render = (supported: boolean) => renderToStaticMarkup(
  <LanguageProvider>
    <UpdateDialog update={{ version: '0.2.0', notes: '<script>alert(1)</script>\nSecond line', available: true, supported, reason: supported ? '' : 'No macOS asset' }} onClose={() => {}} />
  </LanguageProvider>,
);

describe('更新弹窗', () => {
  it('将发布说明作为纯文本保留，呈现版本和两个更新动作', () => {
    const html = render(true);
    expect(html).toContain('role="dialog"');
    expect(html).toContain('aria-modal="true"');
    expect(html).toContain('发现新版本 v0.2.0');
    expect(html).not.toContain('更新说明');
    expect(html).toContain('&lt;script&gt;alert(1)&lt;/script&gt;\nSecond line');
    expect(html).not.toContain('<script>');
    expect(html).toContain('忽略此版本');
    expect(html).toContain('立即更新');
  });

  it('没有安装包时仍显示发布说明和原因，并禁用更新', () => {
    const html = render(false);
    expect(html).toContain('No macOS asset');
    expect(html).toContain('Second line');
    expect(html).toMatch(/<button disabled=""[^>]*>立即更新<\/button>/);
  });
});
