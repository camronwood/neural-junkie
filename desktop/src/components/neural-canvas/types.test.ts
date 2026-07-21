import { describe, expect, it } from 'vitest';
import { storedArtifactToCanvas } from './types';

describe('storedArtifactToCanvas', () => {
  it('maps the backend envelope into the renderer contract', () => {
    const canvas = storedArtifactToCanvas({
      schemaVersion: 1,
      id: 'a-1',
      revision: 2,
      title: 'Report',
      renderer: {
        id: 'nj.chart',
        apiVersion: '1',
        mediaType: 'application/vnd.neural-junkie.chart+json',
      },
      payload: { type: 'bar', series: [] },
      provenance: [{ kind: 'agent', label: 'Analyst' }],
      createdAt: '2026-07-21T00:00:00Z',
      updatedAt: '2026-07-21T01:00:00Z',
    }, 3);
    expect(canvas).toMatchObject({
      id: 'a-1',
      renderer_id: 'nj.chart',
      revision: 2,
      revision_count: 3,
      provenance: { source: 'agent', author: 'Analyst' },
    });
  });
});
