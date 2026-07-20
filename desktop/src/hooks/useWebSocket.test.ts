import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useWebSocket } from './useWebSocket';

class MockWebSocket {
  static OPEN = 1;
  static CONNECTING = 0;
  static CLOSED = 3;
  static instances: MockWebSocket[] = [];

  readyState = MockWebSocket.CONNECTING;
  onopen: ((ev?: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onclose: ((ev?: CloseEvent) => void) | null = null;
  url: string;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close'));
  }

  open() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event('open'));
  }
}

describe('useWebSocket', () => {
  const OriginalWebSocket = globalThis.WebSocket;

  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.stubGlobal('WebSocket', MockWebSocket as unknown as typeof WebSocket);
  });

  afterEach(() => {
    vi.stubGlobal('WebSocket', OriginalWebSocket);
  });

  it('connects and reports connected on open', () => {
    const { result } = renderHook(() =>
      useWebSocket({
        url: 'ws://127.0.0.1:18765/ws?channel=a',
        onMessage: () => {},
        autoReconnect: false,
      }),
    );

    expect(MockWebSocket.instances).toHaveLength(1);
    act(() => {
      MockWebSocket.instances[0].open();
    });
    expect(result.current.status).toBe('connected');
  });

  it('ignores stale onclose from a previous generation after reconnect', () => {
    const { result } = renderHook(() =>
      useWebSocket({
        url: 'ws://127.0.0.1:18765/ws?channel=a',
        onMessage: () => {},
        autoReconnect: false,
      }),
    );

    const first = MockWebSocket.instances[0];
    const staleClose = first.onclose;
    expect(staleClose).toBeTypeOf('function');

    act(() => {
      first.open();
    });
    expect(result.current.status).toBe('connected');

    act(() => {
      result.current.connect();
    });
    const second = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    expect(second).not.toBe(first);

    act(() => {
      second.open();
    });
    expect(result.current.status).toBe('connected');

    // Fire the captured first-generation close after a newer socket is live.
    act(() => {
      staleClose?.(new CloseEvent('close'));
    });
    expect(result.current.status).toBe('connected');
  });

  it('reconnects when URL changes', () => {
    const { result, rerender } = renderHook(
      ({ url }: { url: string }) =>
        useWebSocket({
          url,
          onMessage: () => {},
          autoReconnect: false,
        }),
      { initialProps: { url: 'ws://127.0.0.1:18765/ws?channel=a' } },
    );

    act(() => {
      MockWebSocket.instances[0].open();
    });
    expect(result.current.status).toBe('connected');

    rerender({ url: 'ws://127.0.0.1:18765/ws?channel=b' });
    const latest = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    expect(latest.url).toContain('channel=b');
    act(() => {
      latest.open();
    });
    expect(result.current.status).toBe('connected');
  });
});
