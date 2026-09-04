import { useState, useEffect } from 'react';
import { useToastStore } from '../stores/toastStore';
import { resolveEditorPdfSrc } from '../utils/chatImageSrc';
import { workspaceAbsolutePath } from '../utils/editorFileKind';
import { useFileExplorerStore } from '../stores/fileExplorerStore';

interface EditorPdfPreviewProps {
  src: string;
  alt: string;
  reloadKey: number;
  workspaceId: string;
  relativePath: string;
}

export function EditorPdfPreview({
  src: initialSrc,
  alt,
  reloadKey,
  workspaceId,
  relativePath,
}: EditorPdfPreviewProps) {
  const { addToast } = useToastStore();
  const [src, setSrc] = useState(initialSrc);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    setSrc(initialSrc);
    setFailed(false);
  }, [initialSrc]);

  useEffect(() => {
    if (reloadKey === 0) return;
    let cancelled = false;
    void (async () => {
      try {
        const ws = useFileExplorerStore.getState().workspaces.find((w) => w.id === workspaceId);
        const absolutePath = ws ? workspaceAbsolutePath(ws.path, relativePath) : relativePath;
        const next = await resolveEditorPdfSrc({ workspaceId, relativePath, absolutePath });
        if (!cancelled) {
          setSrc(next);
          setFailed(false);
        }
      } catch {
        if (!cancelled) {
          setFailed(true);
          addToast({
            type: 'error',
            title: 'Could not reload PDF',
            message: `Failed to reload preview for ${alt}.`,
          });
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [reloadKey, workspaceId, relativePath, alt, addToast]);

  if (failed) {
    return (
      <div className="flex items-center justify-center h-full text-slack-textMuted p-6">
        <div className="text-center max-w-sm">
          <div className="text-4xl mb-3">📄</div>
          <div className="text-sm font-medium mb-1">PDF preview unavailable</div>
          <div className="text-xs break-all">{alt}</div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full w-full bg-slack-bg">
      <iframe
        key={`${reloadKey}:${src.slice(0, 48)}`}
        title={alt}
        src={src}
        className="w-full h-full border-0"
      />
    </div>
  );
}
