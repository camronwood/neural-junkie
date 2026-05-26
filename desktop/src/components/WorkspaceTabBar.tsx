import { useEffect, useRef } from 'react';
import type { Workspace } from '../stores/fileExplorerStore';
import { workspacesForTabBar } from '../utils/workspaceOrder';

export interface WorkspaceTabBarProps {
  workspaces: Workspace[];
  activeWorkspaceId: string | null;
  onSelect: (id: string) => void;
  onRemove: (e: React.MouseEvent, id: string, name: string) => void;
}

export function WorkspaceTabBar({
  workspaces,
  activeWorkspaceId,
  onSelect,
  onRemove,
}: WorkspaceTabBarProps) {
  const activeRef = useRef<HTMLDivElement>(null);
  const { visible } = workspacesForTabBar(workspaces, activeWorkspaceId);

  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }, [activeWorkspaceId]);

  if (workspaces.length === 0) {
    return null;
  }

  return (
    <div className="flex items-center gap-1 min-w-0 overflow-x-auto">
      {visible.map((workspace) => {
        const isActive = activeWorkspaceId === workspace.id;
        return (
          <div
            key={workspace.id}
            ref={isActive ? activeRef : undefined}
            onClick={() => onSelect(workspace.id)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                onSelect(workspace.id);
              }
            }}
            role="button"
            tabIndex={0}
            className={`group flex items-center gap-1 px-3 py-1 text-xs rounded transition-colors whitespace-nowrap cursor-pointer flex-shrink-0 ${
              isActive
                ? 'bg-slack-accent text-white'
                : 'bg-slack-bgHover text-slack-textMuted hover:text-slack-text'
            }`}
            title={workspace.path}
          >
            <span>{workspace.name}</span>
            <button
              type="button"
              onClick={(e) => onRemove(e, workspace.id, workspace.name)}
              className={`ml-1 p-0.5 rounded-sm opacity-0 group-hover:opacity-100 focus:opacity-100 transition-opacity ${
                isActive ? 'hover:bg-white/20' : 'hover:bg-slack-border'
              }`}
              title={`Remove ${workspace.name}`}
              aria-label={`Remove ${workspace.name}`}
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        );
      })}
    </div>
  );
}
