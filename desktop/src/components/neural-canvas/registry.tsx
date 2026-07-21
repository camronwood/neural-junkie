import { createElement } from 'react';
import { NeuralCanvasErrorBoundary } from './NeuralCanvasErrorBoundary';
import { BUILT_IN_RENDERERS, EmptyArtifact } from './renderers';
import type {
  ArtifactRendererRegistration,
  NeuralCanvasArtifact,
  RendererResolution,
} from './types';

function mediaMatches(accepted: string, actual: string): boolean {
  if (accepted === actual || accepted === '*/*') return true;
  if (!accepted.endsWith('/*')) return false;
  return actual.startsWith(accepted.slice(0, -1));
}

function ordered(registrations: readonly ArtifactRendererRegistration[]) {
  return registrations
    .map((registration, index) => ({ registration, index }))
    .sort((left, right) =>
      (right.registration.priority ?? 0) - (left.registration.priority ?? 0)
      || left.index - right.index)
    .map(({ registration }) => registration);
}

export function resolveArtifactRenderer(
  artifact: NeuralCanvasArtifact,
  registrations: readonly ArtifactRendererRegistration[] = BUILT_IN_RENDERERS,
): RendererResolution {
  const candidates = ordered(registrations);
  const hasApiVersion = (registration: ArtifactRendererRegistration) =>
    registration.apiVersions.includes(artifact.api_version);
  const hasMediaType = (registration: ArtifactRendererRegistration) =>
    registration.mediaTypes.some((mediaType) => mediaMatches(mediaType, artifact.media_type));

  if (artifact.renderer_id) {
    const requested = candidates.find((registration) =>
      registration.id === artifact.renderer_id
      && hasApiVersion(registration)
      && hasMediaType(registration));
    if (requested) return { registration: requested, reason: 'requested' };
  }

  const mediaFallback = candidates.find((registration) =>
    hasApiVersion(registration) && hasMediaType(registration));
  if (mediaFallback) return { registration: mediaFallback, reason: 'media-fallback' };

  const mediaExists = candidates.some(hasMediaType);
  return {
    registration: null,
    reason: mediaExists ? 'unsupported-api' : 'unsupported-media',
  };
}

export interface ArtifactRendererHostProps {
  artifact: NeuralCanvasArtifact;
  compact?: boolean;
  registrations?: readonly ArtifactRendererRegistration[];
}

export function ArtifactRendererHost({
  artifact,
  compact = false,
  registrations = BUILT_IN_RENDERERS,
}: ArtifactRendererHostProps) {
  const resolution = resolveArtifactRenderer(artifact, registrations);
  if (!resolution.registration) {
    const message = resolution.reason === 'unsupported-api'
      ? `Unsupported artifact API version: ${artifact.api_version}`
      : `Unsupported artifact media type: ${artifact.media_type}`;
    return <EmptyArtifact message={message} />;
  }

  return (
    <NeuralCanvasErrorBoundary
      key={`${artifact.id}:${artifact.revision ?? 0}:${resolution.registration.id}`}
    >
      {createElement(resolution.registration.component, { artifact, compact })}
    </NeuralCanvasErrorBoundary>
  );
}

export { BUILT_IN_RENDERERS };
