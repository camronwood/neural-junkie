import { describe, expect, it } from 'vitest';
import {
  buildHumanOutboundMetadata,
  isCollabSandboxPath,
  isCollaborateCommand,
  isPersonalAssistantDmChannel,
  trimWorkspaceContext,
} from './outboundChatMetadata';
import type { WorkspaceContext } from './workspaceContext';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useEditorStore } from '../stores/editorStore';

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

  it('attaches workspace for knowledge-graph relate questions even in chat mode', () => {
    useFileExplorerStore.setState({
      workspaces: [
        {
          id: 'ciso',
          name: 'CISO',
          path: '/Users/me/CISO',
          kind: 'local',
        },
      ],
      activeWorkspaceId: 'ciso',
      fileTree: { ciso: [] },
    });
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      conversationMode: 'chat',
      message:
        "How does CISO relate to the rest of the codebase? CISO (repo) in community 'root' — degree 1, 1 neighbors",
      channel: 'dm-camron-softwarearchitect',
      channelType: 'dm',
    });
    expect(meta?.context_scope).not.toBe('none');
    expect(meta?.workspace_context).toBeDefined();
    const ws = meta?.workspace_context as { workspace_path?: string } | undefined;
    expect(ws?.workspace_path).toBe('/Users/me/CISO');
  });
});

describe('buildHumanOutboundMetadata custom expert turn context', () => {
  const writingRoute = {
    ide_route_agent_type: 'expert',
    editor_mode: 'agent',
    implementation_session: true,
  };

  it.each(['auto', 'always'] as const)(
    'keeps a Writing expert in Chat with no workspace when sharing is %s',
    (contextMode) => {
      const meta = buildHumanOutboundMetadata({
        contextMode,
        conversationMode: 'auto',
        message: 'Write a warm introduction for this quarterly update.',
        channel: 'dm-camron-writing',
        channelType: 'dm',
        composerMetadata: writingRoute,
        ideCoding: true,
      });

      expect(meta?.conversation_mode).toBe('chat');
      expect(meta?.context_scope).toBe('none');
      expect(meta?.workspace_context).toBeUndefined();
      expect(meta?.editor_mode).toBe('ask');
      expect(meta?.implementation_session).toBeUndefined();
    }
  );

  it('gives a Writing expert focused context for an explicit open-file request', () => {
    useFileExplorerStore.setState({
      workspaces: [{ id: 'proj', name: 'Project', path: '/proj', kind: 'local' }],
      activeWorkspaceId: 'proj',
      fileTree: { proj: [] },
    });
    useEditorStore.setState({
      tabs: [
        {
          id: 'draft',
          workspaceId: 'proj',
          path: '/proj/draft.md',
          content: '# Draft\n',
          language: 'markdown',
          isDirty: false,
        },
        {
          id: 'notes',
          workspaceId: 'proj',
          path: '/proj/notes.md',
          content: '# Notes\n',
          language: 'markdown',
          isDirty: false,
        },
      ],
      activeTabId: 'draft',
    });

    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      conversationMode: 'auto',
      message: 'Please proofread the file open in my editor.',
      channel: 'dm-camron-writing',
      channelType: 'dm',
      composerMetadata: writingRoute,
      ideCoding: true,
    });

    expect(meta?.conversation_mode).toBe('code');
    expect(meta?.context_scope).toBe('focus');
    const workspace = meta?.workspace_context as WorkspaceContext;
    expect(workspace.open_files.map((file) => file.path)).toEqual(['/proj/draft.md']);
  });

  it('honors explicit Code for Writing while focusing context, including workspace Off', () => {
    useFileExplorerStore.setState({
      workspaces: [{ id: 'proj', name: 'Project', path: '/proj', kind: 'local' }],
      activeWorkspaceId: 'proj',
      fileTree: { proj: [] },
    });
    useEditorStore.setState({
      tabs: [
        {
          id: 'draft',
          workspaceId: 'proj',
          path: '/proj/draft.md',
          content: '# Draft\n',
          language: 'markdown',
          isDirty: false,
        },
        {
          id: 'notes',
          workspaceId: 'proj',
          path: '/proj/notes.md',
          content: '# Notes\n',
          language: 'markdown',
          isDirty: false,
        },
      ],
      activeTabId: 'draft',
    });

    const codeMeta = buildHumanOutboundMetadata({
      contextMode: 'always',
      conversationMode: 'code',
      message: 'Help me improve this.',
      channel: 'dm-camron-writing',
      channelType: 'dm',
      composerMetadata: writingRoute,
      ideCoding: true,
    });

    expect(codeMeta?.conversation_mode).toBe('code');
    expect(codeMeta?.context_scope).toBe('focus');
    expect(codeMeta?.editor_mode).toBe('agent');
    const workspace = codeMeta?.workspace_context as WorkspaceContext;
    expect(workspace.open_files.map((file) => file.path)).toEqual(['/proj/draft.md']);

    const offMeta = buildHumanOutboundMetadata({
      contextMode: 'off',
      conversationMode: 'code',
      message: 'Help me improve this.',
      channel: 'dm-camron-writing',
      channelType: 'dm',
      composerMetadata: writingRoute,
      ideCoding: true,
    });

    expect(offMeta?.conversation_mode).toBe('code');
    expect(offMeta?.context_scope).toBe('none');
    expect(offMeta?.workspace_context).toBeUndefined();
    expect(offMeta?.editor_mode).toBe('agent');
  });

  it('preserves IDE code behavior for engineering specialists', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      conversationMode: 'auto',
      message: 'Investigate the current implementation.',
      channel: 'dm-camron-softwarearchitect',
      channelType: 'dm',
      composerMetadata: {
        ide_route_agent_type: 'architecture',
        editor_mode: 'agent',
      },
      ideCoding: true,
    });

    expect(meta?.conversation_mode).toBe('code');
    expect(meta?.context_scope).not.toBe('none');
  });
});

describe('isPersonalAssistantDmChannel', () => {
  it('matches assistant DMs only', () => {
    expect(isPersonalAssistantDmChannel('dm-camron-assistant')).toBe(true);
    expect(isPersonalAssistantDmChannel('dm-camron-softwarearchitect')).toBe(false);
  });
});
