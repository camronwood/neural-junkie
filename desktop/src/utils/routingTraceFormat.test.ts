import { describe, expect, it } from 'vitest';
import {
  formatContextSelectionSummary,
  formatGovernanceSummary,
  formatOmissionReasons,
  formatRetrievalLabel,
  formatRoutingTelemetryHeadline,
  formatRoutingTelemetrySubline,
  formatRoutingAttempt,
  formatTierLabel,
  formatToolTelemetryHeadline,
} from './routingTraceFormat';
import { getRoutingMeta } from '../types/protocol';

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

  it('formats context selection, recovery attempts, and omission reasons', () => {
    const selection = {
      selected_sections: ['summary', 'recent_exchanges'],
      selected_context_ids: ['msg-1', 'msg-2'],
      dropped_context_ids: ['old-1'],
      digest_version: 4,
      omission_reasons: { 'old-1': 'superseded' },
      budget_omission_reasons: { workspace: 'compressed_to_budget' },
    };
    expect(formatContextSelectionSummary(selection)).toBe(
      'summary, recent_exchanges · 2 selected · 1 omitted · digest v4'
    );
    expect(formatOmissionReasons(selection)).toEqual([
      'old-1: superseded',
      'workspace: compressed_to_budget',
    ]);
    expect(
      formatRoutingAttempt({
        model: 'qwen3.5:9b',
        tier: 'reliable',
        failure_reason: 'quality_gate_failure',
      })
    ).toBe('qwen3.5:9b · reliable · failed: quality_gate_failure');
  });

  it('parses model attempts and failure evidence from response metadata', () => {
    expect(
      getRoutingMeta({
        routing_model: 'qwen3.5:9b',
        routing_attempts: [
          { provider_id: 'ollama-local', model: 'qwen2.5:3b', failure_reason: 'empty_response' },
          { provider_id: 'ollama-local', model: 'qwen3.5:9b', reason: 'local_escalation' },
        ],
        routing_failure_evidence: ['empty_response'],
      })
    ).toMatchObject({
      attempts: [{ model: 'qwen2.5:3b' }, { model: 'qwen3.5:9b' }],
      failure_evidence: ['empty_response'],
    });
  });
});
