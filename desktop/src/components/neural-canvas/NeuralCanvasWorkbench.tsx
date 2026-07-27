import { ArtifactRendererHost } from './registry';
import type { NeuralCanvasWorkbenchProps } from './types';

function provenanceLabel(provenance: NeuralCanvasWorkbenchProps['artifact']['provenance']) {
  if (!provenance) return 'Provenance unavailable';
  return provenance.source || provenance.author || provenance.model || 'View provenance';
}

export function NeuralCanvasWorkbench({
  artifact,
  className = '',
  onTitleClick,
  onProvenanceClick,
  onRevisionChange,
  onClose,
}: NeuralCanvasWorkbenchProps) {
  const revision = Math.max(1, artifact.revision ?? 1);
  const revisionCount = Math.max(revision, artifact.revision_count ?? revision);

  return (
    <section
      aria-label={`Neural Canvas: ${artifact.title}`}
      className={`flex h-full min-h-0 flex-col bg-[#0b1220] text-slate-100 ${className}`}
    >
      <header className="flex items-center gap-3 border-b border-slate-800 bg-slate-900 px-4 py-3">
        <div className="min-w-0 flex-1">
          {onTitleClick ? (
            <button
              type="button"
              onClick={() => onTitleClick(artifact)}
              className="max-w-full truncate text-left text-base font-semibold hover:text-violet-300"
            >
              {artifact.title}
            </button>
          ) : (
            <h2 className="truncate text-base font-semibold">{artifact.title}</h2>
          )}
          <div className="mt-0.5 flex items-center gap-2 text-[11px] text-slate-400">
            <span>{artifact.media_type}</span>
            <span aria-hidden="true">·</span>
            <span>API {artifact.api_version}</span>
          </div>
        </div>

        <button
          type="button"
          disabled={!artifact.provenance || !onProvenanceClick}
          onClick={() => onProvenanceClick?.(artifact)}
          title={provenanceLabel(artifact.provenance)}
          className="max-w-48 truncate rounded border border-slate-700 px-2 py-1 text-xs text-slate-300 hover:bg-slate-800 disabled:cursor-default disabled:opacity-50"
        >
          {provenanceLabel(artifact.provenance)}
        </button>

        <div className="flex items-center rounded border border-slate-700" aria-label="Artifact revisions">
          <button
            type="button"
            aria-label="Previous revision"
            disabled={!onRevisionChange || revision <= 1}
            onClick={() => onRevisionChange?.(revision - 1, artifact)}
            className="px-2 py-1 text-sm hover:bg-slate-800 disabled:opacity-30"
          >
            ‹
          </button>
          <span className="border-x border-slate-700 px-2 py-1 text-xs tabular-nums">
            {revision} / {revisionCount}
          </span>
          <button
            type="button"
            aria-label="Next revision"
            disabled={!onRevisionChange || revision >= revisionCount}
            onClick={() => onRevisionChange?.(revision + 1, artifact)}
            className="px-2 py-1 text-sm hover:bg-slate-800 disabled:opacity-30"
          >
            ›
          </button>
        </div>

        {onClose && (
          <button
            type="button"
            aria-label="Close Neural Canvas"
            onClick={onClose}
            className="rounded p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white"
          >
            ×
          </button>
        )}
      </header>

      <div className="min-h-0 flex-1 overflow-hidden">
        <div className="h-full min-h-0">
          <ArtifactRendererHost artifact={artifact} />
        </div>
      </div>
    </section>
  );
}
