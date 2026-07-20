import { describe, expect, it, vi, afterEach, beforeEach } from 'vitest';

const fetchWorkspaceImageDataUrl = vi.fn();

vi.mock('../api/chatAPI', () => ({
  ChatAPI: class {
    fetchWorkspaceImageDataUrl = fetchWorkspaceImageDataUrl;
  },
}));

vi.mock('../config/hubUrl', () => ({
  getHubBaseURL: () => 'http://127.0.0.1:18765',
}));

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

describe('resolveEditorImageSrc', () => {
  beforeEach(() => {
    fetchWorkspaceImageDataUrl.mockReset();
    fetchWorkspaceImageDataUrl.mockResolvedValue('data:image/png;base64,abc');
  });

  afterEach(() => {
    vi.resetModules();
    Reflect.deleteProperty(window, '__TAURI__');
  });

  it('loads via hub data URL even inside Tauri', async () => {
    Object.defineProperty(window, '__TAURI__', { value: {}, configurable: true });
    vi.resetModules();
    const { resolveEditorImageSrc } = await import('./chatImageSrc');
    const src = await resolveEditorImageSrc({
      workspaceId: 'ws-1',
      relativePath: 'assets/dickory-docs-download-ad-1080.png',
      absolutePath: '/Users/me/proj/assets/dickory-docs-download-ad-1080.png',
    });
    expect(src).toBe('data:image/png;base64,abc');
    expect(fetchWorkspaceImageDataUrl).toHaveBeenCalledWith(
      'ws-1',
      'assets/dickory-docs-download-ad-1080.png',
    );
  });

  it('loads nested workspace image paths via hub', async () => {
    vi.resetModules();
    const { resolveEditorImageSrc } = await import('./chatImageSrc');
    await resolveEditorImageSrc({
      workspaceId: 'ws-2',
      relativePath: 'docs/media/cover.png',
      absolutePath: '/tmp/ws/docs/media/cover.png',
    });
    expect(fetchWorkspaceImageDataUrl).toHaveBeenCalledWith('ws-2', 'docs/media/cover.png');
  });
});

describe('resolveChatImageSrc security', () => {
  afterEach(() => {
    vi.resetModules();
  });

  it('rejects javascript URLs and non-image data URLs', async () => {
    const { resolveChatImageSrc } = await import('./chatImageSrc');
    expect(resolveChatImageSrc('javascript:alert(1)')).toBe('');
    expect(resolveChatImageSrc('data:text/html,hi')).toBe('');
  });
});
