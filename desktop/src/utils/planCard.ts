export const NJ_BUILD_PLAN_EVENT = 'nj-build-plan';

export type PlanTodo = {
  id: string;
  content: string;
  status: string;
};

export type ParsedPlan = {
  name: string;
  overview: string;
  todos: PlanTodo[];
  body: string;
  raw: string;
};

export type PlanBuildRequest = {
  markdown: string;
  planId: string;
};

export function parsePlanMarkdown(content: string): ParsedPlan | null {
  const raw = stripFence(content.trim());
  if (!raw.startsWith('---')) return null;
  const rest = raw.slice(3).replace(/^\r?\n/, '');
  const end = rest.indexOf('\n---');
  if (end < 0) return null;
  const fm = rest.slice(0, end);
  const body = rest.slice(end + 4).replace(/^\r?\n/, '').trim();
  const name = matchScalar(fm, 'name');
  const overview = matchScalar(fm, 'overview');
  const todos = parseTodos(fm);
  if (!name && !overview) return null;
  if (todos.length === 0) return null;
  return { name, overview, todos, body, raw: content.trim() };
}

export function planFromMessage(
  metadata?: Record<string, unknown> | null,
  content?: string,
): { planId: string; parsed: ParsedPlan | null } {
  const planId = typeof metadata?.plan_id === 'string' ? metadata.plan_id.trim() : '';
  const parsed = parsePlanMarkdown(content || '');
  return { planId, parsed };
}

export function shouldShowPlanCard(
  metadata?: Record<string, unknown> | null,
  content?: string,
): boolean {
  const { planId, parsed } = planFromMessage(metadata, content);
  return Boolean(planId) || parsed !== null;
}

export function buildPlanBuildMessage(markdown: string): string {
  return (
    'Implement the approved plan. Follow the todos in order. Do not expand scope.\n\n' +
    markdown.trim()
  );
}

export function buildPlanBuildPayload(markdown: string, planId: string): {
  content: string;
  metadata: Record<string, unknown>;
} {
  return {
    content: buildPlanBuildMessage(markdown),
    metadata: {
      composer_mode: 'agent',
      editor_mode: 'agent',
      implementation_session: true,
      plan_id: planId,
    },
  };
}

export function dispatchBuildPlan(detail: PlanBuildRequest): void {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(NJ_BUILD_PLAN_EVENT, { detail }));
}

export function renderPlanWithTodoStatus(markdown: string, todoId: string, status: string): string {
  const parsed = parsePlanMarkdown(markdown);
  if (!parsed) return markdown;
  const lines = markdown.split('\n');
  let inTodos = false;
  let currentId = '';
  const out: string[] = [];
  for (const line of lines) {
    const idMatch = line.match(/^\s+-\s+id:\s*(.+)\s*$/);
    if (idMatch) {
      inTodos = true;
      currentId = unquote(idMatch[1]);
      out.push(line);
      continue;
    }
    if (inTodos && currentId === todoId && /^\s+status:\s*/.test(line)) {
      const indent = line.match(/^\s*/)?.[0] ?? '    ';
      out.push(`${indent}status: ${status}`);
      currentId = '';
      continue;
    }
    out.push(line);
  }
  return out.join('\n');
}

function stripFence(s: string): string {
  if (!s.startsWith('```')) return s;
  const nl = s.indexOf('\n');
  if (nl < 0) return s;
  let rest = s.slice(nl + 1);
  if (rest.endsWith('```')) rest = rest.slice(0, -3).trim();
  return rest.trim();
}

function matchScalar(fm: string, key: string): string {
  const re = new RegExp(`^${key}:\\s*(.*)$`, 'm');
  const m = fm.match(re);
  if (!m) return '';
  return unquote(m[1]);
}

function parseTodos(fm: string): PlanTodo[] {
  const todos: PlanTodo[] = [];
  const lines = fm.split('\n');
  let current: PlanTodo | null = null;
  let inTodos = false;
  for (const line of lines) {
    if (/^todos:\s*$/.test(line)) {
      inTodos = true;
      continue;
    }
    if (inTodos && /^\S/.test(line) && !/^\s/.test(line) && !line.startsWith('-')) {
      if (current) todos.push(current);
      current = null;
      break;
    }
    if (!inTodos) continue;
    const idMatch = line.match(/^\s+-\s+id:\s*(.+)\s*$/);
    if (idMatch) {
      if (current) todos.push(current);
      current = { id: unquote(idMatch[1]), content: '', status: 'pending' };
      continue;
    }
    const contentMatch = line.match(/^\s+content:\s*(.+)\s*$/);
    if (contentMatch && current) current.content = unquote(contentMatch[1]);
    const statusMatch = line.match(/^\s+status:\s*(.+)\s*$/);
    if (statusMatch && current) current.status = unquote(statusMatch[1]);
  }
  if (current) todos.push(current);
  return todos;
}

function unquote(s: string): string {
  const t = s.trim();
  if ((t.startsWith('"') && t.endsWith('"')) || (t.startsWith("'") && t.endsWith("'"))) {
    return t.slice(1, -1).replace(/\\"/g, '"');
  }
  return t;
}
