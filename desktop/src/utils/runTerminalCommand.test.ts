import { beforeEach, describe, expect, it, vi } from 'vitest';
import { isLongRunningShellCommand, runAgentTerminalCommand } from './runTerminalCommand';

vi.mock('../api/terminalAPI', () => ({
  terminalAPI: {
    writePtySession: vi.fn(),
    executeCommand: vi.fn(),
  },
}));

vi.mock('../stores/terminalStore', () => ({
  useTerminalStore: {
    getState: () => ({
      activeTabId: 'tab-1',
      alignActiveTabCwd: vi.fn(),
      setPanelOpen: vi.fn(),
    }),
  },
}));

vi.mock('./postCommandOutput', () => ({
  postCommandOutputToHub: vi.fn(async () => undefined),
}));

import { terminalAPI } from '../api/terminalAPI';

describe('isLongRunningShellCommand', () => {
  it('detects common dev servers', () => {
    expect(isLongRunningShellCommand('tauri dev')).toBe(true);
    expect(isLongRunningShellCommand('npm run tauri')).toBe(true);
    expect(isLongRunningShellCommand('npm run tauri dev')).toBe(true);
    expect(isLongRunningShellCommand('npm run dev')).toBe(true);
    expect(isLongRunningShellCommand('cargo run')).toBe(true);
  });

  it('rejects finite commands', () => {
    expect(isLongRunningShellCommand('npm run build')).toBe(false);
    expect(isLongRunningShellCommand('go test ./...')).toBe(false);
    expect(isLongRunningShellCommand('ls -la')).toBe(false);
  });
});

describe('runAgentTerminalCommand security', () => {
  beforeEach(() => {
    vi.mocked(terminalAPI.writePtySession).mockReset();
    vi.mocked(terminalAPI.executeCommand).mockReset();
  });

  it('does not PTY-fallback when execute_command security-denies', async () => {
    vi.mocked(terminalAPI.executeCommand).mockRejectedValue(
      new Error('command not allowed: rm -rf /')
    );
    await runAgentTerminalCommand(
      {
        id: 's1',
        command: 'rm -rf /',
        cwd: '/tmp',
        agent_name: 'Agent',
        message_id: 'm1',
      } as never,
      {
        channel: 'general',
        api: {} as never,
        mirrorInPty: true,
      }
    );
    expect(terminalAPI.writePtySession).not.toHaveBeenCalled();
  });
});
