import { lazy, type ComponentType, type ReactNode, Suspense } from 'react';
import { ErrorBoundary } from './ErrorBoundary';

function panelFallback(label = 'Loading…'): ReactNode {
  return (
    <div
      style={{
        padding: 16,
        fontSize: 13,
        color: 'var(--text-secondary, #999)',
      }}
    >
      {label}
    </div>
  );
}

export const LazyFileExplorerPanel = lazy(() =>
  import('./FileExplorerPanel').then((m) => ({ default: m.FileExplorerPanel }))
);
export const LazyCodeEditorPanel = lazy(() =>
  import('./CodeEditorPanel').then((m) => ({ default: m.CodeEditorPanel }))
);
export const LazyCollaborationPanel = lazy(() =>
  import('./CollaborationPanel').then((m) => ({ default: m.CollaborationPanel }))
);
export const LazyRunbookBuilderPanel = lazy(() =>
  import('./RunbookBuilderPanel').then((m) => ({ default: m.RunbookBuilderPanel }))
);
export const LazyRunbookLibraryModal = lazy(() =>
  import('./runbook/RunbookLibraryModal').then((m) => ({ default: m.RunbookLibraryModal }))
);
export const LazySecondaryAnalysisPanel = lazy(() =>
  import('./SecondaryAnalysisPanel').then((m) => ({ default: m.SecondaryAnalysisPanel }))
);
export const LazyTaskManagementPanel = lazy(() =>
  import('./TaskManagementPanel').then((m) => ({ default: m.TaskManagementPanel }))
);
export const LazyDomainPacksModal = lazy(() =>
  import('./DomainPacksModal').then((m) => ({ default: m.DomainPacksModal }))
);
export const LazyModelArenaModal = lazy(() =>
  import('./ModelArenaModal').then((m) => ({ default: m.ModelArenaModal }))
);
export const LazyAIInterviewPrepModal = lazy(() =>
  import('./AIInterviewPrepModal').then((m) => ({ default: m.AIInterviewPrepModal }))
);
export const LazyPhoenixBrowserModal = lazy(() =>
  import('./PhoenixBrowserModal').then((m) => ({ default: m.PhoenixBrowserModal }))
);
export const LazyModelLibraryModal = lazy(() =>
  import('./ModelLibraryModal').then((m) => ({ default: m.ModelLibraryModal }))
);
export const LazyRoomChatModal = lazy(() =>
  import('./RoomChatModal').then((m) => ({ default: m.RoomChatModal }))
);

type LazyPanelShellProps<P extends object> = {
  label?: string;
  panelName: string;
  Component: ComponentType<P>;
  props: P;
};

export function LazyPanelShell<P extends object>({
  label,
  panelName,
  Component,
  props,
}: LazyPanelShellProps<P>) {
  return (
    <ErrorBoundary panelName={panelName}>
      <Suspense fallback={panelFallback(label ?? `Loading ${panelName}…`)}>
        <Component {...props} />
      </Suspense>
    </ErrorBoundary>
  );
}
