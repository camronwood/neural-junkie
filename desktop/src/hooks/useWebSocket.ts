import { useEffect, useRef, useState, useCallback } from 'react';
import type { Message } from '../types/protocol';
import { perfMarkStart, perfMarkEnd } from '../utils/perfMarks';
import { sanitizeWsUrl, updateWsDiagnostic } from '../utils/wsDiagnostics';

export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error';

interface UseWebSocketOptions {
  url: string;
  onMessage: (message: Message) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  autoReconnect?: boolean;
  reconnectInterval?: number;
  /** When false, tear down any socket and stay disconnected until enabled. */
  enabled?: boolean;
  /** Whether a hub session token is present (for diagnostics only). */
  hasSession?: boolean;
}

function clearSocketHandlers(ws: WebSocket): void {
  ws.onopen = null;
  ws.onmessage = null;
  ws.onerror = null;
  ws.onclose = null;
}

export function useWebSocket({
  url,
  onMessage,
  onConnect,
  onDisconnect,
  onError,
  autoReconnect = true,
  reconnectInterval = 3000,
  enabled = true,
  hasSession = false,
}: UseWebSocketOptions) {
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const shouldReconnectRef = useRef(autoReconnect);
  /** Bumps on each connect/teardown so stale socket events cannot clobber UI status. */
  const generationRef = useRef(0);

  const onMessageRef = useRef(onMessage);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const onErrorRef = useRef(onError);
  const urlRef = useRef(url);

  useEffect(() => {
    onMessageRef.current = onMessage;
    onConnectRef.current = onConnect;
    onDisconnectRef.current = onDisconnect;
    onErrorRef.current = onError;
  }, [onMessage, onConnect, onDisconnect, onError]);

  useEffect(() => {
    urlRef.current = url;
  }, [url]);

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimeoutRef.current != null) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  const tearDownSocket = useCallback((ws: WebSocket | null) => {
    if (!ws) return;
    clearSocketHandlers(ws);
    if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
      try {
        ws.close();
      } catch {
        /* ignore */
      }
    }
  }, []);

  const connect = useCallback(() => {
    clearReconnectTimer();

    const gen = ++generationRef.current;
    const prev = wsRef.current;
    wsRef.current = null;
    tearDownSocket(prev);

    setStatus('connecting');
    console.log('Connecting to WebSocket:', urlRef.current);
    updateWsDiagnostic({
      url: sanitizeWsUrl(urlRef.current),
      origin: window.location.origin,
      enabled: true,
      hasSession,
      status: 'connecting',
    });

    try {
      const ws = new WebSocket(urlRef.current);

      ws.onopen = () => {
        if (gen !== generationRef.current) return;
        console.log('WebSocket connected');
        setStatus('connected');
        updateWsDiagnostic({ status: 'connected', lastError: null });
        onConnectRef.current?.();
      };

      ws.onmessage = (event) => {
        if (gen !== generationRef.current) return;
        perfMarkStart('ws.onmessage');
        try {
          const message: Message = JSON.parse(event.data);
          onMessageRef.current(message);
        } catch (error) {
          console.error('Failed to parse message:', error);
        } finally {
          perfMarkEnd('ws.onmessage');
        }
      };

      ws.onerror = (error) => {
        if (gen !== generationRef.current) return;
        console.error('WebSocket error:', error);
        setStatus('error');
        updateWsDiagnostic({
          status: 'error',
          lastError: 'WebSocket error event (see console)',
        });
        onErrorRef.current?.(error);
      };

      ws.onclose = (event) => {
        if (gen !== generationRef.current) return;
        console.log('WebSocket disconnected', event.code, event.reason || '');
        if (wsRef.current === ws) {
          wsRef.current = null;
        }
        setStatus('disconnected');
        updateWsDiagnostic({
          status: 'disconnected',
          lastCloseCode: event.code,
          lastCloseReason: event.reason || null,
        });
        onDisconnectRef.current?.();

        if (shouldReconnectRef.current && autoReconnect) {
          reconnectTimeoutRef.current = window.setTimeout(() => {
            if (gen !== generationRef.current) return;
            console.log('Attempting to reconnect...');
            connect();
          }, reconnectInterval);
        }
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      if (gen === generationRef.current) {
        setStatus('error');
        updateWsDiagnostic({
          status: 'error',
          lastError: error instanceof Error ? error.message : 'Failed to create WebSocket',
        });
      }
    }
  }, [autoReconnect, reconnectInterval, clearReconnectTimer, tearDownSocket, hasSession]);

  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;
    clearReconnectTimer();
    generationRef.current += 1;
    const prev = wsRef.current;
    wsRef.current = null;
    tearDownSocket(prev);
    setStatus('disconnected');
  }, [clearReconnectTimer, tearDownSocket]);

  // Connect on mount and whenever URL / reconnect policy / enabled changes.
  useEffect(() => {
    if (!enabled) {
      shouldReconnectRef.current = false;
      clearReconnectTimer();
      generationRef.current += 1;
      const prev = wsRef.current;
      wsRef.current = null;
      tearDownSocket(prev);
      setStatus('disconnected');
      updateWsDiagnostic({
        url: sanitizeWsUrl(url),
        origin: window.location.origin,
        enabled: false,
        hasSession,
        status: 'disconnected',
      });
      return;
    }

    updateWsDiagnostic({
      url: sanitizeWsUrl(url),
      origin: window.location.origin,
      enabled: true,
      hasSession,
    });
    shouldReconnectRef.current = autoReconnect;
    connect();

    return () => {
      shouldReconnectRef.current = false;
      clearReconnectTimer();
      generationRef.current += 1;
      const prev = wsRef.current;
      wsRef.current = null;
      tearDownSocket(prev);
    };
  }, [url, connect, autoReconnect, enabled, hasSession, clearReconnectTimer, tearDownSocket]);

  return {
    status,
    connect,
    disconnect,
  };
}
