import { describe, expect, it } from 'vitest';
import { getArtifactReference } from './protocol';

describe('getArtifactReference', () => {
  it('parses a compact typed reference', () => {
    expect(getArtifactReference({
      artifact_ref: { id: 'a-1', renderer_id: 'nj.chart', revision: 3 },
    })).toEqual({ id: 'a-1', renderer_id: 'nj.chart', revision: 3 });
  });

  it('rejects malformed metadata', () => {
    expect(getArtifactReference({ artifact_ref: { title: 'missing id' } })).toBeNull();
    expect(getArtifactReference()).toBeNull();
  });
});
