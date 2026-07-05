import dagre from '@dagrejs/dagre';
import type { Edge, Node } from '@xyflow/react';
import type { CollaborationTask } from '../types/protocol';

export const LAYOUT_STORAGE_PREFIX = 'nj-runbook-graph-layout:';

export type LayoutMap = Record<string, { x: number; y: number }>;

export type RunbookTaskNodeData = {
  task: CollaborationTask;
  index: number;
  phase: string;
  editable?: boolean;
  allTasks?: CollaborationTask[];
};

export type DAGValidation = { ok: true } | { ok: false; error: string };

/** Merge dependencies from task.dependencies and dependency_edges (deduped). */
export function effectiveDependencies(task: CollaborationTask): string[] {
  const deps = [...(task.dependencies ?? [])];
  const seen = new Set(deps);
  for (const edge of task.dependency_edges ?? []) {
    const from = edge.from_task_id?.trim();
    if (from && !seen.has(from)) {
      seen.add(from);
      deps.push(from);
    }
  }
  return deps;
}

export function layoutStorageKey(collaborationId: string): string {
  return `${LAYOUT_STORAGE_PREFIX}${collaborationId}`;
}

export function loadLayout(collaborationId: string, serverLayout?: GraphLayoutFromCollab): LayoutMap {
  if (serverLayout && Object.keys(serverLayout).length > 0) {
    const map: LayoutMap = {};
    for (const [id, pos] of Object.entries(serverLayout)) {
      map[id] = { x: pos.x, y: pos.y };
    }
    return map;
  }
  try {
    const raw = localStorage.getItem(layoutStorageKey(collaborationId));
    if (!raw) return {};
    const parsed = JSON.parse(raw) as LayoutMap;
    return parsed && typeof parsed === 'object' ? parsed : {};
  } catch {
    return {};
  }
}

export type GraphLayoutFromCollab = Record<string, { x: number; y: number }>;

export function saveLayout(collaborationId: string, positions: LayoutMap): void {
  try {
    localStorage.setItem(layoutStorageKey(collaborationId), JSON.stringify(positions));
  } catch {
    // ignore quota errors
  }
}

export function validateDAG(tasks: CollaborationTask[]): DAGValidation {
  if (tasks.length === 0) {
    return { ok: true };
  }
  const ids = new Map<string, number>();
  for (let i = 0; i < tasks.length; i++) {
    const t = tasks[i];
    if (!t.id) {
      return { ok: false, error: `Task ${i + 1} has no id` };
    }
    ids.set(t.id, i);
  }
  for (let i = 0; i < tasks.length; i++) {
    const t = tasks[i];
    for (const dep of effectiveDependencies(t)) {
      if (dep === t.id) {
        return { ok: false, error: `Task ${i + 1} depends on itself` };
      }
      if (!ids.has(dep)) {
        return { ok: false, error: `Task ${i + 1} references unknown dependency` };
      }
    }
  }
  const state = new Map<string, 0 | 1 | 2>();
  const visit = (id: string): DAGValidation => {
    const s = state.get(id) ?? 0;
    if (s === 1) {
      return { ok: false, error: 'Dependency cycle detected' };
    }
    if (s === 2) {
      return { ok: true };
    }
    state.set(id, 1);
    const idx = ids.get(id)!;
    const task = tasks[idx];
    for (const dep of effectiveDependencies(task)) {
      const err = visit(dep);
      if (!err.ok) return err;
    }
    state.set(id, 2);
    return { ok: true };
  };
  for (const id of ids.keys()) {
    if ((state.get(id) ?? 0) === 0) {
      const err = visit(id);
      if (!err.ok) return err;
    }
  }
  return { ok: true };
}

export function tasksToFlow(
  tasks: CollaborationTask[],
  layoutMap: LayoutMap,
  options?: { defaultX?: number; defaultY?: number }
): { nodes: Node<RunbookTaskNodeData>[]; edges: Edge[] } {
  const defaultX = options?.defaultX ?? 0;
  const defaultY = options?.defaultY ?? 0;
  const nodes: Node<RunbookTaskNodeData>[] = tasks.map((task, index) => {
    const pos = layoutMap[task.id];
    const col = index % 3;
    const row = Math.floor(index / 3);
    return {
      id: task.id,
      type: 'runbookTask',
      position: pos ?? { x: defaultX + col * 280, y: defaultY + row * 160 },
      data: { task, index, phase: '' },
    };
  });
  const edges: Edge[] = [];
  for (const task of tasks) {
    for (const depId of effectiveDependencies(task)) {
      edges.push({
        id: `${depId}->${task.id}`,
        source: depId,
        target: task.id,
        type: 'smoothstep',
        label: edgeLabel(task, depId),
        style: edgeStyle(task, depId),
      });
    }
  }
  return { nodes, edges };
}

