import { describe, expect, it } from 'vitest';
import {
  formatGovernanceSummary,
  formatRetrievalLabel,
  formatRoutingTelemetryHeadline,
  formatRoutingTelemetrySubline,
  formatTierLabel,
  formatToolTelemetryHeadline,
} from './routingTraceFormat';

describe('routingTraceFormat', () => {
  it('formats retrieval and tier labels', () => {
    expect(formatRetrievalLabel('codebase')).toBe('Codebase');
    expect(formatTierLabel('cheap')).toBe('Cheap');
  });

  it('builds routing telemetry headline', () => {
    expect(
      formatRoutingTelemetryHeadline({
        chat_model: 'qwen3.5:9b',
        cost_tier: 'standard',
        knowledge_route: 'codebase',
      }),
    ).toBe('qwen3.5:9b · Standard · Codebase');
  });

  it('builds routing telemetry subline with governance', () => {
    expect(
      formatRoutingTelemetrySubline({
        reason: 'capability_routing',
        source: 'capabilities',
        governance: { composer_mode: 'agent', context_scope: 'workspace' },
      }),
    ).toBe('capability_routing · capabilities · agent · workspace');
  });

  it('summarizes governance', () => {
    expect(
      formatGovernanceSummary({
        composer_mode: 'ask',
        context_scope: 'channel',
        can_run_impl_session: true,
      }),
    ).toBe('ask · channel · impl');
  });

  it('formats tool telemetry headline with iteration', () => {
    expect(formatToolTelemetryHeadline({ name: 'read_file', kind: 'start', iteration: 2 })).toBe(
      'read_file (start) #2',
    );
  });
});
