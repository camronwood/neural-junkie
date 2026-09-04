import { useSyncExternalStore } from 'react';
import type { ConnectionStatus } from '../hooks/useWebSocket';

export type WsDiagnostic = {
  updatedAt: number;
  url: string;
  origin: string;
  enabled: boolean;
  hasSession: boolean;
  status: ConnectionStatus;
  lastError: string | null;
  lastCloseCode: number | null;
  lastCloseReason: string | null;
  nativeProbe: string | null;
};

type WsDiagnosticListener = () => void;

const listeners = new Set<WsDiagnosticListener>();

let diagnostic: WsDiagnostic = {
  updatedAt: 0,
  url: '',
  origin: typeof window !== 'undefined' ? window.location.origin : '',
  enabled: true,
  hasSession: false,
  status: 'disconnected',
  lastError: null,
  lastCloseCode: null,
  lastCloseReason: null,
  nativeProbe: null,
};

function notify(): void {
  for (const listener of listeners) {
    listener();
  }
}

export function sanitizeWsUrl(url: string): string {
  try {
    const u = new URL(url);
    if (u.searchParams.has('nj_session')) {
      u.searchParams.set('nj_session', '…');
    }
    if (u.searchParams.has('hub_token')) {
      u.searchParams.set('hub_token', '…');
    }
    return u.toString();
  } catch {
    return url;
  }
}

export function updateWsDiagnostic(patch: Partial<WsDiagnostic>): void {
  diagnostic = { ...diagnostic, ...patch, updatedAt: Date.now() };
  notify();
}

export function getWsDiagnostic(): WsDiagnostic {
  return diagnostic;
}

export function subscribeWsDiagnostics(listener: WsDiagnosticListener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function useWsDiagnostics(): WsDiagnostic {
  return useSyncExternalStore(subscribeWsDiagnostics, getWsDiagnostic, getWsDiagnostic);
}
