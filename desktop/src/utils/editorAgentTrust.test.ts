import { describe, expect, it } from 'vitest';
import { resolveEditorAgentTrust } from './editorAgentTrust';

describe('resolveEditorAgentTrust', () => {
  it('forces auto_apply_edits in agent mode even when settings prefer interactive', () => {
    expect(
      resolveEditorAgentTrust(
        { editorAgentTrust: 'interactive', editorAgentMode: 'agent' } as never,
        'agent'
      )
    ).toBe('auto_apply_edits');
  });

  it('uses interactive for ask and plan modes', () => {
    expect(resolveEditorAgentTrust({ editorAgentMode: 'ask' } as never, 'ask')).toBe('interactive');
    expect(resolveEditorAgentTrust({ editorAgentMode: 'plan' } as never, 'plan')).toBe('interactive');
  });

  it('uses auto_apply_edits for export mode', () => {
    expect(resolveEditorAgentTrust({ editorAgentMode: 'export' } as never, 'export')).toBe(
      'auto_apply_edits'
    );
  });
});
