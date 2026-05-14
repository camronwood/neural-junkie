import { describe, expect, it, vi, afterEach } from 'vitest';

describe('resolveChatImageSrc', () => {
  afterEach(() => {
    vi.resetModules();
    Reflect.deleteProperty(window, '__TAURI__');
  });

  it('passes through data and http URLs', async () => {
    vi.resetModules();
    const { resolveChatImageSrc } = await import('./chatImageSrc');
    expect(resolveChatImageSrc('data:image/png;base64,xx')).toBe('data:image/png;base64,xx');
    expect(resolveChatImageSrc('https://ex.com/a.png')).toBe('https://ex.com/a.png');
  });

  it('returns raw absolute path when not in Tauri', async () => {
    vi.resetModules();
    const { resolveChatImageSrc } = await import('./chatImageSrc');
    expect(resolveChatImageSrc('/Users/me/a.png')).toBe('/Users/me/a.png');
  });
});
