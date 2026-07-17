export type GraphNodeKind = 'repo' | 'package' | 'file' | 'symbol';
export type GraphEdgeKind = 'contains' | 'imports' | 'defines' | 'resolves_to';
export type GraphProvenance = 'EXTRACTED' | 'INFERRED';

export interface KnowledgeGraphNode {
  id: string;
  kind: GraphNodeKind;
  label: string;
  path?: string;
  line?: number;
  language?: string;
  community?: string;
  degree?: number;
  symbol_kind?: string;
}

export interface KnowledgeGraphEdge {
  id: string;
  from: string;
  to: string;
  kind: GraphEdgeKind;
  provenance: GraphProvenance;
  path?: string;
  line?: number;
}

export interface KnowledgeGraphCommunity {
  id: string;
  label: string;
  count: number;
  color?: string;
}

export interface KnowledgeGraphMeta {
  repo_path: string;
  repo_hash: string;
  node_count: number;
  edge_count: number;
  git_head?: string;
  last_built_at?: string;
  ready: boolean;
  building?: boolean;
  error?: string;
}

export interface KnowledgeGraphSummary {
  meta: KnowledgeGraphMeta;
  communities: KnowledgeGraphCommunity[];
  god_nodes: KnowledgeGraphNode[];
  nodes: KnowledgeGraphNode[];
  edges: KnowledgeGraphEdge[];
}

export interface KnowledgeGraphExplain {
  node: KnowledgeGraphNode;
  neighbors: KnowledgeGraphNode[];
  edges: KnowledgeGraphEdge[];
  community: string;
  degree: number;
  provenance_summary: string[];
  description: string;
}

export type KnowledgeNodeData = {
  node: KnowledgeGraphNode;
  color: string;
  selected: boolean;
};
