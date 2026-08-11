import { describe, expect, it } from 'vitest';
import { resolveArtifactRenderer } from './registry';
import type {
  ArtifactRendererRegistration,
  NeuralCanvasArtifact,
} from './types';

const artifact = (
  overrides: Partial<NeuralCanvasArtifact> = {},
): NeuralCanvasArtifact => ({
  id: 'artifact-1',
  title: 'Artifact',
  api_version: '1',
  media_type: 'text/markdown',
  data: '# Trusted',
  ...overrides,
});

describe('resolveArtifactRenderer', () => {
  it('resolves a compatible requested renderer', () => {
    const result = resolveArtifactRenderer(artifact({ renderer_id: 'nj.markdown' }));

    expect(result.registration?.id).toBe('nj.markdown');
    expect(result.reason).toBe('requested');
  });

  it('resolves nj.document by renderer and media type', () => {
    const result = resolveArtifactRenderer(artifact({
      renderer_id: 'nj.document',
      media_type: 'application/vnd.neural-junkie.document+json',
      data: { schema_version: 1, blocks: [] },
    }));

    expect(result.registration?.id).toBe('nj.document');
    expect(result.reason).toBe('requested');
  });

  it('falls back deterministically by media type', () => {
    const result = resolveArtifactRenderer(artifact({ renderer_id: 'missing' }));

    expect(result.registration?.id).toBe('nj.markdown');
    expect(result.reason).toBe('media-fallback');
  });

  it('does not render data with an unsupported API version', () => {
    const result = resolveArtifactRenderer(artifact({ api_version: '99' }));

    expect(result.registration).toBeNull();
    expect(result.reason).toBe('unsupported-api');
  });

  it('uses priority and then registration order for stable fallback', () => {
    const component: ArtifactRendererRegistration['component'] = () => null;
    const registrations: ArtifactRendererRegistration[] = [
      { id: 'first', apiVersions: ['1'], mediaTypes: ['text/*'], component },
      { id: 'preferred', apiVersions: ['1'], mediaTypes: ['text/markdown'], component, priority: 1 },
      { id: 'also-preferred', apiVersions: ['1'], mediaTypes: ['text/markdown'], component, priority: 1 },
    ];

    const result = resolveArtifactRenderer(artifact(), registrations);

    expect(result.registration?.id).toBe('preferred');
  });
});
