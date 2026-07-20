import { useEffect, useRef, useState, useCallback } from 'react';
import type { Message } from '../types/protocol';
import { perfMarkStart, perfMarkEnd } from '../utils/perfMarks';

export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error';

interface UseWebSocketOptions {
  url: string;
  onMessage: (message: Message) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
  autoReconnect?: boolean;
  reconnectInterval?: number;
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

    try {
      const ws = new WebSocket(urlRef.current);

      ws.onopen = () => {
        if (gen !== generationRef.current) return;
        console.log('WebSocket connected');
        setStatus('connected');
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
        onErrorRef.current?.(error);
      };

      ws.onclose = () => {
        if (gen !== generationRef.current) return;
        console.log('WebSocket disconnected');
        if (wsRef.current === ws) {
          wsRef.current = null;
        }
        setStatus('disconnected');
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
      }
    }
  }, [autoReconnect, reconnectInterval, clearReconnectTimer, tearDownSocket]);

  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;
    clearReconnectTimer();
    generationRef.current += 1;
    const prev = wsRef.current;
    wsRef.current = null;
    tearDownSocket(prev);
    setStatus('disconnected');
  }, [clearReconnectTimer, tearDownSocket]);

  // Connect on mount and whenever URL / reconnect policy changes.
  useEffect(() => {
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
  }, [url, connect, autoReconnect, clearReconnectTimer, tearDownSocket]);

  return {
    status,
    connect,
    disconnect,
  };
}
