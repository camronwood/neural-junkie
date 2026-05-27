import { describe, expect, it } from 'vitest';
import { formatCommandOutputContent } from './postCommandOutput';
import type { CommandResult } from '../api/terminalAPI';

describe('formatCommandOutputContent', () => {
  it('includes command, exit code, and streams', () => {
    const result: CommandResult = {
      id: '1',
      command: 'ls -la',
      exit_code: 0,
      stdout: 'file.txt\n',
      stderr: '',
      duration_ms: 10,
      success: true,
    };
    const text = formatCommandOutputContent(result, 'Cursor');
    expect(text).toContain('@Cursor');
    expect(text).toContain('ls -la');
    expect(text).toContain('Exit code: 0');
    expect(text).toContain('file.txt');
  });
});
