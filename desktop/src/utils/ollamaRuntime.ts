import { invoke } from '@tauri-apps/api/tauri';
import { isTauriRuntime } from './promptAttachments';

export interface OllamaRuntimeStatus {
  installed: boolean;
  bundled?: boolean;
  running: boolean;
  managed?: boolean;
  version?: string;
  path?: string;
}

interface TauriOllamaRuntimeStatus {
  installed: boolean;
  bundled: boolean;
  running: boolean;
  managed: boolean;
  version?: string | null;
}

async function fetchHubOllamaStatus(serverAddr: string): Promise<OllamaRuntimeStatus | null> {
  try {
    const resp = await fetch(`${serverAddr}/api/ollama/install-status`);
    if (!resp.ok) return null;
    const data = await resp.json();
    return {
      installed: Boolean(data.installed),
      bundled: data.bundled,
      running: Boolean(data.running),
      version: data.version,
      path: data.path,
    };
  } catch {
    return null;
  }
}

async function fetchTauriOllamaStatus(): Promise<TauriOllamaRuntimeStatus | null> {
  if (!isTauriRuntime()) return null;
  try {
    return await invoke<TauriOllamaRuntimeStatus>('get_ollama_runtime_status');
  } catch {
    return null;
  }
}

/** Hub is authoritative for dev (`make start-all`); Tauri augments bundled/managed state. */
export async function fetchOllamaRuntimeStatus(serverAddr: string): Promise<OllamaRuntimeStatus> {
  const [hub, tauri] = await Promise.all([
    fetchHubOllamaStatus(serverAddr),
    fetchTauriOllamaStatus(),
  ]);

  if (tauri?.bundled) {
    return {
      installed: true,
      bundled: true,
      running: tauri.running || hub?.running === true,
      managed: tauri.managed,
      version: tauri.version ?? hub?.version,
      path: hub?.path,
    };
  }

  if (hub) {
    return {
      ...hub,
      installed: hub.installed || hub.running,
    };
  }

  if (tauri) {
    return {
      installed: tauri.installed || tauri.running,
      bundled: false,
      running: tauri.running,
      managed: tauri.managed,
      version: tauri.version ?? undefined,
    };
  }

  return { installed: false, running: false };
}

async function invokeTauriIfBundled(
  action: 'start_bundled_ollama' | 'stop_bundled_ollama' | 'restart_bundled_ollama',
): Promise<boolean> {
  const tauri = await fetchTauriOllamaStatus();
  if (!tauri?.bundled) return false;
  try {
    await invoke(action);
    return true;
  } catch {
    return false;
  }
}

export async function startOllamaRuntime(serverAddr: string): Promise<void> {
  if (await invokeTauriIfBundled('start_bundled_ollama')) return;

  const resp = await fetch(`${serverAddr}/api/ollama/start`, { method: 'POST' });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(text || 'Failed to start Ollama');
  }
}

export async function stopOllamaRuntime(serverAddr: string): Promise<void> {
  if (await invokeTauriIfBundled('stop_bundled_ollama')) return;

  const resp = await fetch(`${serverAddr}/api/ollama/stop`, { method: 'POST' });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(text || 'Failed to stop Ollama');
  }
}

export async function restartOllamaRuntime(serverAddr: string): Promise<void> {
  if (await invokeTauriIfBundled('restart_bundled_ollama')) return;

  await stopOllamaRuntime(serverAddr);
  await new Promise((r) => setTimeout(r, 500));
  await startOllamaRuntime(serverAddr);
}
