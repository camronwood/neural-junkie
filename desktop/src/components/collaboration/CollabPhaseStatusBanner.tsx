import type { PlanningGenerationErrorEntry } from '../../utils/collaborationPanelState';

interface CollabPhaseStatusBannerProps {
  phase: string;
  planningGenErrors: PlanningGenerationErrorEntry[];
}

/** Planning-phase model generation error banner (extraction loop 1 from CollaborationPanel). */
export function CollabPhaseStatusBanner({
  phase,
  planningGenErrors,
}: CollabPhaseStatusBannerProps) {
  if (phase !== 'planning' || planningGenErrors.length === 0) {
    return null;
  }
  return (
    <div
      data-testid="collaboration-generation-error-banner"
      style={{
        marginBottom: 16,
        padding: 12,
        borderRadius: 8,
        border: '1px solid #dc2626',
        backgroundColor: 'rgba(220, 38, 38, 0.12)',
      }}
    >
      <div style={{ fontSize: 13, fontWeight: 600, color: '#fca5a5', marginBottom: 6 }}>
        Model generation failed during planning
      </div>
      <ul
        style={{
          margin: 0,
          paddingLeft: 18,
          fontSize: 12,
          color: 'var(--text-secondary, #ccc)',
          lineHeight: 1.5,
        }}
      >
        {planningGenErrors.map((entry) => (
          <li key={entry.agentId}>
            <strong>@{entry.agentName}</strong>: {entry.errorCode}
            {entry.count > 1 ? ` (${entry.count}×)` : ''}
            {entry.turnAdvanced
              ? ' — turn skipped; next participant prompted.'
              : ' — retrying this turn. Check Settings → Collaboration planning provider or model availability.'}
          </li>
        ))}
      </ul>
    </div>
  );
}
