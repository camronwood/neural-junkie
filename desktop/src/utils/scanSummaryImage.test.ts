import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const fetchScanSummaryWellImage = vi.fn();

vi.mock('../api/chatAPI', () => ({
  ChatAPI: class {
    fetchScanSummaryWellImage = fetchScanSummaryWellImage;
  },
}));

vi.mock('../config/hubUrl', () => ({
  getHubBaseURL: () => 'http://127.0.0.1:18765',
}));

vi.mock('@tauri-apps/api/tauri', () => ({
  invoke: vi.fn(),
}));

import { resolveScanSummaryWellImageSrc } from './scanSummaryImage';

describe('resolveScanSummaryWellImageSrc', () => {
  const originalTauri = (window as Window & { __TAURI__?: unknown }).__TAURI__;

  beforeEach(() => {
    fetchScanSummaryWellImage.mockReset();
    delete (window as Window & { __TAURI__?: unknown }).__TAURI__;
  });

  afterEach(() => {
    if (originalTauri !== undefined) {
      (window as Window & { __TAURI__?: unknown }).__TAURI__ = originalTauri;
    } else {
      delete (window as Window & { __TAURI__?: unknown }).__TAURI__;
    }
  });

  it('uses hub API when not in Tauri shell', async () => {
    fetchScanSummaryWellImage.mockResolvedValue('data:image/png;base64,abc');
    const src = await resolveScanSummaryWellImageSrc({
      workspaceId: 'ws-1',
      workspacePath: '/tmp/ws',
      summaryDir: 'run1',
      wellId: 'A1',
    });
    expect(src).toBe('data:image/png;base64,abc');
    expect(fetchScanSummaryWellImage).toHaveBeenCalledWith('ws-1', 'run1', 'A1');
  });
});