export function applyEdgeConnect(
  tasks: CollaborationTask[],
  sourceId: string,
  targetId: string,
  condition?: import('../types/protocol').EdgeCondition
): { tasks: CollaborationTask[]; error?: string } {
  if (sourceId === targetId) {
    return { tasks, error: 'A task cannot depend on itself' };
  }
  const next = tasks.map((t) => {
    if (t.id !== targetId) return t;
    const deps = [...(t.dependencies ?? [])];
    const edges = [...(t.dependency_edges ?? [])];
    if (!deps.includes(sourceId)) {
      deps.push(sourceId);
    }
    if (condition && condition.mode !== 'always') {
      edges.push({ from_task_id: sourceId, condition });
    } else if (!edges.some((e) => e.from_task_id === sourceId)) {
      edges.push({ from_task_id: sourceId, condition: { mode: 'always' } });
    }
    return { ...t, dependencies: deps, dependency_edges: edges };
  });
  const validation = validateDAG(next);
  if (!validation.ok) {
    return { tasks, error: validation.error };
  }
  return { tasks: next };
}

export function applyEdgeRemove(
  tasks: CollaborationTask[],
  targetId: string,
  sourceId: string
): CollaborationTask[] {
  return tasks.map((t) => {
    if (t.id !== targetId) return t;
    return {
      ...t,
      dependencies: (t.dependencies ?? []).filter((d) => d !== sourceId),
      dependency_edges: (t.dependency_edges ?? []).filter((e) => e.from_task_id !== sourceId),
    };
  });
}

export function removeTask(tasks: CollaborationTask[], taskId: string): CollaborationTask[] {
  return tasks
    .filter((t) => t.id !== taskId)
    .map((t) => ({
      ...t,
      dependencies: (t.dependencies ?? []).filter((d) => d !== taskId),
    }));
}

export function positionsFromNodes(nodes: Node[]): LayoutMap {
  const map: LayoutMap = {};
  for (const n of nodes) {
    map[n.id] = { x: n.position.x, y: n.position.y };
  }
  return map;
}

const NODE_WIDTH = 220;
const NODE_HEIGHT = 100;

export function autoLayoutDagre(
  tasks: CollaborationTask[],
  layoutMap: LayoutMap
): LayoutMap {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: 'LR', nodesep: 60, ranksep: 80 });

  for (const task of tasks) {
    g.setNode(task.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  }
  for (const task of tasks) {
    for (const depId of effectiveDependencies(task)) {
      g.setEdge(depId, task.id);
    }
  }

  dagre.layout(g);

  const next: LayoutMap = { ...layoutMap };
  for (const task of tasks) {
    const n = g.node(task.id);
    if (n) {
      next[task.id] = {
        x: n.x - NODE_WIDTH / 2,
        y: n.y - NODE_HEIGHT / 2,
      };
    }
  }
  return next;
}

export function edgeIsActive(depTask: CollaborationTask | undefined, phase: string): boolean {
  if (phase !== 'executing') return true;
  return depTask?.status === 'completed';
}

function edgeLabel(task: CollaborationTask, depId: string): string | undefined {
  const edge = (task.dependency_edges ?? []).find((e) => e.from_task_id === depId);
  if (!edge?.condition || edge.condition.mode === 'always') return undefined;
  if (edge.condition.mode === 'on_output' && edge.condition.contains) {
    return `if contains "${edge.condition.contains}"`;
  }
  if (edge.condition.mode === 'on_status' && edge.condition.status) {
    return `if ${edge.condition.status}`;
  }
  return edge.condition.mode;
}

function edgeStyle(task: CollaborationTask, depId: string): { strokeDasharray?: string } | undefined {
  const edge = (task.dependency_edges ?? []).find((e) => e.from_task_id === depId);
  if (edge?.condition && edge.condition.mode !== 'always') {
    return { strokeDasharray: '5 5' };
  }
  return undefined;
}
