import { useEffect, useCallback } from 'react';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';

export function useEditorShortcuts() {
  const {
    saveTab,
    saveAllTabs,
    closeTab,
    getActiveTab,
    hasUnsavedChanges,
  } = useEditorStore();
  
  const { setFileExplorerOpen } = useFileExplorerStore();

  const handleKeyDown = useCallback(async (event: KeyboardEvent) => {
    const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
    const cmdKey = isMac ? event.metaKey : event.ctrlKey;
    
    // Don't trigger shortcuts if user is typing in an input/textarea
    const target = event.target as HTMLElement;
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.contentEditable === 'true') {
      return;
    }

    // Cmd+S / Ctrl+S - Save active file
    if (cmdKey && event.key === 's' && !event.shiftKey) {
      event.preventDefault();
      const activeTab = getActiveTab();
      if (activeTab) {
        await saveTab(activeTab.id);
      }
      return;
    }

    // Cmd+Shift+S / Ctrl+Shift+S - Save all files
    if (cmdKey && event.key === 'S' && event.shiftKey) {
      event.preventDefault();
      await saveAllTabs();
      return;
    }

    // Cmd+W / Ctrl+W - Close active tab
    if (cmdKey && event.key === 'w' && !event.shiftKey) {
      event.preventDefault();
      const activeTab = getActiveTab();
      if (activeTab) {
        closeTab(activeTab.id);
      }
      return;
    }

    // Cmd+P / Ctrl+P - Quick file search (placeholder)
    if (cmdKey && event.key === 'p' && !event.shiftKey) {
      event.preventDefault();
      // TODO: Implement quick file search modal
      console.log('Quick file search - not implemented yet');
      return;
    }

    // Cmd+Shift+P / Ctrl+Shift+P - Command palette (placeholder)
    if (cmdKey && event.key === 'P' && event.shiftKey) {
      event.preventDefault();
      // TODO: Implement command palette
      console.log('Command palette - not implemented yet');
      return;
    }

    // Cmd+B / Ctrl+B - Toggle file explorer
    if (cmdKey && event.key === 'b' && !event.shiftKey) {
      event.preventDefault();
      setFileExplorerOpen(true);
      return;
    }

    // Cmd+F / Ctrl+F - Find in file (Monaco handles this)
    if (cmdKey && event.key === 'f' && !event.shiftKey) {
      // Let Monaco handle this
      return;
    }

    // Cmd+Tab / Ctrl+Tab - Switch between tabs (placeholder)
    if (cmdKey && event.key === 'Tab' && !event.shiftKey) {
      event.preventDefault();
      // TODO: Implement tab switching
      console.log('Switch tabs - not implemented yet');
      return;
    }

    // Escape - Close any open modals/panels
    if (event.key === 'Escape') {
      // TODO: Implement escape to close modals
      console.log('Escape pressed - not implemented yet');
      return;
    }
  }, [saveTab, saveAllTabs, closeTab, getActiveTab, setFileExplorerOpen]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleKeyDown]);

  // Warn user about unsaved changes when trying to leave
  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (hasUnsavedChanges()) {
        event.preventDefault();
        event.returnValue = 'You have unsaved changes. Are you sure you want to leave?';
        return 'You have unsaved changes. Are you sure you want to leave?';
      }
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload);
    };
  }, [hasUnsavedChanges]);
}
