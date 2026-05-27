import { describe, expect, it } from 'vitest';
import {
  isCollabSandboxPath,
  isCollaborateCommand,
  trimWorkspaceContext,
} from './outboundChatMetadata';
import type { WorkspaceContext } from './workspaceContext';

describe('isCollabSandboxPath', () => {
  it('detects collaboration sandboxes under ~/.neural-junkie/collaborations', () => {
    expect(
      isCollabSandboxPath(
        '/Users/me/.neural-junkie/collaborations/71bc548f-da3e-4485-834a-b6fc7ddbfa15'
      )
    ).toBe(true);
    expect(
      isCollabSandboxPath(
        '/Users/me/.neural-junkie/collaborations/reviews/3ec2d77e/plan.md'
      )
    ).toBe(true);
  });

  it('allows normal project paths', () => {
    expect(isCollabSandboxPath('/Users/me/development/sandbox')).toBe(false);
  });
});

describe('isCollaborateCommand', () => {
  it('matches /collaborate with flags', () => {
    expect(isCollaborateCommand('/collaborate @A @B goal')).toBe(true);
    expect(isCollaborateCommand('/collaborate --workspace @A @B goal')).toBe(true);
    expect(isCollaborateCommand('/runbook @A goal')).toBe(false);
  });
});

describe('trimWorkspaceContext', () => {
  const full: WorkspaceContext = {
    workspace_name: 'sandbox',
    workspace_path: '/proj',
    file_tree: 'src/\n  main.go',
    open_files: [
      { path: '/proj/src/main.go', language: 'go', content: 'package main\n', is_active: false },
      { path: '/proj/rfc.md', language: 'markdown', content: '# RFC\n', is_active: true },
    ],
  };

  it('focus without path refs includes active tab when activeTabPath set', () => {
    const trimmed = trimWorkspaceContext(
      'focus',
      full,
      'I have a new document open, can you review?',
      '/proj/rfc.md'
    );
    expect(trimmed?.open_files).toHaveLength(1);
    expect(trimmed?.open_files?.[0]?.path).toBe('/proj/rfc.md');
    expect(trimmed?.open_files?.[0]?.content).toContain('# RFC');
  });

  it('hint strips open file bodies', () => {
    const trimmed = trimWorkspaceContext('hint', full, 'hello', '/proj/rfc.md');
    expect(trimmed?.open_files).toEqual([]);
    expect(trimmed?.workspace_name).toBe('sandbox');
  });

  it('preserves compact scan-summary context when trimming', () => {
    const trimmed = trimWorkspaceContext(
      'focus',
      {
        ...full,
        scan_summary: {
          summary_dir: 'scan-1',
          wells_count: 96,
          analytes: ['IL-6'],
          note: 'metadata only',
        },
      },
      'do you see issues with the image I have open?',
      '/proj/rfc.md'
    );
    expect(trimmed?.scan_summary?.summary_dir).toBe('scan-1');
    expect(trimmed?.scan_summary?.note).toContain('metadata');
  });
});
