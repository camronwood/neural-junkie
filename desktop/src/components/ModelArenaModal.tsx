import { usePacksStore } from '../stores/packsStore';
import { PACK_CAP } from '../stores/packCapabilities';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { ArenaWorkbench } from './ArenaWorkbench';
import './arena/arenaRetro.css';

interface ModelArenaModalProps {
  isOpen: boolean;
  onClose: () => void;
  /** Opens the arena workbench in the IDE editor panel (also shows the panel). */
  onOpenInEditor?: (workspaceId: string) => void;
}

export function ModelArenaModal({ isOpen, onClose, onOpenInEditor }: ModelArenaModalProps) {
  const hasArena = usePacksStore(
    (s) =>
      s.hasCapability(PACK_CAP.MODEL_ARENA) ||
      s.hasCapability(PACK_CAP.MODEL_ARENA_WORKBENCH) ||
      s.hasCapability('model-arena-launcher'),
  );
  const activeWorkspaceId = useFileExplorerStore((s) => s.activeWorkspaceId);
  const openArenaWorkbench = useEditorStore((s) => s.openArenaWorkbench);
  const workspaceId = activeWorkspaceId || 'default';
  const arenaSessionPath = 'arena/model-arena.nj-arena.json';

  const handleOpenInEditor = () => {
    if (onOpenInEditor) {
      onOpenInEditor(workspaceId);
    } else {
      openArenaWorkbench(workspaceId, arenaSessionPath);
    }
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto py-6 px-4" role="presentation">
      <div className="fixed inset-0 bg-black/75" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="model-arena-title"
        className="arena-retro arena-modal-shell relative z-10 flex w-full max-w-5xl flex-col overflow-hidden max-h-[min(92vh,900px)]"
      >
        <div className="arena-retro-marquee flex shrink-0 items-center justify-between gap-3">
          <div>
            <h2 id="model-arena-title" className="arena-retro-title">
              MODEL ARENA
            </h2>
            <p className="arena-retro-subtitle">Chess · Connect 4 · Logic</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleOpenInEditor}
              className="arena-retro-btn secondary"
              title="Open Model Arena in the editor panel"
            >
              Open in editor
            </button>
            <button type="button" onClick={onClose} className="arena-retro-btn danger px-3" aria-label="Close">
              X
            </button>
          </div>
        </div>
        {/* min-h-0 + overflow-y-auto: keep roster/toolbar reachable after long games */}
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
          {!hasArena && (
            <div className="arena-retro-body p-6 text-sm text-slate-300">
              Enable the <strong className="text-amber-300">Model Arena</strong> pack in Settings → Domain packs.
            </div>
          )}
          {hasArena && (
            <ArenaWorkbench workspaceId={workspaceId} tabId="model-arena-modal" showHeader={false} />
          )}
        </div>
      </div>
    </div>
  );
}
