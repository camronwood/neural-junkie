import { LazyStore as Store } from '@tauri-apps/plugin-store';
import { DEFAULT_HUB_HTTP, normalizeHubBaseURL, setHubConnectionOverride } from '../config/hubUrl';

export type ConnectionSettings = {
  hubUrl: string;
  hubToken: string;
};

const defaultConnection: ConnectionSettings = {
  hubUrl: DEFAULT_HUB_HTTP,
  hubToken: '',
};

let cachedConnection: ConnectionSettings = { ...defaultConnection };
let loaded = false;

export function getCachedConnection(): ConnectionSettings {
  return cachedConnection;
}

export async function loadConnectionSettings(): Promise<ConnectionSettings> {
  if (loaded) return cachedConnection;
  try {
    const store = new Store('.neural-junkie-connection.dat');
    const saved = await store.get<ConnectionSettings>('connection');
    if (saved?.hubUrl?.trim()) {
      cachedConnection = {
        hubUrl: normalizeHubBaseURL(saved.hubUrl),
        hubToken: saved.hubToken ?? '',
      };
    }
  } catch {
    // browser dev without Tauri store
  }
  loaded = true;
  return cachedConnection;
}

export async function saveConnectionSettings(next: ConnectionSettings): Promise<void> {
  cachedConnection = {
    hubUrl: normalizeHubBaseURL(next.hubUrl || DEFAULT_HUB_HTTP),
    hubToken: next.hubToken.trim(),
  };
  loaded = true;
  const store = new Store('.neural-junkie-connection.dat');
  await store.set('connection', cachedConnection);
  await store.save();
  setHubConnectionOverride(cachedConnection.hubUrl, cachedConnection.hubToken);
}
