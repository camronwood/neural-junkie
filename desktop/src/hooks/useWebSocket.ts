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
  const isConnectingRef = useRef(false);
  const isMountedRef = useRef(false);
  
  // Use refs for callbacks to avoid recreating connect function
  const onMessageRef = useRef(onMessage);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const onErrorRef = useRef(onError);
  
  // Update refs when callbacks change
  useEffect(() => {
    onMessageRef.current = onMessage;
    onConnectRef.current = onConnect;
    onDisconnectRef.current = onDisconnect;
    onErrorRef.current = onError;
  }, [onMessage, onConnect, onDisconnect, onError]);

  const connect = useCallback(() => {
    // Prevent duplicate connection attempts
    if (isConnectingRef.current) {
      console.log('Connection attempt already in progress, skipping...');
      return;
    }

    // Prevent duplicate connections
    if (wsRef.current?.readyState === WebSocket.OPEN || 
        wsRef.current?.readyState === WebSocket.CONNECTING) {
      console.log('WebSocket already open or connecting, skipping...');
      return;
    }

    // Close any existing connection first (but not if it's still connecting)
    if (wsRef.current && wsRef.current.readyState === WebSocket.CLOSED) {
      wsRef.current = null;
    }

    isConnectingRef.current = true;
    setStatus('connecting');
    console.log('Connecting to WebSocket:', url);

    try {
      const ws = new WebSocket(url);

      ws.onopen = () => {
        console.log('WebSocket connected');
        isConnectingRef.current = false;
        setStatus('connected');
        onConnectRef.current?.();
      };

      ws.onmessage = (event) => {
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
        console.error('WebSocket error:', error);
        isConnectingRef.current = false;
        setStatus('error');
        onErrorRef.current?.(error);
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        isConnectingRef.current = false;
        setStatus('disconnected');
        onDisconnectRef.current?.();

        // Attempt to reconnect if enabled
        if (shouldReconnectRef.current && autoReconnect) {
          reconnectTimeoutRef.current = window.setTimeout(() => {
            console.log('Attempting to reconnect...');
            connect();
          }, reconnectInterval);
        }
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      isConnectingRef.current = false;
      setStatus('error');
    }
  }, [url, autoReconnect, reconnectInterval]);

  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;
    isConnectingRef.current = false;
    
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  // Handle URL changes - disconnect from old URL before connecting to new one
  const urlRef = useRef(url);
  useEffect(() => {
    if (urlRef.current !== url && wsRef.current) {
      // URL changed, disconnect from old connection
      disconnect();
    }
    urlRef.current = url;
  }, [url, disconnect]);

  // Auto-connect on mount
  useEffect(() => {
    isMountedRef.current = true;
    shouldReconnectRef.current = autoReconnect;
    
    // Only connect if we don't already have a connection
    if (!wsRef.current || wsRef.current.readyState === WebSocket.CLOSED) {
      connect();
    }

    return () => {
      // Mark as unmounted
      isMountedRef.current = false;
      shouldReconnectRef.current = false;
      
      // Clear any pending reconnects
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      
      // Close WebSocket gracefully, but only if it's not currently connecting
      // This prevents "closed before connection established" errors in React Strict Mode
      if (wsRef.current) {
        const ws = wsRef.current;
        if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CLOSED) {
          ws.close();
        } else if (ws.readyState === WebSocket.CONNECTING) {
          // Wait for connection to open before closing to avoid errors
          const handleOpen = () => {
            if (!isMountedRef.current) {
              ws.close();
            }
          };
          const handleError = () => {
            // Connection failed, no need to close
          };
          ws.addEventListener('open', handleOpen, { once: true });
          ws.addEventListener('error', handleError, { once: true });
        }
        wsRef.current = null;
      }
    };
  }, [connect, autoReconnect]);

  return {
    status,
    connect,
    disconnect,
  };
}

