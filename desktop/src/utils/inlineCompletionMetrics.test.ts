import { describe, expect, it, beforeEach } from 'vitest';
import {
  RecordInlineCompletionAccept,
  RecordInlineCompletionDismiss,
  RecordInlineCompletionPartialAccept,
  RecordInlineCompletionShown,
  getSnapshot,
  resetInlineCompletionMetrics,
} from './inlineCompletionMetrics';

describe('inlineCompletionMetrics', () => {
  beforeEach(() => {
    resetInlineCompletionMetrics();
  });

  it('tracks accept/dismiss/partial/shown via getSnapshot', () => {
    RecordInlineCompletionShown();
    RecordInlineCompletionAccept();
    RecordInlineCompletionDismiss();
    RecordInlineCompletionPartialAccept();
    expect(getSnapshot()).toEqual({
      accepts: 1,
      dismisses: 1,
      partialAccepts: 1,
      shown: 1,
    });
  });
});
