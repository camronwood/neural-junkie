import { useEffect, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import {
  dispatchBuildPlan,
  parsePlanMarkdown,
  renderPlanWithTodoStatus,
  shouldShowPlanCard,
  type ParsedPlan,
} from '../utils/planCard';

export function PlanCard({
  content,
  metadata,
}: {
  content: string;
  metadata?: Record<string, unknown>;
}) {
  const planId = typeof metadata?.plan_id === 'string' ? metadata.plan_id.trim() : '';
  const [parsed, setParsed] = useState<ParsedPlan | null>(() => parsePlanMarkdown(content));
  const [markdown, setMarkdown] = useState(content);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    if (!planId) {
      setParsed(parsePlanMarkdown(content));
      setMarkdown(content);
      return;
    }
    const api = new ChatAPI();
    void api
      .getPlan(planId)
      .then((rec) => {
        if (cancelled) return;
        setMarkdown(rec.markdown);
        setParsed(
          parsePlanMarkdown(rec.markdown) ?? {
            name: rec.name,
            overview: rec.overview,
            todos: rec.todos,
            body: rec.markdown,
            raw: rec.markdown,
          },
        );
      })
      .catch(() => {
        if (!cancelled) setParsed(parsePlanMarkdown(content));
      });
    return () => {
      cancelled = true;
    };
  }, [planId, content]);

  if (!shouldShowPlanCard(metadata, content) && !parsed) {
    return null;
  }
  if (!parsed) return null;

  const toggle = async (todoId: string, current: string) => {
    if (!planId) return;
    const nextStatus = current === 'completed' ? 'pending' : 'completed';
    const nextMd = renderPlanWithTodoStatus(markdown, todoId, nextStatus);
    setBusy(true);
    try {
      const api = new ChatAPI();
      const rec = await api.putPlan(planId, nextMd);
      setMarkdown(rec.markdown);
      setParsed(parsePlanMarkdown(rec.markdown));
    } catch {
      /* keep local view */
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      data-testid="plan-card"
      className="mt-3 rounded-md border border-teal-700/40 bg-slack-bgHover/70 p-3 text-sm"
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-xs uppercase tracking-wide text-teal-300 font-semibold">Plan</div>
          <div className="font-medium text-slack-text mt-0.5">{parsed.name || 'Implementation plan'}</div>
          {parsed.overview ? (
            <p className="text-slack-textMuted mt-1 text-xs">{parsed.overview}</p>
          ) : null}
        </div>
        <button
          type="button"
          data-testid="plan-card-build"
          className="shrink-0 px-3 py-1 rounded bg-teal-700 hover:bg-teal-600 text-white text-xs font-semibold"
          onClick={() =>
            dispatchBuildPlan({ markdown, planId: planId || parsed.name || 'plan' })
          }
        >
          Build
        </button>
      </div>
      <ul className="mt-2 space-y-1">
        {parsed.todos.map((todo) => (
          <li key={todo.id} className="flex items-start gap-2 text-xs text-slack-text">
            <input
              type="checkbox"
              checked={todo.status === 'completed'}
              disabled={busy || !planId}
              onChange={() => void toggle(todo.id, todo.status)}
              className="mt-0.5"
            />
            <span className={todo.status === 'completed' ? 'line-through text-slack-textMuted' : ''}>
              {todo.content || todo.id}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
