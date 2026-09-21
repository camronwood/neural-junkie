import { describe, expect, it } from 'vitest';
import { isLongRunningShellCommand } from './runTerminalCommand';

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
