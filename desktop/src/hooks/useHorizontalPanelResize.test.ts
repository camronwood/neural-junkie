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

  it('reclamps saved width when max shrinks', () => {
    localStorage.setItem('test-panel-width-reclamp', '800');

    const { result, rerender } = renderHook(
      ({ max }: { max: number }) =>
        useHorizontalPanelResize({
          storageKey: 'test-panel-width-reclamp',
          defaultWidth: 420,
          minWidth: 260,
          maxWidthRatio: 0.65,
          getMaxWidth: () => max,
          edge: 'left',
          reclampKey: String(max),
        }),
      { initialProps: { max: 1200 } },
    );

    expect(result.current.width).toBe(800);

    rerender({ max: 600 });
    expect(result.current.width).toBe(600);
  });

  it('reclamps on window resize when max shrinks', () => {
    localStorage.setItem('test-panel-width-window', '800');
    let max = 1200;

    const { result } = renderHook(() =>
      useHorizontalPanelResize({
        storageKey: 'test-panel-width-window',
        defaultWidth: 420,
        minWidth: 260,
        maxWidthRatio: 0.65,
        getMaxWidth: () => max,
        edge: 'left',
      }),
    );

    expect(result.current.width).toBe(800);

    max = 600;
    act(() => {
      window.dispatchEvent(new Event('resize'));
    });

    expect(result.current.width).toBe(600);
  });
});
