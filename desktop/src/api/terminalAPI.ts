import { invoke } from '@tauri-apps/api/tauri';
import { listen } from '@tauri-apps/api/event';

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

export class TerminalAPI {
  private eventListeners: (() => void)[] = [];

  // ── PTY session methods ───────────────────────────────────────────

  async createPtySession(
    id: string,
    cwd?: string,
    cols?: number,
    rows?: number
  ): Promise<void> {
    await invoke('create_pty_session', { id, cwd: cwd ?? null, cols: cols ?? null, rows: rows ?? null });
  }

  async writePtySession(id: string, data: string): Promise<void> {
    await invoke('write_pty_session', { id, data });
  }

  async resizePtySession(id: string, cols: number, rows: number): Promise<void> {
    await invoke('resize_pty_session', { id, cols, rows });
  }

  async closePtySession(id: string): Promise<void> {
    await invoke('close_pty_session', { id });
  }

  async onPtyOutput(callback: (payload: PtyOutputPayload) => void): Promise<() => void> {
    const unlisten = await listen<PtyOutputPayload>('pty-output', (event) => {
      callback(event.payload);
    });
    this.eventListeners.push(unlisten);
    return unlisten;
  }

  // ── One-off command execution (suggestions) ───────────────────────

  async executeCommand(
    command: string,
    workingDir?: string
  ): Promise<CommandResult> {
    return invoke<CommandResult>('execute_command', {
      command,
      workingDir: workingDir ?? null,
    });
  }

  cleanup(): void {
    this.eventListeners.forEach((unlisten) => unlisten());
    this.eventListeners = [];
  }
}

export const terminalAPI = new TerminalAPI();
