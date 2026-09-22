/** IdeApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class IdeApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async searchWorkspaceFiles(
    workspaceId: string,
    query: string,
    limit = 50
  ): Promise<string[]> {
    const params = new URLSearchParams({
      workspace: workspaceId,
      q: query,
      limit: String(limit),
    });
    const response = await this.hubFetch(`/api/workspaces/files/search?${params}`, {
      method: 'GET',
    });
    if (!response.ok) {
      throw new Error(`Failed to search files: ${response.statusText}`);
    }
    const data = (await response.json()) as { paths?: string[] };
    return data.paths ?? [];
  }

  async searchWorkspaceSymbols(
    workspaceId: string,
    query: string,
    limit = 50
  ): Promise<
    Array<{ name: string; path: string; line: number; kind: string; language: string }>
  > {
    const params = new URLSearchParams({
      workspace: workspaceId,
      q: query,
      limit: String(limit),
    });
    const response = await this.hubFetch(`/api/workspaces/symbols/search?${params}`, {
      method: 'GET',
    });
    if (!response.ok) {
      throw new Error(`Failed to search symbols: ${response.statusText}`);
    }
    const data = (await response.json()) as { symbols?: Array<{
      name: string;
      path: string;
      line: number;
      kind: string;
      language: string;
    }> };
    return data.symbols ?? [];
  }

  async devFastEdit(params: {
    workspaceId: string;
    path?: string;
    instruction: string;
    selection?: string;
    agentType?: string;
    metadata?: Record<string, unknown>;
  }): Promise<{
    response: string;
    proposed: boolean;
    change_id?: string;
    agent?: string;
    agent_type?: string;
  }> {
    const response = await this.hubFetch('/api/dev/fast-edit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workspace_id: params.workspaceId,
        path: params.path,
        instruction: params.instruction,
        selection: params.selection,
        agent_type: params.agentType,
        metadata: params.metadata,
      }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Fast edit failed: ${response.statusText}`);
    }
    return response.json();
  }

  async getGoLSPDiagnostics(
    workspaceId: string
  ): Promise<
    Array<{ path: string; line: number; column: number; message: string; severity: string }>
  > {
    const params = new URLSearchParams({ workspace: workspaceId });
    const response = await this.hubFetch(`/api/lsp/go/diagnostics?${params}`, {
      method: 'GET',
    });
    if (!response.ok) {
      return [];
    }
    const data = (await response.json()) as {
      diagnostics?: Array<{
        path: string;
        line: number;
        column: number;
        message: string;
        severity: string;
      }>;
    };
    return data.diagnostics ?? [];
  }

  async getLSPDiagnostics(
    lang: 'rust' | 'python',
    workspaceId: string
  ): Promise<
    Array<{ path: string; line: number; column: number; message: string; severity: string }>
  > {
    const params = new URLSearchParams({ workspace: workspaceId });
    const response = await this.hubFetch(`/api/lsp/${lang}/diagnostics?${params}`, {
      method: 'GET',
    });
    if (!response.ok) return [];
    const data = (await response.json()) as {
      diagnostics?: Array<{
        path: string;
        line: number;
        column: number;
        message: string;
        severity: string;
      }>;
    };
    return data.diagnostics ?? [];
  }

  async devComplete(params: {
    prefix: string;
    suffix?: string;
    language?: string;
    path?: string;
    context?: string;
    model?: string;
    neighbor_snippets?: Array<{ path: string; content: string; source?: string }>;
    n?: number;
    signal?: AbortSignal;
  }): Promise<{ completion: string; completions?: string[]; model?: string }> {
    const response = await this.hubFetch('/api/dev/complete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        prefix: params.prefix,
        suffix: params.suffix ?? '',
        language: params.language,
        path: params.path,
        context: params.context,
        model: params.model,
        neighbor_snippets: params.neighbor_snippets,
        n: params.n ?? 2,
      }),
      signal: params.signal,
    });
    if (!response.ok) {
      return { completion: '' };
    }
    return response.json();
  }

  /**
   * Stream ghost-text completion via NDJSON (`/api/dev/complete/stream`).
   * Falls back to non-stream `devComplete` when the stream endpoint is unavailable.
   */
  async devCompleteStream(
    params: {
      prefix: string;
      suffix?: string;
      language?: string;
      path?: string;
      context?: string;
      model?: string;
      neighbor_snippets?: Array<{ path: string; content: string; source?: string }>;
      n?: number;
      signal?: AbortSignal;
    },
    onChunk?: (text: string) => void
  ): Promise<{ completion: string; completions: string[]; model?: string; streamed: boolean }> {
    const body = {
      prefix: params.prefix,
      suffix: params.suffix ?? '',
      language: params.language,
      path: params.path,
      context: params.context,
      model: params.model,
      neighbor_snippets: params.neighbor_snippets,
      n: params.n ?? 2,
    };
    try {
      const response = await this.hubFetch('/api/dev/complete/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/x-ndjson' },
        body: JSON.stringify(body),
        signal: params.signal,
      });
      if (!response.ok || !response.body) {
        throw new Error(`stream unavailable: ${response.status}`);
      }
      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      let completion = '';
      let completions: string[] = [];
      let model: string | undefined;
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';
        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed) continue;
          let evt: {
            type?: string;
            text?: string;
            completion?: string;
            completions?: string[];
            model?: string;
            error?: string;
          };
          try {
            evt = JSON.parse(trimmed);
          } catch {
            continue;
          }
          if (evt.type === 'chunk' && evt.text) {
            completion += evt.text;
            onChunk?.(evt.text);
          } else if (evt.type === 'done') {
            if (evt.completion) completion = evt.completion;
            if (Array.isArray(evt.completions) && evt.completions.length > 0) {
              completions = evt.completions;
            }
            model = evt.model;
          } else if (evt.type === 'error') {
            throw new Error(evt.error || 'stream error');
          }
        }
      }
      if (completions.length === 0 && completion) {
        completions = [completion];
      }
      return { completion: completion.trim(), completions, model, streamed: true };
    } catch {
      if (params.signal?.aborted) {
        return { completion: '', completions: [], streamed: false };
      }
      const fallback = await this.devComplete({ ...params, signal: params.signal });
      const completions =
        Array.isArray(fallback.completions) && fallback.completions.length > 0
          ? fallback.completions
          : fallback.completion
            ? [fallback.completion]
            : [];
      return {
        completion: fallback.completion ?? '',
        completions,
        model: fallback.model,
        streamed: false,
      };
    }
  }

  async devAgentTurn(params: {
    workspaceId: string;
    instruction: string;
    sessionId?: string;
    mode?: 'ask' | 'agent';
    path?: string;
    selection?: string;
    agentType?: string;
    metadata?: Record<string, unknown>;
    attachments?: Array<Record<string, unknown>>;
  }): Promise<{
    response: string;
    proposed: boolean;
    session_id: string;
    channel: string;
    change_ids?: string[];
    agent?: string;
    agent_type?: string;
  }> {
    const response = await this.hubFetch('/api/dev/agent-turn', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workspace_id: params.workspaceId,
        instruction: params.instruction,
        session_id: params.sessionId,
        mode: params.mode ?? 'agent',
        path: params.path,
        selection: params.selection,
        agent_type: params.agentType,
        metadata: params.metadata,
        attachments: params.attachments,
      }),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Agent turn failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoSemanticSearch(params: {
    repoPath?: string;
    repoPaths?: string[];
    query: string;
    limit?: number;
  }): Promise<{
    chunks: Array<{ path: string; content: string; repo_path?: string; repo_name?: string }>;
  }> {
    const paths =
      params.repoPaths?.filter((p) => p?.trim()).map((p) => p.trim()) ??
      (params.repoPath?.trim() ? [params.repoPath.trim()] : []);
    const response = await this.hubFetch('/api/repo/search/semantic', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        repo_paths: paths.length > 1 ? paths : undefined,
        repo_path: paths.length === 1 ? paths[0] : params.repoPath,
        query: params.query,
        limit: params.limit ?? 8,
      }),
    });
    if (!response.ok) {
      throw new Error(`Semantic search failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoIndexStatus(repoPath: string): Promise<{
    ready: boolean;
    building: boolean;
    chunk_count: number;
    embedding_model?: string;
  }> {
    const params = new URLSearchParams({ repo_path: repoPath });
    const response = await this.hubFetch(`/api/repo/index/status?${params}`);
    if (!response.ok) {
      throw new Error(`Index status failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraph(repoPath: string): Promise<import('../../components/knowledge-graph/types').KnowledgeGraphSummary> {
    const params = new URLSearchParams({ repo_path: repoPath });
    const response = await this.hubFetch(`/api/repo/graph?${params}`);
    if (!response.ok) {
      throw new Error(`Knowledge graph failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraphSubgraph(
    repoPath: string,
    q: string,
    hops = 1,
    limit = 120,
  ): Promise<import('../../components/knowledge-graph/types').KnowledgeGraphSummary & { query?: string; nodes: import('../../components/knowledge-graph/types').KnowledgeGraphNode[]; edges: import('../../components/knowledge-graph/types').KnowledgeGraphEdge[] }> {
    const params = new URLSearchParams({
      repo_path: repoPath,
      q,
      hops: String(hops),
      limit: String(limit),
    });
    const response = await this.hubFetch(`/api/repo/graph/subgraph?${params}`);
    if (!response.ok) {
      throw new Error(`Graph subgraph failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraphPath(
    repoPath: string,
    from: string,
    to: string,
  ): Promise<{
    from: string;
    to: string;
    found: boolean;
    nodes: import('../../components/knowledge-graph/types').KnowledgeGraphNode[];
    edges: import('../../components/knowledge-graph/types').KnowledgeGraphEdge[];
  }> {
    const params = new URLSearchParams({ repo_path: repoPath, from, to });
    const response = await this.hubFetch(`/api/repo/graph/path?${params}`);
    if (!response.ok) {
      throw new Error(`Graph path failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraphExplain(
    repoPath: string,
    node: string,
  ): Promise<import('../../components/knowledge-graph/types').KnowledgeGraphExplain> {
    const params = new URLSearchParams({ repo_path: repoPath, node });
    const response = await this.hubFetch(`/api/repo/graph/explain?${params}`);
    if (!response.ok) {
      throw new Error(`Graph explain failed: ${response.statusText}`);
    }
    return response.json();
  }

  async repoGraphStatus(
    repoPath: string,
    rebuild = false,
  ): Promise<import('../../components/knowledge-graph/types').KnowledgeGraphMeta> {
    const params = new URLSearchParams({ repo_path: repoPath });
    if (rebuild) params.set('rebuild', '1');
    const response = await this.hubFetch(`/api/repo/graph/status?${params}`);
    if (!response.ok) {
      throw new Error(`Graph status failed: ${response.statusText}`);
    }
    return response.json();
  }

}
