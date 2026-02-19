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

export interface CommandRequest {
  command: string;
  working_dir?: string;
}

export class TerminalAPI {
  private eventListeners: (() => void)[] = [];

  /**
   * Execute a shell command
   */
  async executeCommand(
    command: string,
    workingDir?: string
  ): Promise<CommandResult> {
    try {
      const result = await invoke<CommandResult>('execute_command', {
        command,
        workingDir,
      });
      return result;
    } catch (error) {
      throw new Error(`Failed to execute command: ${error}`);
    }
  }

  /**
   * Start a persistent shell session
   */
  async startShellSession(): Promise<void> {
    try {
      await invoke('start_shell_session');
    } catch (error) {
      throw new Error(`Failed to start shell session: ${error}`);
    }
  }

  /**
   * Execute a command in the active shell session
   */
  async executeInSession(command: string): Promise<void> {
    try {
      await invoke('execute_in_session', { command });
    } catch (error) {
      throw new Error(`Failed to execute command in session: ${error}`);
    }
  }

  /**
   * Get current working directory of the shell session
   */
  async getCurrentWorkingDir(): Promise<string> {
    try {
      return await invoke<string>('get_session_cwd');
    } catch (error) {
      throw new Error(`Failed to get working directory: ${error}`);
    }
  }

  /**
   * Listen for command execution events
   */
  async onCommandExecuted(callback: (result: CommandResult) => void): Promise<() => void> {
    const unlisten = await listen<CommandResult>('command-executed', (event) => {
      callback(event.payload);
    });

    this.eventListeners.push(unlisten);
    return unlisten;
  }

  /**
   * Listen for shell output events
   */
  async onShellOutput(callback: (output: string) => void): Promise<() => void> {
    const unlisten = await listen<string>('shell-output', (event) => {
      callback(event.payload);
    });

    this.eventListeners.push(unlisten);
    return unlisten;
  }

  /**
   * Listen for shell error events
   */
  async onShellError(callback: (error: string) => void): Promise<() => void> {
    const unlisten = await listen<string>('shell-error', (event) => {
      callback(event.payload);
    });

    this.eventListeners.push(unlisten);
    return unlisten;
  }

  /**
   * Clean up all event listeners
   */
  cleanup(): void {
    this.eventListeners.forEach((unlisten) => unlisten());
    this.eventListeners = [];
  }
}

export const terminalAPI = new TerminalAPI();
