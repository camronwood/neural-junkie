import { describe, expect, it } from 'vitest';
import { markdownPreviewLine } from './markdownPreview';

describe('markdownPreviewLine', () => {
  it('strips markdown headers and bullets for sidebar preview', () => {
    const raw = '### Final session summary\n\n- All tasks delivered on schedule.';
    expect(markdownPreviewLine(raw, 160)).toBe('Final session summary All tasks delivered on schedule.');
  });
});
