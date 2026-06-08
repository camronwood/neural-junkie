import { invoke } from '@tauri-apps/api/tauri';
import { isTauriRuntime } from './promptAttachments';

export interface OllamaRuntimeStatus {
  installed: boolean;
  bundled?: boolean;
  running: boolean;
  managed?: boolean;
  autoInstallSupported?: boolean;
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
      autoInstallSupported: data.auto_install_supported,
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

async function readOllamaInstallSSE(
  resp: Response,
  onProgress: (message: string) => void,
): Promise<void> {
  const reader = resp.body?.getReader();
  if (!reader) {
    throw new Error('No response body');
  }
  const decoder = new TextDecoder();
  let buffer = '';
  let error: string | null = null;

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split('\n\n');
    buffer = parts.pop() ?? '';
    for (const chunk of parts) {
      for (const line of chunk.split('\n')) {
        if (!line.startsWith('data: ')) continue;
        const payload = line.slice(6).trim();
        if (!payload) continue;
        if (payload.startsWith('ERROR: ')) {
          error = payload.slice(7);
          onProgress(error);
          continue;
        }
        if (payload === 'DONE') {
          onProgress('Install complete');
          continue;
        }
        onProgress(payload);
      }
    }
  }

  if (error) {
    throw new Error(error);
  }
}

/** Download and install system Ollama via the hub (macOS/Linux). Skips when bundled. */
export async function installOllamaRuntime(
  serverAddr: string,
  onProgress: (message: string) => void,
): Promise<void> {
  const resp = await fetch(`${serverAddr}/api/ollama/install`, { method: 'POST' });
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(text || 'Failed to install Ollama');
  }
  await readOllamaInstallSSE(resp, onProgress);
}
