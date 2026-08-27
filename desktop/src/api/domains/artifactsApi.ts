import type { FileChange, StoredArtifact, StoredArtifactRevision } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

function guessImageMimeFromName(name: string): string {
  const lower = name.toLowerCase();
  if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg';
  if (lower.endsWith('.webp')) return 'image/webp';
  if (lower.endsWith('.gif')) return 'image/gif';
  if (lower.endsWith('.svg')) return 'image/svg+xml';
  return 'image/png';
}

/** Artifact CRUD, revisions, export, and asset HTTP surface. */
export class ArtifactsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchArtifacts(filters: {
    workspace_id?: string;
    project_id?: string;
    channel_id?: string;
    collaboration_id?: string;
    renderer_id?: string;
    kind?: string;
  } = {}): Promise<StoredArtifact[]> {
    const params = new URLSearchParams();
    Object.entries(filters).forEach(([key, value]) => {
      if (value) params.set(key, value);
    });
    const response = await this.hubFetch(`/api/artifacts${params.size ? `?${params}` : ''}`);
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async fetchArtifact(id: string): Promise<StoredArtifact> {
    const response = await this.hubFetch(`/api/artifacts/${encodeURIComponent(id)}`);
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async createArtifact(artifact: Partial<StoredArtifact>): Promise<StoredArtifact> {
    const response = await this.hubFetch('/api/artifacts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(artifact),
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async updateArtifact(artifact: StoredArtifact): Promise<StoredArtifact> {
    const response = await this.hubFetch(`/api/artifacts/${encodeURIComponent(artifact.id)}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'If-Match': String(artifact.revision),
      },
      body: JSON.stringify(artifact),
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async deleteArtifact(id: string, revision: number): Promise<void> {
    const response = await this.hubFetch(`/api/artifacts/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: { 'If-Match': String(revision) },
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
  }

  async fetchArtifactRevisions(id: string): Promise<StoredArtifactRevision[]> {
    const response = await this.hubFetch(`/api/artifacts/${encodeURIComponent(id)}/revisions`);
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async fetchArtifactRevision(id: string, revision: number): Promise<StoredArtifactRevision> {
    const response = await this.hubFetch(`/api/artifacts/${encodeURIComponent(id)}/revisions/${revision}`);
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async duplicateArtifact(id: string, newId = ''): Promise<StoredArtifact> {
    const response = await this.hubFetch(`/api/artifacts/${encodeURIComponent(id)}/duplicate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: newId }),
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async exportArtifact(id: string, workspaceId: string, path: string, channel = ''): Promise<FileChange> {
    const response = await this.hubFetch(`/api/artifacts/${encodeURIComponent(id)}/export`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ workspace_id: workspaceId, path, channel }),
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  /** Load a Neural Canvas artifact asset as a data URL (auth via hub session). */
  async fetchArtifactAssetDataUrl(artifactId: string, name: string): Promise<string> {
    const response = await this.hubFetch(
      `/api/artifacts/${encodeURIComponent(artifactId)}/assets/${encodeURIComponent(name)}`,
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    const buf = await response.arrayBuffer();
    const bytes = new Uint8Array(buf);
    let binary = '';
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i]!);
    }
    const b64 = btoa(binary);
    const ct = response.headers.get('Content-Type') || 'application/octet-stream';
    const mime = ct.includes('octet-stream')
      ? guessImageMimeFromName(name)
      : ct.split(';')[0]!.trim();
    return `data:${mime};base64,${b64}`;
  }
}
