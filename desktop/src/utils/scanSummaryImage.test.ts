import { beforeEach, describe, expect, it, vi } from 'vitest';

const { fetchScanSummaryWellImage, invokeMock, isTauriMock } = vi.hoisted(() => ({
  fetchScanSummaryWellImage: vi.fn(),
  invokeMock: vi.fn(),
  isTauriMock: vi.fn(),
}));

vi.mock('../api/chatAPI', () => ({
  ChatAPI: class {
    fetchScanSummaryWellImage = fetchScanSummaryWellImage;
  },
}));

vi.mock('../config/hubUrl', () => ({
  getHubBaseURL: () => 'http://127.0.0.1:18765',
}));

vi.mock('@tauri-apps/api/core', () => ({
  invoke: invokeMock,
  isTauri: isTauriMock,
}));

import { resolveScanSummaryWellImageSrc } from './scanSummaryImage';

describe('resolveScanSummaryWellImageSrc', () => {
  beforeEach(() => {
    fetchScanSummaryWellImage.mockReset();
    invokeMock.mockReset();
    isTauriMock.mockReturnValue(false);
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

  it('passes allowedRoots to Tauri decode_scan_well_tiff', async () => {
    isTauriMock.mockReturnValue(true);
    invokeMock.mockResolvedValue({ mime: 'image/png', content_base64: 'abc' });
    await resolveScanSummaryWellImageSrc({
      workspaceId: 'ws-1',
      workspacePath: '/tmp/ws',
      summaryDir: 'run1',
      wellId: 'A1',
    });
    expect(invokeMock).toHaveBeenCalledWith(
      'decode_scan_well_tiff',
      expect.objectContaining({ allowedRoots: expect.any(Array) })
    );
  });
});
