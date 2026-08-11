import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DocumentArtifactRenderer } from './DocumentArtifactRenderer';
import type { NeuralCanvasArtifact } from './types';

const artifact = (data: unknown): NeuralCanvasArtifact => ({
  id: 'a-1',
  title: 'Canvas',
  api_version: '1',
  media_type: 'application/vnd.neural-junkie.document+json',
  renderer_id: 'nj.document',
  data,
});

describe('DocumentArtifactRenderer', () => {
  it('renders an unknown block without crashing', () => {
    render(
      <DocumentArtifactRenderer
        artifact={artifact({
          schema_version: 1,
          blocks: [{ type: 'widget', text: 'nope' }],
        })}
      />,
    );
    expect(screen.getByText(/Unknown block: widget/)).toBeTruthy();
  });

  it('unwraps a markdown string into hosted blocks', () => {
    render(
      <DocumentArtifactRenderer
        artifact={artifact('# Hello\n\n- one\n')}
      />,
    );
    expect(screen.getByRole('heading', { name: 'Hello' })).toBeTruthy();
    expect(screen.getByText('one')).toBeTruthy();
  });
});
