import type { Edge, Node } from '@xyflow/react';
import type { KnowledgeGraphEdge, KnowledgeGraphNode, KnowledgeNodeData } from './types';

const DEFAULT_COLORS = [
  '#e94560',
  '#0f3460',
  '#533483',
  '#16a34a',
  '#ca8a04',
  '#0891b2',
  '#db2777',
  '#ea580c',
  '#4f46e5',
  '#059669',
];

export function communityColor(id: string, explicit?: string): string {
  if (explicit) return explicit;
  let h = 0;
  for (let i = 0; i < id.length; i++) h = h * 31 + id.charCodeAt(i);
  if (h < 0) h = -h;
  return DEFAULT_COLORS[h % DEFAULT_COLORS.length];
}

/** Circular community layout for a dense Graphify-like look. */
export function layoutKnowledgeGraph(
  nodes: KnowledgeGraphNode[],
  edges: KnowledgeGraphEdge[],
  colorByCommunity: Record<string, string>,
): { nodes: Node<KnowledgeNodeData>[]; edges: Edge[] } {
  const byComm = new Map<string, KnowledgeGraphNode[]>();
  for (const n of nodes) {
    const c = n.community || 'root';
    if (!byComm.has(c)) byComm.set(c, []);
    byComm.get(c)!.push(n);
  }
  const communities = [...byComm.keys()].sort();
  const flowNodes: Node<KnowledgeNodeData>[] = [];
  const R = 320 + Math.min(communities.length, 20) * 18;

  communities.forEach((comm, ci) => {
    const members = byComm.get(comm)!;
    members.sort((a, b) => (b.degree ?? 0) - (a.degree ?? 0));
    const angle = (ci / Math.max(communities.length, 1)) * Math.PI * 2;
    const cx = Math.cos(angle) * R;
    const cy = Math.sin(angle) * R;
    const r = 40 + Math.min(members.length, 40) * 4;
    members.forEach((n, i) => {
      const a = (i / Math.max(members.length, 1)) * Math.PI * 2;
      const size = n.kind === 'package' ? 1.2 : n.kind === 'symbol' ? 0.85 : 1;
      flowNodes.push({
        id: n.id,
        type: 'knowledgeNode',
        position: {
          x: cx + Math.cos(a) * r * size - 40,
          y: cy + Math.sin(a) * r * size - 16,
        },
        data: {
          node: n,
          color: communityColor(comm, colorByCommunity[comm]),
          selected: false,
        },
      });
    });
  });

  const nodeIds = new Set(flowNodes.map((n) => n.id));
  const flowEdges: Edge[] = [];
  for (const e of edges) {
    if (!nodeIds.has(e.from) || !nodeIds.has(e.to)) continue;
    const isImport = e.kind === 'imports' || e.kind === 'resolves_to';
    flowEdges.push({
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'default',
      animated: isImport && e.provenance === 'INFERRED',
      style: {
        stroke: isImport ? '#94a3b8' : '#475569',
        strokeWidth: isImport ? 1.2 : 0.7,
        opacity: e.kind === 'contains' ? 0.35 : 0.75,
      },
      label: isImport ? e.kind : undefined,
      labelStyle: { fontSize: 9, fill: '#94a3b8' },
    });
  }
  return { nodes: flowNodes, edges: flowEdges };
}
