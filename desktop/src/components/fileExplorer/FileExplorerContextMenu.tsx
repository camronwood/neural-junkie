import { ViewportContextMenu } from '../ViewportContextMenu';

export interface FileExplorerContextMenuState {
  x: number;
  y: number;
  path: string;
  isDir: boolean;
  isBackground?: boolean;
}

export interface FileExplorerContextMenuProps {
  contextMenu: FileExplorerContextMenuState;
  onClose: () => void;
  hasScanAnalysis: boolean;
  hasSecondaryAnalysis: boolean;
  hasHtmlBrowserWorkbench: boolean;
  hasCadWorkbench: boolean;
  hasScanSummary: boolean;
  contextMenuIsScanAnalysis: () => boolean;
  contextMenuIsComparator: () => boolean;
  contextMenuIsScanSummary: () => boolean;
  onRevealInFinder: () => void;
  onOpenInTerminal: () => void;
  onCreateFile: () => void;
  onCreateFolder: () => void;
  onOpenFromMenu: () => void;
  onCopyName: () => void;
  onCopyPath: () => void;
  onCopyRelativePath: () => void;
  onDuplicate: () => void;
  onRename: () => void;
  onDelete: () => void;
  onOpenScanAnalysis: () => void;
  onRun12PlexQC: () => void;
  onAddToAnalysisBasket: () => void;
  onOpenSecondaryPanel: () => void;
  onOpenComparator: () => void;
  onOpenHtml: () => void;
  onOpenCad: () => void;
  onOpenScanSummary: () => void;
  onPreviewMarkdown: () => void;
}

/** Portaled explorer context menu (file + background). */
export function FileExplorerContextMenu({
  contextMenu,
  onClose,
  hasScanAnalysis,
  hasSecondaryAnalysis,
  hasHtmlBrowserWorkbench,
  hasCadWorkbench,
  hasScanSummary,
  contextMenuIsScanAnalysis,
  contextMenuIsComparator,
  contextMenuIsScanSummary,
  onRevealInFinder,
  onOpenInTerminal,
  onCreateFile,
  onCreateFolder,
  onOpenFromMenu,
  onCopyName,
  onCopyPath,
  onCopyRelativePath,
  onDuplicate,
  onRename,
  onDelete,
  onOpenScanAnalysis,
  onRun12PlexQC,
  onAddToAnalysisBasket,
  onOpenSecondaryPanel,
  onOpenComparator,
  onOpenHtml,
  onOpenCad,
  onOpenScanSummary,
  onPreviewMarkdown,
}: FileExplorerContextMenuProps) {
  return (
    <ViewportContextMenu x={contextMenu.x} y={contextMenu.y} onClose={onClose}>
      {contextMenu.isBackground ? (
        <>
          <button
            type="button"
            onClick={() => void onRevealInFinder()}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Reveal in Finder
          </button>
          <button
            type="button"
            onClick={onOpenInTerminal}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Open in terminal
          </button>
          <div className="border-t border-slack-border my-1" />
          <button
            type="button"
            onClick={onCreateFile}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            New file…
          </button>
          <button
            type="button"
            onClick={onCreateFolder}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            New folder…
          </button>
        </>
      ) : (
        <>
          {!contextMenu.isDir && (
            <button
              type="button"
              onClick={() => void onOpenFromMenu()}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              Open
            </button>
          )}

          <button
            type="button"
            onClick={() => void onCopyName()}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Copy name
          </button>
          <button
            type="button"
            onClick={onCopyPath}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Copy absolute path
          </button>
          <button
            type="button"
            onClick={onCopyRelativePath}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Copy relative path
          </button>

          <div className="border-t border-slack-border my-1" />

          <button
            type="button"
            onClick={() => void onRevealInFinder()}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Reveal in Finder
          </button>
          <button
            type="button"
            onClick={onOpenInTerminal}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Open in terminal
          </button>

          {!contextMenu.isDir && (
            <button
              type="button"
              onClick={() => void onDuplicate()}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              Duplicate
            </button>
          )}

          <div className="border-t border-slack-border my-1" />

          <button
            type="button"
            onClick={onCreateFile}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            New file…
          </button>
          <button
            type="button"
            onClick={onCreateFolder}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            New folder…
          </button>
          <button
            type="button"
            onClick={onRename}
            className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
          >
            Rename…
          </button>
          <button
            type="button"
            onClick={onDelete}
            className="w-full px-4 py-2 text-left text-sm text-red-500 hover:bg-slack-bgHover"
          >
            Delete
          </button>

          {(hasScanAnalysis && contextMenuIsScanAnalysis()) ||
          (hasSecondaryAnalysis && contextMenuIsScanAnalysis() && contextMenu?.isDir) ||
          (hasSecondaryAnalysis && contextMenuIsComparator()) ||
          (hasHtmlBrowserWorkbench && !contextMenu.isDir && /\.html?$/i.test(contextMenu.path)) ||
          (hasCadWorkbench && !contextMenu.isDir && contextMenu.path.toLowerCase().endsWith('.scad')) ||
          (hasScanSummary && contextMenuIsScanSummary()) ||
          (!contextMenu.isDir && contextMenu.path.toLowerCase().endsWith('.md')) ? (
            <div className="border-t border-slack-border my-1" />
          ) : null}

          {hasScanAnalysis && contextMenuIsScanAnalysis() && (
            <button
              onClick={() => void onOpenScanAnalysis()}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              📊 Open scan analysis
            </button>
          )}

          {hasSecondaryAnalysis && contextMenuIsScanAnalysis() && contextMenu?.isDir && (
            <>
              <button
                onClick={() => void onRun12PlexQC()}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                ✅ Run 12-Plex QC
              </button>
              <button
                onClick={onAddToAnalysisBasket}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                🧪 Add to analysis basket
              </button>
              <button
                onClick={onOpenSecondaryPanel}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                📋 Secondary analysis panel…
              </button>
            </>
          )}

          {hasSecondaryAnalysis && contextMenuIsComparator() && (
            <>
              <button
                onClick={onOpenComparator}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                📈 Open comparator analysis
              </button>
              <button
                onClick={onOpenSecondaryPanel}
                className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
              >
                🧬 Run endogenous analysis…
              </button>
            </>
          )}

          {hasHtmlBrowserWorkbench && !contextMenu.isDir && /\.html?$/i.test(contextMenu.path) && (
            <button
              onClick={() => void onOpenHtml()}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              🌐 Open HTML browser
            </button>
          )}

          {hasCadWorkbench && !contextMenu.isDir && contextMenu.path.toLowerCase().endsWith('.scad') && (
            <button
              onClick={() => void onOpenCad()}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              📐 Open CAD workbench
            </button>
          )}

          {hasScanSummary && contextMenuIsScanSummary() && (
            <button
              onClick={onOpenScanSummary}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              🔬 Open scan summary
            </button>
          )}

          {!contextMenu.isDir && contextMenu.path.toLowerCase().endsWith('.md') && (
            <button
              onClick={onPreviewMarkdown}
              className="w-full px-4 py-2 text-left text-sm text-slack-text hover:bg-slack-bgHover"
            >
              Preview Markdown
            </button>
          )}
        </>
      )}
    </ViewportContextMenu>
  );
}
