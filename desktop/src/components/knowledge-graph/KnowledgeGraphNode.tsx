import { Handle, Position, type Node, type NodeProps } from '@xyflow/react';
import type { KnowledgeNodeData } from './types';

type KGNode = Node<KnowledgeNodeData, 'knowledgeNode'>;

export function KnowledgeGraphNodeView({ data, selected }: NodeProps<KGNode>) {
  const { node, color } = data;
  const size =
    node.kind === 'package' ? 18 : node.kind === 'symbol' ? 10 : node.kind === 'repo' ? 22 : 12;
  const label =
    node.label.length > 28 ? `${node.label.slice(0, 26)}…` : node.label;

  return (
    <div
      title={`${node.label} (${node.kind})\n${node.path ?? ''}${node.line ? `:${node.line}` : ''}`}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 2,
        minWidth: 72,
      }}
    >
      <Handle type="target" position={Position.Top} className="!w-1 !h-1 !opacity-0" />
      <div
        style={{
          width: size,
          height: size,
          borderRadius: '50%',
          backgroundColor: color,
          border: selected ? '2px solid #fff' : '1px solid rgba(255,255,255,0.25)',
          boxShadow: selected ? `0 0 0 3px ${color}88` : `0 0 8px ${color}55`,
        }}
      />
      <div
        style={{
          fontSize: 9,
          color: '#e2e8f0',
          maxWidth: 90,
          textAlign: 'center',
          lineHeight: 1.15,
          textShadow: '0 1px 2px rgba(0,0,0,0.8)',
        }}
      >
        {label}
      </div>
      <Handle type="source" position={Position.Bottom} className="!w-1 !h-1 !opacity-0" />
    </div>
  );
}
