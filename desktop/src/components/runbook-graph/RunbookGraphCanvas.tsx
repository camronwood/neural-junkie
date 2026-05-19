import {
  Background,
  Controls,
  ReactFlow,
  type Connection,
  type Edge,
  type Node,
  type OnEdgesChange,
  type OnNodesChange,
  applyEdgeChanges,
  applyNodeChanges,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useCallback, useEffect, useMemo, useState } from 'react';
import type { CollaborationTask } from '../../types/protocol';
import {
  applyEdgeConnect,
  applyEdgeRemove,
  autoLayoutDagre,
  edgeIsActive,
  loadLayout,
  positionsFromNodes,
  saveLayout,
  tasksToFlow,
  validateDAG,
  type RunbookTaskNodeData,
} from '../../utils/runbookDAG';
import { RunbookTaskNode } from './RunbookTaskNode';

const nodeTypes = { runbookTask: RunbookTaskNode };

interface RunbookGraphCanvasProps {
  collaborationId: string;
  tasks: CollaborationTask[];
  phase: string;
  editable: boolean;
  onTasksChange: (tasks: CollaborationTask[]) => void;
  selectedTaskId: string | null;
  onSelectTask: (id: string | null) => void;
  onValidationError: (msg: string | null) => void;
  layoutVersion: number;
}

export function RunbookGraphCanvas({
  collaborationId,
  tasks,
  phase,
  editable,
  onTasksChange,
  selectedTaskId,
  onSelectTask,
  onValidationError,
  layoutVersion,
}: RunbookGraphCanvasProps) {
  const taskById = useMemo(() => new Map(tasks.map((t) => [t.id, t])), [tasks]);

  const buildNodesEdges = useCallback(
    (layoutMap: ReturnType<typeof loadLayout>) => {
      const { nodes, edges } = tasksToFlow(tasks, layoutMap);
      const nodesWithData = nodes.map((n) => {
        const idx = tasks.findIndex((t) => t.id === n.id);
        const task = tasks[idx];
        return {
          ...n,
          data: {
            task,
            index: idx,
            phase,
            editable,
            allTasks: tasks,
          } satisfies RunbookTaskNodeData,
          selected: n.id === selectedTaskId,
        };
      });
      const edgesStyled = edges.map((e) => {
        const dep = taskById.get(e.source);
        const active = edgeIsActive(dep, phase);
        return {
          ...e,
          animated: phase === 'executing' && active,
          style: {
            stroke: active ? '#64748b' : '#475569',
            strokeWidth: 2,
            opacity: active ? 1 : 0.45,
          },
        };
      });
      return { nodes: nodesWithData, edges: edgesStyled };
    },
    [tasks, phase, editable, selectedTaskId, taskById]
  );

  const [layoutMap, setLayoutMap] = useState(() => loadLayout(collaborationId));
  const initial = buildNodesEdges(layoutMap);
  const [nodes, setNodes] = useState<Node<RunbookTaskNodeData>[]>(initial.nodes);
  const [edges, setEdges] = useState<Edge[]>(initial.edges);

  useEffect(() => {
    const map = loadLayout(collaborationId);
    setLayoutMap(map);
    const built = buildNodesEdges(map);
    setNodes(built.nodes);
    setEdges(built.edges);
  }, [tasks, collaborationId, buildNodesEdges, layoutVersion]);

  useEffect(() => {
    const v = validateDAG(tasks);
    onValidationError(v.ok ? null : v.error);
  }, [tasks, onValidationError]);

  const onNodesChange: OnNodesChange<Node<RunbookTaskNodeData>> = useCallback(
    (changes) => {
      setNodes((nds) => applyNodeChanges(changes, nds));
    },
    []
  );

  const onEdgesChange: OnEdgesChange = useCallback(
    (changes) => {
      if (!editable) return;
      let nextTasks = tasks;
      for (const ch of changes) {
        if (ch.type === 'remove' && 'id' in ch) {
          const edge = edges.find((e) => e.id === ch.id);
          if (edge) {
            nextTasks = applyEdgeRemove(nextTasks, edge.target, edge.source);
          }
        }
      }
      if (nextTasks !== tasks) {
        onTasksChange(nextTasks);
      }
      setEdges((eds) => applyEdgeChanges(changes, eds));
    },
    [editable, tasks, edges, onTasksChange]
  );

  const onConnect = useCallback(
    (conn: Connection) => {
      if (!editable || !conn.source || !conn.target) return;
      const { tasks: next, error } = applyEdgeConnect(tasks, conn.source, conn.target);
      if (error) {
        onValidationError(error);
        return;
      }
      onValidationError(null);
      onTasksChange(next);
    },
    [editable, tasks, onTasksChange, onValidationError]
  );

  const onNodeDragStop = useCallback(
    (_: React.MouseEvent, node: Node) => {
      const positions = positionsFromNodes(
        nodes.map((n) => (n.id === node.id ? { ...n, position: node.position } : n))
      );
      saveLayout(collaborationId, positions);
      setLayoutMap(positions);
    },
    [nodes, collaborationId]
  );

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      onSelectTask(node.id);
    },
    [onSelectTask]
  );

  const onPaneClick = useCallback(() => {
    onSelectTask(null);
  }, [onSelectTask]);

  return (
    <div style={{ flex: 1, minHeight: 0 }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeDragStop={onNodeDragStop}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        nodeTypes={nodeTypes}
        nodesDraggable
        nodesConnectable={editable}
        elementsSelectable
        deleteKeyCode={editable ? ['Backspace', 'Delete'] : null}
        fitView
        snapToGrid
        snapGrid={[16, 16]}
        proOptions={{ hideAttribution: true }}
      >
        <Background gap={16} color="#333" />
        <Controls />
      </ReactFlow>
    </div>
  );
}

export function useRunbookGraphLayoutActions(
  collaborationId: string,
  tasks: CollaborationTask[],
  setLayoutVersion: React.Dispatch<React.SetStateAction<number>>
) {
  const runAutoLayout = useCallback(() => {
    const map = loadLayout(collaborationId);
    const next = autoLayoutDagre(tasks, map);
    saveLayout(collaborationId, next);
    setLayoutVersion((v) => v + 1);
  }, [collaborationId, tasks, setLayoutVersion]);

  return { runAutoLayout };
}
