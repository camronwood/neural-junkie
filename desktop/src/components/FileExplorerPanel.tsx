import { FileExplorerTree } from './fileExplorer/FileExplorerTree';
import { FileExplorerContextMenu } from './fileExplorer/FileExplorerContextMenu';
import { FileExplorerAddWorkspaceModal } from './fileExplorer/FileExplorerAddWorkspaceModal';
import { FileExplorerRemoveConfirm } from './fileExplorer/FileExplorerRemoveConfirm';
import { useFileExplorerPanelModel } from './fileExplorer/useFileExplorerPanelModel';
import { WorkspaceSwitcherModal } from './WorkspaceSwitcherModal';
import { WorkspaceTabBar } from './WorkspaceTabBar';
import { FilesPanelMenu } from './FilesPanelMenu';
import { WorkspaceScopeChip } from './WorkspaceScopeChip';
import { FileNameDialog } from './FileNameDialog';
import { shrinkablePanelStyle } from '../utils/panelLayout';

import type { FileExplorerPanelProps } from './fileExplorer/types';

export type { FileExplorerPanelProps };

export function FileExplorerPanel(props: FileExplorerPanelProps) {
  const {
    COMPACT_MIN_WIDTH,
    MIN_WIDTH,
    activeIsCombinedRun,
    activeIsScanAnalysisRoot,
    activeIsScanSummaryRoot,
    activeWorkspaceId,
    canSwitchWorkspaces,
    clearError,
    closeContextMenu,
    confirmRemoveWorkspace,
    contextMenu,
    contextMenuIsComparator,
    contextMenuIsScanAnalysis,
    contextMenuIsScanSummary,
    devcontainerPlan,
    devcontainerPlanLoading,
    devcontainerRepoPath,
    embedded,
    error,
    expandedPaths,
    files,
    getActiveWorkspace,
    handleAddToAnalysisBasket,
    handleAddWorkspace,
    handleBackgroundContextMenu,
    handleBrowseDirectory,
    handleContextMenu,
    handleCopyName,
    handleCopyPath,
    handleCopyRelativePath,
    handleCreateFile,
    handleCreateFolder,
    handleDelete,
    handleDuplicate,
    handleFileClick,
    handleFileDragEnd,
    handleFileDragStart,
    handleLoadDevcontainerPlan,
    handleNameDialogConfirm,
    handleOpenCadFromMenu,
    handleOpenComparatorFromMenu,
    handleOpenFromMenu,
    handleOpenHtmlFromMenu,
    handleOpenInTerminal,
    handleOpenScanAnalysisFromMenu,
    handleOpenScanSummaryFromMenu,
    handleOpenSecondaryPanel,
    handlePreviewMarkdown,
    handleRemoveWorkspace,
    handleRename,
    handleResizeStart,
    handleRevealInFinder,
    handleRun12PlexQCFromMenu,
    hasCadWorkbench,
    hasHtmlBrowserWorkbench,
    hasScanAnalysis,
    hasScanSummary,
    hasSecondaryAnalysis,
    isPathSelected,
    loadingFiles,
    nameDialog,
    newWorkspaceName,
    newWorkspaceParentPath,
    newWorkspacePath,
    onClose,
    openCreateAtRoot,
    openScanAnalysisAtPath,
    openScanSummaryAtPath,
    pendingRemove,
    remoteHost,
    remotePath,
    remoteUser,
    resetAddWorkspaceForm,
    setActiveWorkspace,
    setDevcontainerPlan,
    setDevcontainerRepoPath,
    setNameDialog,
    setNewWorkspaceName,
    setNewWorkspaceParentPath,
    setNewWorkspacePath,
    setPendingRemove,
    setRemoteHost,
    setRemotePath,
    setRemoteUser,
    setShowAddWorkspace,
    setShowWorkspaceSwitcher,
    setSidecarToken,
    setSidecarUrl,
    setWorkspaceAddMode,
    showAddWorkspace,
    showWorkspaceSwitcher,
    sidecarToken,
    sidecarUrl,
    width,
    workspaceAddMode,
    workspaceSwitcherOverflow,
    workspaces,
  } = useFileExplorerPanelModel(props);



  return (
    <div 
      className={
        embedded
          ? 'border-r border-slack-border bg-slack-bg flex flex-col h-full relative'
          : 'border-r border-slack-border bg-slack-bg flex flex-col h-full relative animate-slide-in-left'
      }
      style={shrinkablePanelStyle(width, embedded ? MIN_WIDTH : COMPACT_MIN_WIDTH)}
    >
        <div
          className="absolute right-0 top-0 bottom-0 cursor-col-resize z-[100] group"
          onMouseDown={handleResizeStart}
          aria-label="Resize file explorer panel"
          style={{ 
            width: '6px', 
            marginRight: '-3px',
            pointerEvents: 'auto',
          }}
        >
          <div className="absolute inset-0 bg-transparent group-hover:bg-blue-500/30 transition-colors" />
          <div className="absolute right-1/2 top-1/2 -translate-y-1/2 translate-x-1/2 w-1 h-8 bg-gray-400 group-hover:bg-blue-500 rounded-full opacity-0 group-hover:opacity-100 transition-opacity" />
        </div>
      
      {/* Header */}
      <div className="border-b border-slack-border bg-slack-bgHover min-w-0">
        <div className="px-4 py-2 flex items-center justify-between gap-2 min-w-0">
          <h2 className="font-bold text-slack-text flex-shrink-0">Files</h2>
          <div className="flex items-center gap-2 flex-shrink-0">
            {canSwitchWorkspaces && (
              <button
                type="button"
                onClick={() => setShowWorkspaceSwitcher(true)}
                className="text-slack-textMuted hover:text-slack-text transition-colors flex-shrink-0 px-1 py-0.5 text-xs font-medium"
                title="All workspaces"
                aria-label={
                  workspaceSwitcherOverflow > 0
                    ? `All workspaces, ${workspaceSwitcherOverflow} not shown in tabs`
                    : 'All workspaces'
                }
              >
                {workspaceSwitcherOverflow > 0 ? `... +${workspaceSwitcherOverflow}` : '...'}
              </button>
            )}
            <button
              onClick={() => setShowAddWorkspace(true)}
              className="text-slack-textMuted hover:text-slack-text transition-colors"
              title="Add workspace"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
            </button>
            <button
              onClick={onClose}
              className="text-slack-textMuted hover:text-slack-text transition-colors"
              title="Close file explorer"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        <div className="px-4 py-1.5 border-t border-slack-border flex items-center justify-center gap-2 min-w-0">
          <WorkspaceScopeChip variant="row" />
          <FilesPanelMenu repoPath={getActiveWorkspace()?.path} variant="bar" />
        </div>
      </div>

      {(activeIsScanSummaryRoot || activeIsCombinedRun) && activeWorkspaceId && (
        <div className="px-4 py-2 border-b border-slack-border bg-slack-bg">
          <button
            type="button"
            onClick={() => void openScanSummaryAtPath(activeWorkspaceId, '')}
            className="w-full px-3 py-1.5 text-xs font-medium rounded bg-slack-accent/20 text-slack-accent hover:bg-slack-accent hover:text-white transition-colors"
          >
            Open scan summary
          </button>
        </div>
      )}

      {(activeIsScanAnalysisRoot || activeIsCombinedRun) && activeWorkspaceId && (
        <div className="px-4 py-2 border-b border-slack-border bg-slack-bg">
          <button
            type="button"
            onClick={() => void openScanAnalysisAtPath(activeWorkspaceId, '')}
            className="w-full px-3 py-1.5 text-xs font-medium rounded bg-purple-600/20 text-purple-300 hover:bg-purple-600 hover:text-white transition-colors"
          >
            {activeIsCombinedRun ? 'Open combined run (analysis)' : 'Open scan analysis'}
          </button>
        </div>
      )}

      {/* Workspace Tabs */}
      <div className="px-4 py-2 border-b border-slack-border bg-slack-bgHover flex items-center min-w-0">
        <WorkspaceTabBar
          workspaces={workspaces}
          activeWorkspaceId={activeWorkspaceId}
          onSelect={setActiveWorkspace}
          onRemove={handleRemoveWorkspace}
        />
      </div>

      {/* File Tree */}
      <div
        className="flex-1 overflow-y-auto overflow-x-hidden scrollbar-none"
        onContextMenu={handleBackgroundContextMenu}
      >
        {error ? (
          <div className="px-3 py-2 border-b border-red-900/40 bg-red-950/30 flex items-start gap-2">
            <div className="flex-1 min-w-0">
              <div className="text-xs text-red-400 break-words">{error}</div>
            </div>
            <button
              type="button"
              onClick={clearError}
              className="flex-shrink-0 px-2 py-0.5 bg-red-600 hover:bg-red-700 text-white text-[10px] rounded transition-colors"
            >
              Dismiss
            </button>
          </div>
        ) : null}
        {loadingFiles && files.length === 0 ? (
          <div className="flex items-center justify-center h-32">
            <div className="flex items-center gap-2 text-slack-textMuted">
              <div className="w-4 h-4 border border-slack-textMuted border-t-transparent rounded-full animate-spin"></div>
              Loading files...
            </div>
          </div>
        ) : files.length === 0 ? (
          <div className="p-4 text-center">
            <div className="text-4xl mb-2">📁</div>
            <div className="text-sm text-slack-textMuted mb-3">No files yet</div>
            <div className="text-xs text-slack-textMuted mb-4">
              Right-click here, or create something to get started.
            </div>
            {activeWorkspaceId ? (
              <div className="flex items-center justify-center gap-2">
                <button
                  type="button"
                  onClick={() => openCreateAtRoot('create-file')}
                  className="px-3 py-1.5 text-xs font-medium rounded bg-slack-accent text-white hover:opacity-90 transition-opacity"
                >
                  New file…
                </button>
                <button
                  type="button"
                  onClick={() => openCreateAtRoot('create-folder')}
                  className="px-3 py-1.5 text-xs font-medium rounded border border-slack-border text-slack-text hover:bg-slack-bgHover transition-colors"
                >
                  New folder…
                </button>
              </div>
            ) : null}
          </div>
        ) : (
          <div className="py-2 min-h-full">
            <FileExplorerTree
              files={files}
              expandedPaths={expandedPaths}
              isPathSelected={isPathSelected}
              onFileClick={handleFileClick}
              onContextMenu={handleContextMenu}
              onDragStart={handleFileDragStart}
              onDragEnd={handleFileDragEnd}
            />
          </div>
        )}
      </div>

      <WorkspaceSwitcherModal
        isOpen={showWorkspaceSwitcher}
        onClose={() => setShowWorkspaceSwitcher(false)}
        workspaces={workspaces}
        activeWorkspaceId={activeWorkspaceId}
        onSelect={setActiveWorkspace}
        onRemoveRequest={(id, name) => {
          setShowWorkspaceSwitcher(false);
          setPendingRemove({ id, name });
        }}
      />

      <FileExplorerAddWorkspaceModal
        open={showAddWorkspace}
        workspaceAddMode={workspaceAddMode}
        setWorkspaceAddMode={setWorkspaceAddMode}
        newWorkspaceName={newWorkspaceName}
        setNewWorkspaceName={setNewWorkspaceName}
        newWorkspaceParentPath={newWorkspaceParentPath}
        setNewWorkspaceParentPath={setNewWorkspaceParentPath}
        newWorkspacePath={newWorkspacePath}
        setNewWorkspacePath={setNewWorkspacePath}
        remoteHost={remoteHost}
        setRemoteHost={setRemoteHost}
        remoteUser={remoteUser}
        setRemoteUser={setRemoteUser}
        remotePath={remotePath}
        setRemotePath={setRemotePath}
        sidecarUrl={sidecarUrl}
        setSidecarUrl={setSidecarUrl}
        sidecarToken={sidecarToken}
        setSidecarToken={setSidecarToken}
        devcontainerRepoPath={devcontainerRepoPath}
        setDevcontainerRepoPath={setDevcontainerRepoPath}
        forDevcontainerPlanClear={() => setDevcontainerPlan(null)}
        devcontainerPlan={devcontainerPlan}
        devcontainerPlanLoading={devcontainerPlanLoading}
        onBrowseDirectory={(target) => void handleBrowseDirectory(target)}
        onLoadDevcontainerPlan={() => void handleLoadDevcontainerPlan()}
        onAddWorkspace={() => void handleAddWorkspace()}
        onCancel={resetAddWorkspaceForm}
      />

      {contextMenu && (
        <FileExplorerContextMenu
          contextMenu={contextMenu}
          onClose={closeContextMenu}
          hasScanAnalysis={hasScanAnalysis}
          hasSecondaryAnalysis={hasSecondaryAnalysis}
          hasHtmlBrowserWorkbench={hasHtmlBrowserWorkbench}
          hasCadWorkbench={hasCadWorkbench}
          hasScanSummary={hasScanSummary}
          contextMenuIsScanAnalysis={contextMenuIsScanAnalysis}
          contextMenuIsComparator={contextMenuIsComparator}
          contextMenuIsScanSummary={contextMenuIsScanSummary}
          onRevealInFinder={() => void handleRevealInFinder()}
          onOpenInTerminal={handleOpenInTerminal}
          onCreateFile={handleCreateFile}
          onCreateFolder={handleCreateFolder}
          onOpenFromMenu={() => void handleOpenFromMenu()}
          onCopyName={() => void handleCopyName()}
          onCopyPath={handleCopyPath}
          onCopyRelativePath={handleCopyRelativePath}
          onDuplicate={() => void handleDuplicate()}
          onRename={handleRename}
          onDelete={handleDelete}
          onOpenScanAnalysis={() => void handleOpenScanAnalysisFromMenu()}
          onRun12PlexQC={() => void handleRun12PlexQCFromMenu()}
          onAddToAnalysisBasket={handleAddToAnalysisBasket}
          onOpenSecondaryPanel={handleOpenSecondaryPanel}
          onOpenComparator={handleOpenComparatorFromMenu}
          onOpenHtml={() => void handleOpenHtmlFromMenu()}
          onOpenCad={() => void handleOpenCadFromMenu()}
          onOpenScanSummary={handleOpenScanSummaryFromMenu}
          onPreviewMarkdown={handlePreviewMarkdown}
        />
      )}

      {nameDialog && (
        <FileNameDialog
          title={
            nameDialog.mode === 'rename'
              ? 'Rename'
              : nameDialog.mode === 'create-file'
                ? 'New file'
                : 'New folder'
          }
          label={
            nameDialog.mode === 'rename'
              ? 'New name'
              : nameDialog.mode === 'create-file'
                ? 'File name'
                : 'Folder name'
          }
          initialValue={nameDialog.initial}
          confirmLabel={
            nameDialog.mode === 'rename'
              ? 'Rename'
              : nameDialog.mode === 'create-file'
                ? 'Create'
                : 'Create folder'
          }
          onConfirm={(value) => void handleNameDialogConfirm(value)}
          onCancel={() => setNameDialog(null)}
        />
      )}

      {pendingRemove && (
        <FileExplorerRemoveConfirm
          pendingRemove={pendingRemove}
          onCancel={() => setPendingRemove(null)}
          onConfirm={confirmRemoveWorkspace}
        />
      )}
    </div>
  );
}
