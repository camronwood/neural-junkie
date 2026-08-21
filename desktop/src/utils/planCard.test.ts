// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import {
  buildPlanBuildPayload,
  dispatchBuildPlan,
  NJ_BUILD_PLAN_EVENT,
  parsePlanMarkdown,
  renderPlanWithTodoStatus,
  shouldShowPlanCard,
} from './planCard';

const sample = `---
name: HelloWorld plan
overview: Add a HelloWorld helper.
todos:
  - id: add-fn
    content: Add HelloWorld in core/sample/main.go
    status: pending
  - id: verify
    content: Run go test
    status: pending
isProject: false
---

# HelloWorld

## Out of scope

- Extra features
`;

describe('planCard', () => {
  it('parses YAML todos', () => {
    const parsed = parsePlanMarkdown(sample);
    expect(parsed?.name).toBe('HelloWorld plan');
    expect(parsed?.todos).toHaveLength(2);
    expect(parsed?.todos[0].id).toBe('add-fn');
  });

  it('rejects ask-style outlines', () => {
    expect(parsePlanMarkdown('Outline:\n1. Add HelloWorld')).toBeNull();
    expect(shouldShowPlanCard({ editor_mode: 'ask' }, 'Outline:\n1. Add HelloWorld')).toBe(false);
  });

  it('shows a card when plan_id is set even before parse', () => {
    expect(shouldShowPlanCard({ plan_id: 'hello_abc123' }, 'still thinking')).toBe(true);
  });

  it('builds an agent-mode payload', () => {
    const payload = buildPlanBuildPayload(sample, 'hello_abc123');
    expect(payload.metadata.editor_mode).toBe('agent');
    expect(payload.metadata.composer_mode).toBe('agent');
    expect(payload.metadata.implementation_session).toBe(true);
    expect(payload.metadata.plan_id).toBe('hello_abc123');
    expect(payload.content).toContain('Implement the approved plan');
    expect(payload.content).toContain('todos:');
  });

  it('toggles a todo status in markdown', () => {
    const next = renderPlanWithTodoStatus(sample, 'add-fn', 'completed');
    expect(next).toContain('id: add-fn');
    expect(next).toMatch(/id: add-fn[\s\S]*status: completed/);
  });

  it('dispatchBuildPlan fires NJ_BUILD_PLAN_EVENT with correct detail', () => {
    const events: CustomEvent[] = [];
    const handler = (e: Event) => events.push(e as CustomEvent);
    window.addEventListener(NJ_BUILD_PLAN_EVENT, handler);
    dispatchBuildPlan({ markdown: sample, planId: 'hello_abc123' });
    window.removeEventListener(NJ_BUILD_PLAN_EVENT, handler);
    expect(events).toHaveLength(1);
    expect(events[0].detail.planId).toBe('hello_abc123');
    expect(events[0].detail.markdown).toBe(sample);
  });

  it('shouldShowPlanCard returns false when artifact_ref is present (no plan_id)', () => {
    // When a plan has been promoted to an artifact, the message will have
    // artifact_ref but NOT plan_id until the hub stamps both. Ensure the
    // fallback PlanCard is suppressed only via the !artifactRef guard in
    // Message.tsx; shouldShowPlanCard itself does not know about artifact_ref.
    // This test documents that the original helper still returns true for a
    // valid plan so our guard in JSX is the only gate.
    expect(shouldShowPlanCard({ plan_id: 'hello_abc123' }, sample)).toBe(true);
    // A message that has artifact_ref but no plan markdown (no plan_id) still
    // correctly returns false.
    expect(shouldShowPlanCard({ artifact_ref: { id: 'art-1' } }, 'no markdown')).toBe(false);
  });
});
