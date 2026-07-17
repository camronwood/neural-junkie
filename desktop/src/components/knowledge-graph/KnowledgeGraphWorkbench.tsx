import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Edge,
  type Node,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useCallback, useEffect, useMemo, useState, type MouseEvent } from 'react';
import { ChatAPI } from '../../api/chatAPI';
import { getHubBaseURL } from '../../config/hubUrl';
import { useComposerPrefillStore } from '../../stores/composerPrefillStore';
import { useEditorStore } from '../../stores/editorStore';
import { useFileExplorerStore } from '../../stores/fileExplorerStore';
import { KnowledgeGraphNodeView } from './KnowledgeGraphNode';
import { communityColor, layoutKnowledgeGraph } from './layout';
import type {
  KnowledgeGraphCommunity,
  KnowledgeGraphEdge,
  KnowledgeGraphExplain,
  KnowledgeGraphMeta,
  KnowledgeGraphNode,
  KnowledgeNodeData,
} from './types';

const api = new ChatAPI(getHubBaseURL());
const nodeTypes = { knowledgeNode: KnowledgeGraphNodeView };

interface KnowledgeGraphWorkbenchProps {
  workspaceId: string;
  repoPath: string;
}

export function KnowledgeGraphWorkbench({ workspaceId, repoPath }: KnowledgeGraphWorkbenchProps) {
  const openFile = useEditorStore((s) => s.openFile);
  const revealLine = useEditorStore((s) => s.revealLine);
  const requestComposerPrefill = useComposerPrefillStore((s) => s.requestPrefill);
  const workspaces = useFileExplorerStore((s) => s.workspaces);
  const activeWorkspaceId = useFileExplorerStore((s) => s.activeWorkspaceId);
  const setActiveWorkspace = useFileExplorerStore((s) => s.setActiveWorkspace);
  const activeWorkspace = useMemo(
    () => workspaces.find((workspace) => workspace.id === activeWorkspaceId),
    [activeWorkspaceId, workspaces],
  );
  const graphWorkspaceId = activeWorkspace?.id ?? workspaceId;
  const graphRepoPath = activeWorkspace?.path ?? repoPath;
  const graphPathParts = graphRepoPath.split(/[\\/]/).filter(Boolean);
  const graphProjectName =
    activeWorkspace?.name || graphPathParts[graphPathParts.length - 1] || 'Workspace';

  const [meta, setMeta] = useState<KnowledgeGraphMeta | null>(null);
  const [communities, setCommunities] = useState<KnowledgeGraphCommunity[]>([]);
  const [godNodes, setGodNodes] = useState<KnowledgeGraphNode[]>([]);
  const [graphNodes, setGraphNodes] = useState<KnowledgeGraphNode[]>([]);
  const [graphEdges, setGraphEdges] = useState<KnowledgeGraphEdge[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [activeCommunity, setActiveCommunity] = useState<string | null>(null);
  const [explain, setExplain] = useState<KnowledgeGraphExplain | null>(null);
  const [pathFrom, setPathFrom] = useState<string | null>(null);
  const [pathMode, setPathMode] = useState(false);
  const [statusMsg, setStatusMsg] = useState('');

  const [nodes, setNodes, onNodesChange] = useNodesState<Node<KnowledgeNodeData>>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const colorMap = useMemo(() => {
    const m: Record<string, string> = {};
    for (const c of communities) m[c.id] = c.color || communityColor(c.id);
    return m;
  }, [communities]);

  const applyGraph = useCallback(
    (sourceNodes: KnowledgeGraphNode[], sourceEdges: Parameters<typeof layoutKnowledgeGraph>[1]) => {
      let filtered = sourceNodes;
      let filteredEdges = sourceEdges;
      if (activeCommunity) {
        filtered = sourceNodes.filter((n) => (n.community || 'root') === activeCommunity);
        const ids = new Set(filtered.map((n) => n.id));
        filteredEdges = sourceEdges.filter((e) => ids.has(e.from) && ids.has(e.to));
      }
      const laid = layoutKnowledgeGraph(filtered, filteredEdges, colorMap);
      setNodes(laid.nodes);
      setEdges(laid.edges);
    },
    [activeCommunity, colorMap, setEdges, setNodes],
  );

  const loadSummary = useCallback(async () => {
    setError(null);
    try {
      const summary = await api.repoGraph(graphRepoPath);
      setMeta(summary.meta);
      setCommunities(summary.communities ?? []);
      setGodNodes(summary.god_nodes ?? []);
      setGraphNodes(summary.nodes ?? []);
      setGraphEdges(summary.edges ?? []);
      if (!summary.meta.ready) {
        setStatusMsg(summary.meta.building ? 'Building knowledge graph…' : 'Graph pending…');
        return;
      }
      setStatusMsg(
        `${summary.meta.node_count} nodes · ${summary.meta.edge_count} edges`,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [graphRepoPath]);

  useEffect(() => {
    setMeta(null);
    setCommunities([]);
    setGodNodes([]);
    setGraphNodes([]);
    setGraphEdges([]);
    setActiveCommunity(null);
    setExplain(null);
    setQuery('');
    setStatusMsg(`Loading ${graphProjectName}…`);
    void loadSummary();
  }, [graphProjectName, graphRepoPath, loadSummary]);

  useEffect(() => {
    applyGraph(graphNodes, graphEdges);
  }, [activeCommunity, applyGraph, colorMap, graphEdges, graphNodes]);

  useEffect(() => {
    if (!meta || meta.ready || !meta.building) return;
    const id = window.setInterval(() => void loadSummary(), 5000);
    return () => window.clearInterval(id);
  }, [meta, loadSummary]);

  const runSearch = async () => {
    const q = query.trim();
    if (!q) {
      void loadSummary();
      return;
    }
    try {
      const sg = await api.repoGraphSubgraph(graphRepoPath, q, 2, 160);
      applyGraph(sg.nodes ?? [], sg.edges ?? []);
      setStatusMsg(`Subgraph for “${q}”: ${sg.nodes?.length ?? 0} nodes`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const onNodeClick = useCallback(
    async (_: MouseEvent, node: Node<KnowledgeNodeData>) => {
      if (pathMode) {
        if (!pathFrom) {
          setPathFrom(node.id);
          setStatusMsg(`Path from ${node.data.node.label} — pick a target`);
          return;
        }
        try {
          const path = await api.repoGraphPath(graphRepoPath, pathFrom, node.id);
          setPathFrom(null);
          if (!path.found) {
            setStatusMsg('No path found');
            return;
          }
          applyGraph(path.nodes ?? [], path.edges ?? []);
          setStatusMsg(`Path: ${path.nodes?.map((n) => n.label).join(' → ')}`);
        } catch (e) {
          setError(e instanceof Error ? e.message : String(e));
        }
        return;
      }
      try {
        const ex = await api.repoGraphExplain(graphRepoPath, node.id);
        setExplain(ex);
        const sg = await api.repoGraphSubgraph(graphRepoPath, node.id, 1, 80);
        applyGraph(sg.nodes ?? [], sg.edges ?? []);
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      }
    },
    [applyGraph, graphRepoPath, pathFrom, pathMode],
  );

  const onNodeDoubleClick = useCallback(
    async (_: MouseEvent, node: Node<KnowledgeNodeData>) => {
      const n = node.data.node;
      if (!n.path) return;
      try {
        const content = await api.fetchFileContent(graphWorkspaceId, n.path);
        openFile(graphWorkspaceId, n.path, content ?? '', undefined);
        if (n.line && n.line > 0) {
          revealLine(graphWorkspaceId, n.path, n.line);
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      }
    },
    [graphWorkspaceId, openFile, revealLine],
  );

  const rebuild = async () => {
    setStatusMsg('Rebuilding…');
    await api.repoGraphStatus(graphRepoPath, true);
    void loadSummary();
  };

  const askAgents = () => {
    if (!explain?.node) return;
    const label = explain.node.label;
    const detail = explain.description;
    requestComposerPrefill(`How does ${label} relate to the rest of the codebase? ${detail}`);
  };

  return (
    <div className="flex h-full w-full min-h-0 bg-[#0b1220] text-slate-100">
      <aside className="w-56 flex-shrink-0 border-r border-slate-800 flex flex-col overflow-hidden">
        <div className="px-3 py-2 border-b border-slate-800">
          <div className="text-sm font-semibold text-slate-100 truncate" title={graphRepoPath}>
            {graphProjectName}
          </div>
          <div className="text-[10px] text-slate-500 truncate mb-2" title={graphRepoPath}>
            {graphRepoPath}
          </div>
          {workspaces.length > 1 && (
            <select
              value={graphWorkspaceId}
              onChange={(event) => setActiveWorkspace(event.target.value)}
              className="w-full mb-2 rounded border border-slate-700 bg-slate-900 px-2 py-1 text-[11px] text-slate-200"
              aria-label="Knowledge graph project"
            >
              {workspaces.map((workspace) => (
                <option key={workspace.id} value={workspace.id}>
                  {workspace.name}
                </option>
              ))}
            </select>
          )}
          <div className="text-[10px] uppercase tracking-wider text-slate-400">Communities</div>
          <div className="text-[11px] text-slate-500 mt-0.5">{statusMsg}</div>
        </div>
        <div className="flex-1 overflow-y-auto p-2 space-y-1">
          <button
            type="button"
            className={`w-full text-left px-2 py-1 rounded text-xs ${
              !activeCommunity ? 'bg-slate-700' : 'hover:bg-slate-800'
            }`}
            onClick={() => {
              setActiveCommunity(null);
            }}
          >
            All communities
          </button>
          {communities.slice(0, 40).map((c) => (
            <button
              key={c.id}
              type="button"
              className={`w-full text-left px-2 py-1 rounded text-xs flex items-center gap-2 ${
                activeCommunity === c.id ? 'bg-slate-700' : 'hover:bg-slate-800'
              }`}
              onClick={() => {
                setActiveCommunity(c.id);
              }}
            >
              <span
                className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                style={{ backgroundColor: c.color || communityColor(c.id) }}
              />
              <span className="truncate flex-1">{c.label}</span>
              <span className="text-slate-500">{c.count}</span>
            </button>
          ))}
        </div>
        {godNodes.length > 0 && (
          <div className="border-t border-slate-800 p-2 max-h-40 overflow-y-auto">
            <div className="text-[10px] uppercase tracking-wider text-slate-400 mb-1">God nodes</div>
            {godNodes.slice(0, 12).map((n) => (
              <button
                key={n.id}
                type="button"
                className="block w-full text-left text-[11px] text-slate-300 hover:text-white truncate py-0.5"
                onClick={() => {
                  setQuery(n.label);
                  void api.repoGraphSubgraph(graphRepoPath, n.id, 1, 80).then((sg) => {
                    applyGraph(sg.nodes ?? [], sg.edges ?? []);
                  });
                }}
              >
                {n.label} <span className="text-slate-500">({n.degree})</span>
              </button>
            ))}
          </div>
        )}
      </aside>

      <div className="flex-1 flex flex-col min-w-0 min-h-0">
        <div className="flex items-center gap-2 px-3 py-2 border-b border-slate-800 bg-[#111827]">
          <input
            className="flex-1 bg-slate-900 border border-slate-700 rounded px-2 py-1 text-sm"
            placeholder="Search symbols, files, packages…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void runSearch();
            }}
          />
          <button
            type="button"
            className="px-2 py-1 text-xs rounded bg-slate-700 hover:bg-slate-600"
            onClick={() => void runSearch()}
          >
            Search
          </button>
          <button
            type="button"
            className={`px-2 py-1 text-xs rounded ${
              pathMode ? 'bg-violet-600' : 'bg-slate-700 hover:bg-slate-600'
            }`}
            onClick={() => {
              setPathMode((v) => !v);
              setPathFrom(null);
            }}
          >
            Path
          </button>
          <button
            type="button"
            className="px-2 py-1 text-xs rounded bg-slate-700 hover:bg-slate-600"
            onClick={() => void loadSummary()}
          >
            Reset
          </button>
          <button
            type="button"
            className="px-2 py-1 text-xs rounded bg-slate-700 hover:bg-slate-600"
            onClick={() => void rebuild()}
          >
            Rebuild
          </button>
        </div>

        {error && (
          <div className="px-3 py-1 text-xs text-red-300 bg-red-950/40 border-b border-red-900">{error}</div>
        )}

        <div className="flex-1 min-h-0 relative">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            nodeTypes={nodeTypes}
            onNodeClick={onNodeClick}
            onNodeDoubleClick={onNodeDoubleClick}
            fitView
            minZoom={0.15}
            maxZoom={2}
            proOptions={{ hideAttribution: true }}
            colorMode="dark"
          >
            <Background color="#1e293b" gap={24} />
            <Controls />
            <MiniMap
              nodeColor={(n) => (n.data as KnowledgeNodeData)?.color ?? '#64748b'}
              maskColor="rgba(15,23,42,0.75)"
            />
          </ReactFlow>
          {!meta?.ready && (
            <div className="absolute inset-0 flex items-center justify-center bg-[#0b1220]/80 text-sm text-slate-300">
              {statusMsg || 'Building knowledge graph…'}
            </div>
          )}
        </div>
      </div>

      <aside className="w-64 flex-shrink-0 border-l border-slate-800 p-3 overflow-y-auto">
        <div className="text-[10px] uppercase tracking-wider text-slate-400 mb-2">Inspector</div>
        {!explain ? (
          <p className="text-xs text-slate-500">
            Click a node to explain. Double-click to open in the editor. Use Path to connect two nodes.
          </p>
        ) : (
          <div className="space-y-2 text-xs">
            <div className="text-sm font-semibold text-white">{explain.node.label}</div>
            <div className="text-slate-400">
              {explain.node.kind}
              {explain.node.symbol_kind ? ` · ${explain.node.symbol_kind}` : ''}
            </div>
            <div className="text-slate-300">{explain.description}</div>
            <div>
              <span className="text-slate-500">Community:</span> {explain.community}
            </div>
            <div>
              <span className="text-slate-500">Degree:</span> {explain.degree}
            </div>
            <div>
              <span className="text-slate-500">Provenance:</span>{' '}
              {(explain.provenance_summary ?? []).join(', ') || '—'}
            </div>
            {explain.node.path && (
              <div className="break-all text-slate-400">
                {explain.node.path}
                {explain.node.line ? `:${explain.node.line}` : ''}
              </div>
            )}
            <div className="flex flex-col gap-1 pt-2">
              <button
                type="button"
                className="px-2 py-1 rounded bg-slate-700 hover:bg-slate-600"
                onClick={() => {
                  if (!explain.node.path) return;
                  void api.fetchFileContent(graphWorkspaceId, explain.node.path).then((content) => {
                    openFile(graphWorkspaceId, explain.node.path!, content ?? '');
                    if (explain.node.line) {
                      revealLine(graphWorkspaceId, explain.node.path!, explain.node.line);
                    }
                  });
                }}
                disabled={!explain.node.path}
              >
                Open in editor
              </button>
              <button
                type="button"
                className="px-2 py-1 rounded bg-violet-700 hover:bg-violet-600"
                onClick={askAgents}
              >
                Ask agents about this
              </button>
            </div>
            {explain.neighbors?.length > 0 && (
              <div className="pt-2">
                <div className="text-[10px] uppercase text-slate-500 mb-1">Neighbors</div>
                {explain.neighbors.slice(0, 20).map((n) => (
                  <div key={n.id} className="truncate text-slate-300">
                    {n.label} <span className="text-slate-500">({n.kind})</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </aside>
    </div>
  );
}
