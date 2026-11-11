import type { PlanningGenerationErrorEntry } from '../../utils/collaborationPanelState';
import { CollabAdvancedDebug } from './CollabAdvancedDebug';
import { CollabPhaseStatusBanner } from './CollabPhaseStatusBanner';

const amberBannerStyle = {
  marginBottom: 16,
  padding: 12,
  borderRadius: 8,
  border: '1px solid #b45309',
  backgroundColor: 'rgba(180, 83, 9, 0.12)',
} as const;

interface CollabPhaseStatusStackProps {
  phase: string;
  collabIdPrefix: string;
  planningGenErrors: PlanningGenerationErrorEntry[];
  planningStalled: string[];
  planningAwaitingFirstTurn: boolean;
  executingStuck: boolean;
}

/**
 * Planning/executing status banners (gen-error, stall, wait, stuck).
 * Extraction loop 2 from CollaborationPanel.
 */
export function CollabPhaseStatusStack({
  phase,
  collabIdPrefix,
  planningGenErrors,
  planningStalled,
  planningAwaitingFirstTurn,
  executingStuck,
}: CollabPhaseStatusStackProps) {
  return (
    <>
      <CollabPhaseStatusBanner phase={phase} planningGenErrors={planningGenErrors} />

      {phase === 'planning' && planningStalled.length > 0 && planningGenErrors.length === 0 && (
        <div data-testid="collaboration-planning-stall-banner" style={amberBannerStyle}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
            Waiting on {planningStalled.map((n) => `@${n}`).join(', ')}
          </div>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            These participants have not posted this round. Confirm agents are online in the sidebar, and check
            Settings → Collaboration planning provider (or that Ollama is running).
          </p>
          <CollabAdvancedDebug
            collabIdPrefix={collabIdPrefix}
            testId="collaboration-planning-stall-advanced"
          />
        </div>
      )}

      {phase === 'planning' && planningAwaitingFirstTurn && (
        <div data-testid="collaboration-planning-wait-banner" style={amberBannerStyle}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
            Waiting for the first agent turn
          </div>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            The hub prompted the first participant. If nothing appears after ~30s, confirm agents are online and
            the planning model is reachable (Settings → Collaboration planning provider).
          </p>
          <CollabAdvancedDebug testId="collaboration-planning-wait-advanced" />
        </div>
      )}

      {phase === 'executing' && executingStuck && (
        <div data-testid="collaboration-executing-stuck-banner" style={amberBannerStyle}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#fcd34d', marginBottom: 6 }}>
            Execution may be stuck
          </div>
          <p style={{ margin: 0, fontSize: 12, color: 'var(--text-secondary, #ccc)', lineHeight: 1.45 }}>
            No tasks completed yet while one is in progress. Try <strong>Resume plan</strong> or{' '}
            <strong>Redispatch</strong> / <strong>Reassign</strong> the in-progress task. Confirm the assignee
            agent is online.
          </p>
          <CollabAdvancedDebug
            collabIdPrefix={collabIdPrefix}
            testId="collaboration-executing-stuck-advanced"
          />
        </div>
      )}
    </>
  );
}
