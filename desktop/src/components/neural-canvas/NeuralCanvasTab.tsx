import { useCallback, useEffect, useState } from 'react';
import { ChatAPI } from '../../api/chatAPI';
import type { StoredArtifact } from '../../types/protocol';
import { ArtifactCard } from './ArtifactCard';
import { NeuralCanvasWorkbench } from './NeuralCanvasWorkbench';
import { storedArtifactToCanvas } from './types';

const api = new ChatAPI();

export interface NeuralCanvasTabProps {
  artifactId: string;
  workspaceId?: string;
  onOpenArtifact: (artifact: StoredArtifact) => void;
}

export function NeuralCanvasTab({
  artifactId,
  workspaceId = '',
  onOpenArtifact,
}: NeuralCanvasTabProps) {
  const [artifact, setArtifact] = useState<StoredArtifact | null>(null);
  const [artifacts, setArtifacts] = useState<StoredArtifact[]>([]);
  const [revisionCount, setRevisionCount] = useState(1);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const loadLibrary = useCallback(async () => {
    setLoading(true);
    try {
      setArtifacts(await api.fetchArtifacts(workspaceId ? { workspace_id: workspaceId } : {}));
      setError('');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Failed to load artifacts');
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  const loadArtifact = useCallback(async () => {
    setLoading(true);
    try {
      const [current, revisions] = await Promise.all([
        api.fetchArtifact(artifactId),
        api.fetchArtifactRevisions(artifactId),
      ]);
      setArtifact(current);
      setRevisionCount(Math.max(1, revisions.length));
      setError('');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Failed to load artifact');
    } finally {
      setLoading(false);
    }
  }, [artifactId]);

  useEffect(() => {
    if (artifactId === '__library__') void loadLibrary();
    else void loadArtifact();
  }, [artifactId, loadArtifact, loadLibrary]);

  if (loading) {
    return <div className="flex h-full items-center justify-center text-sm text-slack-textMuted">Loading Neural Canvas…</div>;
  }
  if (error) {
    return (
      <div className="p-5 text-sm text-red-300">
        <p>{error}</p>
        <button type="button" className="mt-3 rounded border border-slack-border px-3 py-1.5" onClick={() => void (artifactId === '__library__' ? loadLibrary() : loadArtifact())}>
          Retry
        </button>
      </div>
    );
  }
  if (artifactId === '__library__') {
    return (
      <section className="h-full overflow-auto bg-slack-bg p-5 text-slack-text">
        <div className="mb-5 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">Neural Canvas</h2>
            <p className="text-xs text-slack-textMuted">Durable reports, diagrams, charts, timelines, and workbench artifacts</p>
          </div>
          <button type="button" className="rounded border border-slack-border px-3 py-1.5 text-xs" onClick={() => void loadLibrary()}>
            Refresh
          </button>
        </div>
        {artifacts.length === 0 ? (
          <p className="text-sm text-slack-textMuted">No artifacts have been created for this scope.</p>
        ) : (
          <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
            {artifacts.map((item) => (
              <ArtifactCard
                key={item.id}
                artifact={storedArtifactToCanvas(item)}
                onOpen={() => onOpenArtifact(item)}
              />
            ))}
          </div>
        )}
      </section>
    );
  }
  if (!artifact) return null;
  const canvas = storedArtifactToCanvas(artifact, revisionCount);
  return (
    <div className="relative h-full">
      <div className="absolute right-4 top-16 z-10 flex gap-2">
        <button
          type="button"
          className="rounded border border-slack-border bg-slack-bg px-2 py-1 text-xs"
          onClick={async () => {
            const duplicate = await api.duplicateArtifact(artifact.id);
            onOpenArtifact(duplicate);
          }}
        >
          Duplicate
        </button>
        {workspaceId && (
          <button
            type="button"
            className="rounded border border-slack-border bg-slack-bg px-2 py-1 text-xs"
            onClick={async () => {
              const path = window.prompt('Workspace export path', `${artifact.id}.canvas.json`);
              if (path) await api.exportArtifact(artifact.id, workspaceId, path, artifact.links?.channelId);
            }}
          >
            Export
          </button>
        )}
        <button
          type="button"
          className="rounded border border-red-800 bg-slack-bg px-2 py-1 text-xs text-red-300"
          onClick={async () => {
            if (!window.confirm(`Delete “${artifact.title || artifact.id}”?`)) return;
            await api.deleteArtifact(artifact.id, artifact.revision);
            onOpenArtifact({ ...artifact, id: '__library__', title: 'Neural Canvas' });
          }}
        >
          Delete
        </button>
      </div>
      <NeuralCanvasWorkbench
        artifact={canvas}
        onRevisionChange={async (revision) => {
          const snapshot = await api.fetchArtifactRevision(artifact.id, revision);
          setArtifact(snapshot.artifact);
        }}
        onProvenanceClick={() => {
          const source = artifact.provenance?.map((item) => item.label || item.uri || item.kind).join('\n');
          window.alert(source || 'Provenance unavailable');
        }}
      />
    </div>
  );
}
