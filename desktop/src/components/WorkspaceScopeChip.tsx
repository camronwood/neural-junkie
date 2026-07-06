import { useWorkspaceAgentScope } from '../hooks/useWorkspaceAgentScope';

/** Visible badge for multi-repo agent scope in the Files header. */
export function WorkspaceScopeChip() {
  const { scopeLabel } = useWorkspaceAgentScope();

  if (!scopeLabel) return null;

  return (
    <span
      className="px-2 py-0.5 text-[10px] uppercase tracking-wide rounded bg-slack-bgHover text-slack-textMuted whitespace-nowrap truncate max-w-[200px] flex-shrink-0"
      title={`Agent scope: ${scopeLabel}`}
    >
      Scope: {scopeLabel}
    </span>
  );
}
