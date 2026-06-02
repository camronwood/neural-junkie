import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useHorizontalPanelResize } from './useHorizontalPanelResize';

describe('useHorizontalPanelResize', () => {
  beforeEach(() => {
    vi.stubGlobal('innerWidth', 2000);
    localStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('clamps width to getMaxWidth when provided', () => {
    const { result } = renderHook(() =>
      useHorizontalPanelResize({
        storageKey: 'test-panel-width',
        defaultWidth: 420,
        minWidth: 260,
        maxWidthRatio: 0.65,
        getMaxWidth: () => 900,
        edge: 'left',
      })
    );

    act(() => {
      result.current.onResizeStart({ preventDefault: () => {}, clientX: 1000 } as React.MouseEvent);
    });

    act(() => {
      document.dispatchEvent(new MouseEvent('mousemove', { clientX: 0 }));
    });

    expect(result.current.width).toBeLessThanOrEqual(900);
    expect(result.current.width).toBeGreaterThanOrEqual(260);
  });
});
