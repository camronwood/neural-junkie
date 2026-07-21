import { ArtifactRendererHost } from './registry';
import type { ArtifactCardProps } from './types';

export function ArtifactCard({
  artifact,
  className = '',
  onOpen,
}: ArtifactCardProps) {
  return (
    <article
      className={`overflow-hidden rounded-lg border border-slate-700 bg-slate-900 text-slate-100 ${className}`}
    >
      <div className="flex items-center gap-2 border-b border-slate-800 px-3 py-2">
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-sm font-medium">{artifact.title}</h3>
          <p className="truncate text-[10px] text-slate-500">{artifact.media_type}</p>
        </div>
        {artifact.revision !== undefined && (
          <span className="rounded bg-slate-800 px-1.5 py-0.5 text-[10px] text-slate-400">
            r{artifact.revision}
          </span>
        )}
      </div>
      <div className="pointer-events-none max-h-52 overflow-hidden p-3" aria-hidden="true">
        <ArtifactRendererHost artifact={artifact} compact />
      </div>
      {onOpen && (
        <button
          type="button"
          onClick={() => onOpen(artifact)}
          className="w-full border-t border-slate-800 px-3 py-2 text-left text-xs font-medium text-violet-300 hover:bg-slate-800"
        >
          Open in Neural Canvas
        </button>
      )}
    </article>
  );
}
