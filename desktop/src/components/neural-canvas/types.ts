import type { ComponentType } from 'react';
import type { StoredArtifact } from '../../types/protocol';

export interface ArtifactProvenance {
  source?: string;
  author?: string;
  model?: string;
  generated_at?: string;
  references?: readonly string[];
  [key: string]: unknown;
}

export interface NeuralCanvasArtifact<T = unknown> {
  id: string;
  title: string;
  renderer_id?: string;
  api_version: string;
  media_type: string;
  data: T;
  revision?: number;
  revision_count?: number;
  provenance?: ArtifactProvenance;
}

export interface ArtifactRendererProps<T = unknown> {
  artifact: NeuralCanvasArtifact<T>;
  compact?: boolean;
}

export interface ArtifactRendererRegistration<T = unknown> {
  id: string;
  apiVersions: readonly string[];
  mediaTypes: readonly string[];
  component: ComponentType<ArtifactRendererProps<T>>;
  priority?: number;
}

export interface RendererResolution {
  registration: ArtifactRendererRegistration | null;
  reason: 'requested' | 'media-fallback' | 'unsupported-api' | 'unsupported-media';
}

export interface NeuralCanvasWorkbenchProps {
  artifact: NeuralCanvasArtifact;
  className?: string;
  onTitleClick?: (artifact: NeuralCanvasArtifact) => void;
  onProvenanceClick?: (artifact: NeuralCanvasArtifact) => void;
  onRevisionChange?: (revision: number, artifact: NeuralCanvasArtifact) => void;
  onClose?: () => void;
}

export interface ArtifactCardProps {
  artifact: NeuralCanvasArtifact;
  className?: string;
  onOpen?: (artifact: NeuralCanvasArtifact) => void;
}

export function storedArtifactToCanvas(
  artifact: StoredArtifact,
  revisionCount?: number,
): NeuralCanvasArtifact {
  const source = artifact.provenance?.[0];
  return {
    id: artifact.id,
    title: artifact.title || artifact.id,
    renderer_id: artifact.renderer.id,
    api_version: artifact.renderer.apiVersion || '1',
    media_type: artifact.renderer.mediaType,
    data: artifact.payload,
    revision: artifact.revision,
    revision_count: revisionCount,
    provenance: source ? {
      source: source.kind,
      author: source.label,
      generated_at: artifact.updatedAt,
      references: source.uri ? [source.uri] : undefined,
      ...source.metadata,
    } : undefined,
  };
}
