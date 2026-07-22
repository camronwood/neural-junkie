import { describe, expect, it } from 'vitest';
import { hasCodeReviewSignals } from './codeReviewSignals';
import { buildImplementationSessionMetadata } from './ideComposer';

describe('codeReviewSignals', () => {
  it('detects project code review (legacy quarantine helper)', () => {
    expect(hasCodeReviewSignals('code review this project please')).toBe(true);
    expect(hasCodeReviewSignals('Can you review the code in the workspace?')).toBe(true);
    expect(hasCodeReviewSignals('please implement themes')).toBe(false);
    expect(hasCodeReviewSignals('can you review the code for issues?')).toBe(false);
  });
});

describe('buildImplementationSessionMetadata code review', () => {
  it('never infers implementation_session from review wording', () => {
    const metadata = buildImplementationSessionMetadata({
      content: 'code review this project: /Users/me/app',
      agents: [{ name: 'FrontendEngineer', type: 'frontend' } as never],
      activeTab: null,
      editorAgentMode: 'agent',
      editorAgentTrust: 'interactive',
    });
    expect(metadata.implementation_session).toBeUndefined();
  });

  it('never infers implementation_session from workspace review wording', () => {
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
