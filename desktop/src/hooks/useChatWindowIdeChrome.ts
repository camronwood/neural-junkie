import { useState } from 'react';
import { shallow } from 'zustand/shallow';
import { useIdeOverlayStore } from '../stores/ideOverlayStore';
import { usePacksStore } from '../stores/packsStore';
import { useSettingsStore } from '../stores/settingsStore';
import { isIdeLayout } from '../utils/layoutPresets';

/**
 * IDE panel open state, overlay modals, and pack-gated layout flags for ChatWindow.
 * Extraction loop 2 from ChatWindowRoot.
 */
export function useChatWindowIdeChrome() {
  const layoutSettings = useSettingsStore((s) => s.layoutSettings);
  const [fileExplorerOpen, setFileExplorerOpen] = useState(false);
  const [codeEditorOpen, setCodeEditorOpen] = useState(false);
  const {
    quickOpenOpen,
    symbolModalOpen,
    fastEditOpen,
    problemsOpen,
    gitModalOpen,
    setQuickOpenOpen,
    setSymbolModalOpen,
    setFastEditOpen,
    setProblemsOpen,
    setGitModalOpen,
  } = useIdeOverlayStore(
    (s) => ({
      quickOpenOpen: s.quickOpenOpen,
      symbolModalOpen: s.symbolModalOpen,
      fastEditOpen: s.fastEditOpen,
      problemsOpen: s.problemsOpen,
      gitModalOpen: s.gitModalOpen,
      setQuickOpenOpen: s.setQuickOpenOpen,
      setSymbolModalOpen: s.setSymbolModalOpen,
      setFastEditOpen: s.setFastEditOpen,
      setProblemsOpen: s.setProblemsOpen,
      setGitModalOpen: s.setGitModalOpen,
    }),
    shallow
  );
  const [activePackModal, setActivePackModal] = useState<string | null>(null);
  const phoenixModalOpen = activePackModal === 'phoenix-import';
  const roomChatModalOpen = activePackModal === 'room-chat';
  const modelArenaModalOpen = activePackModal === 'model-arena';
  const aiInterviewModalOpen = activePackModal === 'ai-interview-prep';
  const layoutProfile = usePacksStore((s) => s.layoutProfile);
  const hasIdeV2 = usePacksStore((s) => s.hasCapability('ide-v2'));
  const idePackActive = usePacksStore((s) => s.idePackActive());
  const hasIdeComposer = usePacksStore((s) => s.hasCapability('ide-v3-composer'));
  const ideLayout = layoutProfile === 'ide' && isIdeLayout(layoutSettings);
  const ideEnabled = hasIdeV2 || idePackActive;
  const ideLayoutAvailable = ideEnabled;
  const enabledPackCount = usePacksStore((s) => s.packs.filter((p) => p.enabled).length);
  const customPackToolbarActions = usePacksStore((s) => s.getToolbarActions());

  return {
    fileExplorerOpen,
    setFileExplorerOpen,
    codeEditorOpen,
    setCodeEditorOpen,
    quickOpenOpen,
    symbolModalOpen,
    fastEditOpen,
    problemsOpen,
    gitModalOpen,
    setQuickOpenOpen,
    setSymbolModalOpen,
    setFastEditOpen,
    setProblemsOpen,
    setGitModalOpen,
    activePackModal,
    setActivePackModal,
    phoenixModalOpen,
    roomChatModalOpen,
    modelArenaModalOpen,
    aiInterviewModalOpen,
    layoutProfile,
    hasIdeV2,
    idePackActive,
    hasIdeComposer,
    ideLayout,
    ideEnabled,
    ideLayoutAvailable,
    enabledPackCount,
    customPackToolbarActions,
  };
}
