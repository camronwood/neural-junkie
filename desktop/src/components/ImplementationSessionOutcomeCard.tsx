import { useState } from 'react';
import type { ImplementationSessionOutcome } from '../types/protocol';
import { IMPLEMENTATION_SESSION_OUTCOME_KEY } from '../constants/promptMetadata';

function outcomeLabel(outcome: string | undefined): string {
  switch (outcome) {
    case 'applied_and_verified':
      return 'Applied and verified';
    case 'applied_verify_failed':
      return 'Applied — verify failed';
    case 'proposals_submitted':
      return 'Proposals submitted';
    case 'proposal_registration_failed':
      return 'Proposal registration failed';
    case 'wrong_route':
      return 'Wrong specialist route';
    case 'no_changes':
      return 'No file changes';
    default:
      return outcome?.replace(/_/g, ' ') ?? 'Session outcome';
  }
}

function outcomeTone(outcome: ImplementationSessionOutcome): string {
  if (outcome.circuit_breaker_triggered || outcome.verify_failed || outcome.failure_type === 'timeout') {
    return 'border-amber-500/40 bg-amber-500/10 text-amber-200';
  }
  if (outcome.outcome === 'applied_and_verified') {
    return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-200';
  }
  if (outcome.outcome === 'wrong_route') {
    return 'border-purple-500/40 bg-purple-500/10 text-purple-200';
  }
  return 'border-slack-border bg-slack-bgHover text-slack-textMuted';
}

export function parseImplementationSessionOutcome(
  metadata: Record<string, unknown> | undefined,
): ImplementationSessionOutcome | null {
  if (!metadata) return null;
  const raw = metadata[IMPLEMENTATION_SESSION_OUTCOME_KEY];
  if (!raw || typeof raw !== 'object') return null;
  return raw as ImplementationSessionOutcome;
}

export function ImplementationSessionOutcomeCard({
  outcome,
}: {
  outcome: ImplementationSessionOutcome;
}) {
  const [expanded, setExpanded] = useState(false);
  const tone = outcomeTone(outcome);
  const summaryParts: string[] = [];
  if (outcome.failure_type) summaryParts.push(outcome.failure_type);
  if (outcome.playbook_used) summaryParts.push(`playbook: ${outcome.playbook_used}`);
  if (outcome.circuit_breaker_triggered) summaryParts.push('circuit breaker');
  if (typeof outcome.repair_attempts === 'number' && outcome.repair_attempts > 0) {
    summaryParts.push(`${outcome.repair_attempts} repair(s)`);
  }

  return (
    <div className={`mt-2 rounded border text-xs ${tone}`}>
      <button
        type="button"
        className="w-full flex items-center justify-between gap-2 px-2 py-1.5 text-left hover:opacity-90"
        onClick={() => setExpanded((v) => !v)}
        aria-expanded={expanded}
      >
        <span className="font-medium">Implementation: {outcomeLabel(outcome.outcome)}</span>
        <span className="text-[10px] opacity-80">{expanded ? '▾' : '▸'}</span>
      </button>
      {summaryParts.length > 0 && !expanded && (
        <div className="px-2 pb-1.5 opacity-90">{summaryParts.join(' · ')}</div>
      )}
      {expanded && (
        <div className="px-2 pb-2 space-y-1 font-mono text-[11px] border-t border-white/10 pt-1.5">
          {outcome.suggested_agent && <div>suggested: {outcome.suggested_agent}</div>}
          {outcome.routing_reason && <div>routing: {outcome.routing_reason}</div>}
          {outcome.routing_tool_model && <div>tool model: {outcome.routing_tool_model}</div>}
          {outcome.repro_command && <div>repro: {outcome.repro_command}</div>}
          {Array.isArray(outcome.command_failures) && outcome.command_failures.length > 0 && (
            <div>
              command failures:
              <ul className="list-disc pl-4">
                {outcome.command_failures.map((row, i) => (
                  <li key={i}>
                    {row.cmd} (×{row.count})
                  </li>
                ))}
              </ul>
            </div>
          )}
          {Array.isArray(outcome.files_changed) && outcome.files_changed.length > 0 && (
            <div>files: {outcome.files_changed.join(', ')}</div>
          )}
        </div>
      )}
    </div>
  );
}
