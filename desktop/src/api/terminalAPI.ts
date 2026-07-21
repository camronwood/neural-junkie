import { invoke } from '@tauri-apps/api/core';
import { listen } from '@tauri-apps/api/event';
import { getHubBaseURL, hubAuthHeaders, hubSessionHeaders } from '../config/hubUrl';
import { getWorkspaceRoots } from '../utils/workspaceRoots';
import { useFileExplorerStore } from '../stores/fileExplorerStore';

export interface CommandResult {
  id: string;
  command: string;
  exit_code: number;
  stdout: string;
  stderr: string;
  duration_ms: number;
  success: boolean;
}

export interface PtyOutputPayload {
  id: string;
  data: string;
}

export interface PtySessionStatus {
  foreground_work: boolean;
}

function isRemoteWorkspace(kind?: string): boolean {
  return kind === 'ssh' || kind === 'devcontainer';
}

function hubWsUrl(path: string): string {
  const base = getHubBaseURL().replace(/^http/, 'ws');
  return `${base}${path}`;
}

export class TerminalAPI {
  private eventListeners: (() => void)[] = [];
  private remoteSockets = new Map<string, WebSocket>();

  private activeWorkspace() {
    return useFileExplorerStore.getState().getActiveWorkspace();
  }

  private useRemotePty(): boolean {
    const ws = this.activeWorkspace();
    return !!ws && isRemoteWorkspace(ws.kind);
  }

  async createPtySession(
    id: string,
    cwd?: string,
    cols?: number,
    rows?: number
  ): Promise<void> {
    if (this.useRemotePty()) {
      const ws = this.activeWorkspace();
      if (!ws) throw new Error('No active workspace');
      const url = hubWsUrl(`/api/terminal/ws?workspace=${encodeURIComponent(ws.id)}`);
      const socket = new WebSocket(url);
      await new Promise<void>((resolve, reject) => {
        socket.onopen = () => resolve();
        socket.onerror = () => reject(new Error('Remote terminal connection failed'));
      });
      socket.onmessage = (ev) => {
        const data = typeof ev.data === 'string' ? ev.data : '';
        if (data) {
          void this.emitPtyOutput(id, data);
        }
      };
      this.remoteSockets.set(id, socket);
      if (cwd && cwd !== '~') {
        socket.send(`cd ${cwd}\n`);
      }
      return;
    }
    await invoke('create_pty_session', { id, cwd: cwd ?? null, cols: cols ?? null, rows: rows ?? null });
  }

  private async emitPtyOutput(id: string, data: string) {
    const holder = this as unknown as { _ptyCallbacks?: Array<(p: PtyOutputPayload) => void> };
    for (const cb of holder._ptyCallbacks ?? []) cb({ id, data });
  }

  async writePtySession(id: string, data: string): Promise<void> {
    const remote = this.remoteSockets.get(id);
    if (remote && remote.readyState === WebSocket.OPEN) {
      remote.send(data);
      return;
    }
    await invoke('write_pty_session', { id, data });
  }

  async getPtySessionStatus(id: string): Promise<PtySessionStatus> {
    if (this.remoteSockets.has(id)) {
      // The remote PTY protocol does not expose process groups yet.
      return { foreground_work: true };
    }
    return invoke<PtySessionStatus>('get_pty_session_status', { id });
  }

  async resizePtySession(id: string, cols: number, rows: number): Promise<void> {
    const remote = this.remoteSockets.get(id);
    if (remote) {
      remote.send(JSON.stringify({ type: 'resize', cols, rows }));
      return;
    }
    await invoke('resize_pty_session', { id, cols, rows });
  }

  async closePtySession(id: string): Promise<void> {
    const remote = this.remoteSockets.get(id);
    if (remote) {
      remote.close();
      this.remoteSockets.delete(id);
      return;
    }
    await invoke('close_pty_session', { id });
  }

  async onPtyOutput(callback: (payload: PtyOutputPayload) => void): Promise<() => void> {
    const holder = this as unknown as { _ptyCallbacks?: Array<(p: PtyOutputPayload) => void> };
    if (!holder._ptyCallbacks) holder._ptyCallbacks = [];
    holder._ptyCallbacks.push(callback);

    const unlisten = await listen<PtyOutputPayload>('pty-output', (event) => {
      callback(event.payload);
    });
    this.eventListeners.push(unlisten);
    return () => {
      holder._ptyCallbacks = (holder._ptyCallbacks ?? []).filter((c) => c !== callback);
      unlisten();
    };
  }

  async executeCommand(
    command: string,
    workingDir?: string
  ): Promise<CommandResult> {
    const ws = this.activeWorkspace();
    if (ws && isRemoteWorkspace(ws.kind)) {
      const parts = command.trim().split(/\s+/);
      const bin = parts[0] ?? '';
      const args = parts.slice(1);
      const res = await fetch(`${getHubBaseURL()}/api/workspaces/remote-exec`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...hubAuthHeaders(),
          ...hubSessionHeaders(),
        },
        body: JSON.stringify({
          workspace_id: ws.id,
          command: bin,
          args,
          cwd: workingDir && workingDir !== ws.path ? '.' : '.',
        }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Remote exec failed');
      }
      const data = (await res.json()) as { stdout?: string; stderr?: string; exit_code?: number };
      return {
        id: `remote-${Date.now()}`,
        command,
        exit_code: data.exit_code ?? 0,
        stdout: data.stdout ?? '',
        stderr: data.stderr ?? '',
        duration_ms: 0,
        success: (data.exit_code ?? 0) === 0,
      };
    }
    return invoke<CommandResult>('execute_command', {
      command,
      workingDir: workingDir ?? null,
      allowedRoots: getWorkspaceRoots(),
    });
  }

  cleanup(): void {
    for (const ws of this.remoteSockets.values()) ws.close();
    this.remoteSockets.clear();
    this.eventListeners.forEach((unlisten) => unlisten());
    this.eventListeners = [];
  }
}

export const terminalAPI = new TerminalAPI();
