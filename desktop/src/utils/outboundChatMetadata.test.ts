import { describe, expect, it } from 'vitest';
import {
  buildHumanOutboundMetadata,
  isCollabSandboxPath,
  isCollaborateCommand,
  isPersonalAssistantDmChannel,
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

  it('detects project collabs deliverable folders', () => {
    expect(
      isCollabSandboxPath('/Users/me/myproject/collabs/19a9e849-2c26-4591-af05-aca853cf8054')
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

describe('buildHumanOutboundMetadata DM personal questions', () => {
  it('downgrades export/agent composer to ask for non-code DM messages', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message:
        'Can you check flight times? I need to plan a trip from St. Louis, MO to Chicago, IL.',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
      composerMetadata: {
        editor_mode: 'export',
        implementation_session: true,
      },
    });
    expect(meta?.editor_mode).toBe('ask');
    expect(meta?.composer_mode).toBe('ask');
    expect(meta?.can_run_impl_session).toBe(false);
    expect(meta?.implementation_session).toBeUndefined();
  });

  it('keeps export metadata for explicit save-to-file asks in DMs', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message: 'please save that article to docs/plan.md',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
      composerMetadata: {
        editor_mode: 'export',
        implementation_session: true,
      },
    });
    expect(meta?.editor_mode).toBe('export');
    expect(meta?.can_run_impl_session).toBe(true);
  });

  it('strips workspace context for personal assistant DM questions', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message: 'What Amtrak trains run from St. Louis to Chicago tomorrow?',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
      composerMetadata: {
        editor_mode: 'agent',
        implementation_session: true,
      },
    });
    expect(meta?.editor_mode).toBe('ask');
    expect(meta?.context_scope).toBe('none');
    expect(meta?.conversation_mode).toBe('chat');
    expect(meta?.workspace_context).toBeUndefined();
  });

  it('does not strip workspace for specialist DM error-log follow-ups', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message:
        'I am still getting this: VITE v5.4.21 ready in 121 ms\nWarn Waiting for your frontend dev server to start on http://localhost:5177/...',
      channel: 'dm-camron-softwarearchitect',
      channelType: 'dm',
      composerMetadata: { editor_mode: 'ask' },
      recentChannelMessages: [
        {
          type: 'question',
          metadata: {
            implementation_session: true,
            can_run_impl_session: true,
            conversation_mode: 'code',
          },
        },
        {
          type: 'chat',
          metadata: {
            implementation_session_complete: true,
            implementation_session_outcome: { verify_failed: true },
          },
        },
      ],
    });
    expect(meta?.context_scope_reason).not.toBe('personal assistant DM');
    expect(meta?.conversation_mode).toBe('code');
    expect(meta?.context_scope_reason).toBe('implementation thread continuation');
    expect(meta?.editor_mode).toBe('agent');
    expect(meta?.implementation_session).toBe(true);
    expect(meta?.workspace_context).toBeDefined();
  });

  it('keeps agent mode in specialist DMs for non-code messages', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message: 'What is your opinion on microservices?',
      channel: 'dm-camron-softwarearchitect',
      channelType: 'dm',
      composerMetadata: {
        editor_mode: 'agent',
        implementation_session: true,
      },
    });
    expect(meta?.editor_mode).toBe('agent');
    expect(meta?.implementation_session).toBe(true);
  });
});

describe('isPersonalAssistantDmChannel', () => {
  it('matches assistant DMs only', () => {
    expect(isPersonalAssistantDmChannel('dm-camron-assistant')).toBe(true);
    expect(isPersonalAssistantDmChannel('dm-camron-softwarearchitect')).toBe(false);
  });
});
