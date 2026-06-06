import { describe, expect, it } from 'vitest';
import { hasCodeReviewSignals } from './codeReviewSignals';
import { buildImplementationSessionMetadata } from './ideComposer';

describe('codeReviewSignals', () => {
  it('detects project code review', () => {
    expect(hasCodeReviewSignals('code review this project please')).toBe(true);
    expect(hasCodeReviewSignals('Can you review the code in the workspace?')).toBe(true);
    expect(hasCodeReviewSignals('please implement themes')).toBe(false);
    expect(hasCodeReviewSignals('can you review the code for issues?')).toBe(false);
  });
});

describe('buildImplementationSessionMetadata code review', () => {
  it('does not set implementation_session for project code review', () => {
    const metadata = buildImplementationSessionMetadata({
      content: 'code review this project: /Users/me/app',
      agents: [{ name: 'FrontendEngineer', type: 'frontend' } as never],
      activeTab: null,
      editorAgentMode: 'agent',
      editorAgentTrust: 'interactive',
    });
    expect(metadata.implementation_session).toBeUndefined();
  });

  it('does not set implementation_session for workspace code review', () => {
    const metadata = buildImplementationSessionMetadata({
      content: 'Can you review the code in the workspace?',
      agents: [{ name: 'CodeReviewer', type: 'code-review' } as never],
      activeTab: null,
      editorAgentMode: 'agent',
      editorAgentTrust: 'interactive',
    });
    expect(metadata.implementation_session).toBeUndefined();
  });
});
