import { describe, expect, it } from 'vitest';
import { communityColor, layoutKnowledgeGraph } from './layout';
import type { KnowledgeGraphEdge, KnowledgeGraphNode } from './types';

describe('communityColor', () => {
  it('returns explicit color when provided', () => {
    expect(communityColor('hub', '#ff0000')).toBe('#ff0000');
  });

  it('is stable for the same community id', () => {
    expect(communityColor('internal/hub')).toBe(communityColor('internal/hub'));
  });
});

describe('layoutKnowledgeGraph', () => {
  it('places nodes and keeps edges whose endpoints exist', () => {
    const nodes: KnowledgeGraphNode[] = [
      { id: 'a', label: 'A', kind: 'package', community: 'pkg', degree: 3 },
      { id: 'b', label: 'B', kind: 'symbol', community: 'pkg', degree: 1 },
      { id: 'c', label: 'C', kind: 'file', community: 'other', degree: 2 },
    ];
    const edges: KnowledgeGraphEdge[] = [
      { id: 'e1', from: 'a', to: 'b', kind: 'contains', provenance: 'EXTRACTED' },
      { id: 'e2', from: 'a', to: 'missing', kind: 'imports', provenance: 'INFERRED' },
      { id: 'e3', from: 'b', to: 'c', kind: 'imports', provenance: 'INFERRED' },
    ];
    const { nodes: flowNodes, edges: flowEdges } = layoutKnowledgeGraph(nodes, edges, {
      pkg: '#16a34a',
    });
    expect(flowNodes).toHaveLength(3);
    expect(flowNodes.find((n) => n.id === 'a')?.data.color).toBe('#16a34a');
    expect(flowEdges.map((e) => e.id).sort()).toEqual(['e1', 'e3']);
    expect(flowEdges.find((e) => e.id === 'e3')?.animated).toBe(true);
  });
});
