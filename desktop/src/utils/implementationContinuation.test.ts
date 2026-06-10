import { describe, expect, it } from 'vitest';
import { buildHumanOutboundMetadata } from './outboundChatMetadata';
import {
  hasBareWorkspaceDirectiveOnly,
  hasContentDeliverySignals,
  hasFileExportSignals,
  hasImplementationContinuationSignals,
  hasImplementationRequestSignals,
  hasImplementationStatusCheckSignals,
  hasPriorReferenceExportSignals,
  hasCombinedContentDeliveryExport,
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

  it('DM settings modal request uses code mode and outline/focus scope', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message:
        'adding a settings modal with font size and themes dark/light, pick up where we left off',
      channel: 'dm-camron-frontendengineer',
      channelType: 'dm',
    });
    expect(meta?.conversation_mode).toBe('code');
    expect(['outline', 'focus']).toContain(meta?.context_scope);
  });

  it('DM workspace directive uses code mode', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'always',
      message: 'use the open workspace it has all the files you need',
      channel: 'dm-camron-frontendengineer',
      channelType: 'dm',
    });
    expect(meta?.conversation_mode).toBe('code');
  });

  it('article about app attaches outline scope in auto mode', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message: 'Can you write me an article about the app that is in the workspace?',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
    });
    expect(meta?.context_scope).toBe('outline');
    expect(meta?.context_scope_reason).toContain('content delivery');
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
    expect(meta?.context_scope_reason).toContain('shared channel');
    const ws = meta?.workspace_context as { file_tree?: string } | undefined;
    expect(ws?.file_tree ?? '').toBe('');
  });

  it('workspace access affirmation attaches outline scope', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message: 'you have workspace access',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
    });
    expect(meta?.context_scope).toBe('outline');
  });

  it('explicit export editor_mode forces code scope', () => {
    const meta = buildHumanOutboundMetadata({
      contextMode: 'auto',
      message: 'please save it',
      channel: 'dm-camron-assistant',
      channelType: 'dm',
      composerMetadata: { editor_mode: 'export', implementation_session: true },
    });
    expect(meta?.conversation_mode).toBe('code');
    expect(meta?.context_scope_reason).toContain('export');
    expect(meta?.composer_mode).toBe('export');
    expect(meta?.can_run_impl_session).toBe(true);
  });
});
