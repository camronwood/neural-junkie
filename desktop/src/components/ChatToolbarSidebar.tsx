import type { ReactNode } from 'react';

interface ChatToolbarSidebarProps {
  open: boolean;
  children: ReactNode;
}

/** Narrow right rail for toolbar action chips (compact viewports or user preference). */
export function ChatToolbarSidebar({ open, children }: ChatToolbarSidebarProps) {
  if (!open) return null;

  return (
    <aside
      className="flex flex-col shrink-0 w-14 border-l border-slack-border bg-slack-bgHover overflow-hidden"
      aria-label="Toolbar actions"
    >
      <div className="flex-1 min-h-0 overflow-y-auto overflow-x-hidden py-1">{children}</div>
    </aside>
  );
}
