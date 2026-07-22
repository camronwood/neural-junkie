import { describe, expect, it } from 'vitest';
import { buildHumanOutboundMetadata } from './outboundChatMetadata';
import {
  hasBareWorkspaceDirectiveOnly,
  hasContentDeliverySignals,
  hasErrorLogFollowUpSignals,
  hasFileExportSignals,
  hasImplementationContinuationSignals,
  hasImplementationRequestSignals,
  hasImplementationStatusCheckSignals,
  hasPriorReferenceExportSignals,
  hasCombinedContentDeliveryExport,
  channelHasImplementationThread,
} from './implementationContinuation';

describe('implementationContinuation', () => {
  it('detects go-ahead affirmations', () => {
    expect(hasImplementationContinuationSignals('yes please go ahead')).toBe(true);
    expect(hasImplementationContinuationSignals('ok please do it now')).toBe(true);
    expect(hasImplementationContinuationSignals('approved')).toBe(true);
    expect(hasImplementationContinuationSignals('please keep going')).toBe(true);
    expect(hasImplementationContinuationSignals('what?')).toBe(false);
    expect(hasImplementationContinuationSignals('ok goahead')).toBe(true);
    expect(hasImplementationContinuationSignals('looks good')).toBe(false);
    expect(hasImplementationContinuationSignals('ok')).toBe(false);
  });

  it('detects workspace directive as implementation request', () => {
    expect(
      hasImplementationRequestSignals('use the open workspace it has all the files you need')
    ).toBe(true);
  });

  it('detects implementation request phrases', () => {
    expect(
      hasImplementationRequestSignals(
        'yesterday we were working on adding a settings modal for font size and themes dark/light, pick up where we left off'
      )
    ).toBe(true);
    expect(hasImplementationRequestSignals('hello there')).toBe(false);
    expect(hasImplementationRequestSignals('the app is not booting up can you fix it?')).toBe(true);
  });

  it('detects implementation status checks', () => {
    expect(hasImplementationStatusCheckSignals('is it fixed?')).toBe(true);
    expect(hasImplementationStatusCheckSignals('does it work now')).toBe(true);
    expect(hasImplementationStatusCheckSignals('still broken')).toBe(true);
    expect(hasImplementationStatusCheckSignals('hello')).toBe(false);
  });

  it('detects bare workspace directives', () => {
    expect(hasBareWorkspaceDirectiveOnly('use the workspace')).toBe(true);
    expect(hasBareWorkspaceDirectiveOnly('can you use the workspace for this?')).toBe(true);
    expect(hasBareWorkspaceDirectiveOnly('use the workspace to implement dark mode')).toBe(false);
  });

  it('detects content delivery tasks', () => {
    expect(hasContentDeliverySignals('Can you create a linkedin article about this app?')).toBe(
      true
    );
    expect(hasContentDeliverySignals('use the workspace')).toBe(false);
  });

  it('detects file export signals', () => {
    expect(hasFileExportSignals('store that artical in a markdown file')).toBe(true);
    expect(hasFileExportSignals('please create that file new-artical-test.md')).toBe(true);
    expect(hasFileExportSignals('fill the file with the article')).toBe(true);
    expect(hasFileExportSignals('can you save that artical to a file called nj-artical-1')).toBe(true);
    expect(hasFileExportSignals('write me a linkedin article')).toBe(false);
  });

  it('detects prior-reference export signals', () => {
    expect(
      hasPriorReferenceExportSignals(
        'the artical you wrote a few messages back please save it to a markdown file'
      )
    ).toBe(true);
    expect(hasPriorReferenceExportSignals('write me a linkedin article')).toBe(false);
  });

  it('detects combined content delivery and export in one message', () => {
    expect(
      hasCombinedContentDeliveryExport(
        'Write a LinkedIn article about the app and save the file to the workspace root'
      )
    ).toBe(true);
  });

  it('detects error-log follow-up signals', () => {
    expect(
      hasErrorLogFollowUpSignals(
        'I am still getting this: VITE v5.4.21 ready\nWarn Waiting for your frontend dev server'
      )
    ).toBe(true);
    expect(
      hasErrorLogFollowUpSignals(
        'also got this:\nAre they installed?\nat file:///Users/me/proj/node_modules/vite/dist/node/chunks/dep.js:50669:15'
      )
    ).toBe(true);
    expect(hasErrorLogFollowUpSignals('thoughts on lunch?')).toBe(false);
  });

  it('detects implementation thread history in channel messages', () => {
    expect(
      channelHasImplementationThread([
        { type: 'question', metadata: { implementation_session: true } },
      ])
    ).toBe(true);
    expect(
      channelHasImplementationThread([
        {
          type: 'chat',
          metadata: {
            implementation_session_outcome: { verify_failed: true },
          },
        },
      ])
    ).toBe(true);
    expect(channelHasImplementationThread([{ type: 'chat', metadata: {} }])).toBe(false);
  });
});

describe('buildHumanOutboundMetadata continuation', () => {
  it('does not force scope none on chat mode when workspace is always', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message: 'thoughts on lunch?',
      channel: 'general',
      channelType: 'public',
    });
    expect(meta?.context_scope).toBe('full');
  });

  it('does not infer code mode from implementation wording', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message:
        'adding a settings modal with font size and themes dark/light, pick up where we left off',
      channel: 'dm-camron-frontendengineer',
      channelType: 'dm',
    });
    expect(meta?.conversation_mode).toBeUndefined();
    expect(meta?.context_scope).toBe('full');
    expect(meta?.implementation_session).toBeUndefined();
  });

  it('does not infer code mode from a workspace directive', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message: 'use the open workspace it has all the files you need',
      channel: 'dm-camron-frontendengineer',
      channelType: 'dm',
    });
    expect(meta?.conversation_mode).toBeUndefined();
  });

  it('uses a stable auto workspace hint regardless of content wording', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message: 'Can you write me an article about the app that is in the workspace?',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
    });
    expect(meta?.context_scope).toBe('hint');
    expect(meta?.context_scope_reason).toBe('workspace mode auto');
  });

  it('content delivery on general uses hint scope without file tree', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message:
        'Can you write me a LinkedIn artical about the app in the workspace and save the file to the root?',
      channel: 'general',
      channelType: 'public',
    });
    expect(meta?.context_scope).toBe('hint');
    expect(meta?.context_scope_reason).toBe('workspace mode auto');
    const ws = meta?.workspace_context as { file_tree?: string } | undefined;
    expect(ws?.file_tree ?? '').toBe('');
  });

  it('does not expand context from workspace-access phrasing', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message: 'you have workspace access',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
    });
    expect(meta?.context_scope).toBe('hint');
  });

  it('explicit export editor_mode forces code scope', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message: 'please save it',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
      composerMetadata: { editor_mode: 'export', implementation_session: true },
    });
    expect(meta?.conversation_mode).toBeUndefined();
    expect(['outline', 'focus']).toContain(meta?.context_scope);
    expect(meta?.composer_mode).toBe('export');
    expect((meta?.turn_governance as { can_run_impl_session?: boolean })?.can_run_impl_session).toBe(
      true
    );
    expect(meta?.can_run_impl_session).toBeUndefined();
  });
});
