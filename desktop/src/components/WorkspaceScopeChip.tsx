import { useWorkspaceAgentScope } from '../hooks/useWorkspaceAgentScope';

/** Visible badge for multi-repo agent scope in the Files header. */
export function WorkspaceScopeChip({ variant = 'inline' }: { variant?: 'inline' | 'row' }) {
  const { scopeLabel } = useWorkspaceAgentScope();

  if (!scopeLabel) return null;

  return (
    <span
      className={
        variant === 'row'
          ? 'px-2 py-0.5 text-[10px] uppercase tracking-wide rounded bg-slack-bg text-slack-textMuted whitespace-nowrap truncate min-w-0 max-w-[min(100%,240px)]'
          : 'px-2 py-0.5 text-[10px] uppercase tracking-wide rounded bg-slack-bgHover text-slack-textMuted whitespace-nowrap truncate max-w-[200px] flex-shrink-0'
      }
      title={`Agent scope: ${scopeLabel}`}
    >
      Scope: {scopeLabel}
    </span>
  );
}
