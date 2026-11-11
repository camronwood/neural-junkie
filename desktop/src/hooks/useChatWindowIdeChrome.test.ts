import { describe, expect, it, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useChatWindowIdeChrome } from './useChatWindowIdeChrome';

vi.mock('../stores/ideOverlayStore', () => ({
  useIdeOverlayStore: (sel: (s: Record<string, unknown>) => unknown) =>
    sel({
      quickOpenOpen: false,
      symbolModalOpen: false,
      fastEditOpen: false,
      problemsOpen: false,
      gitModalOpen: false,
      setQuickOpenOpen: vi.fn(),
      setSymbolModalOpen: vi.fn(),
      setFastEditOpen: vi.fn(),
      setProblemsOpen: vi.fn(),
      setGitModalOpen: vi.fn(),
    }),
}));

vi.mock('../stores/packsStore', () => ({
  usePacksStore: (sel: (s: Record<string, unknown>) => unknown) =>
    sel({
      layoutProfile: 'team',
      hasCapability: () => false,
      idePackActive: () => false,
      packs: [],
      getToolbarActions: () => [],
    }),
}));

vi.mock('../stores/settingsStore', () => ({
  useSettingsStore: (sel: (s: Record<string, unknown>) => unknown) =>
    sel({
      layoutSettings: { filesPanelVisible: true, chatPanelVisible: true },
    }),
}));

describe('useChatWindowIdeChrome', () => {
  it('exposes panel toggles and pack modal helpers', () => {
    const { result } = renderHook(() => useChatWindowIdeChrome());
    expect(result.current.fileExplorerOpen).toBe(false);
    expect(result.current.ideLayoutAvailable).toBe(false);
    act(() => {
      result.current.setActivePackModal('model-arena');
    });
    expect(result.current.modelArenaModalOpen).toBe(true);
    expect(result.current.phoenixModalOpen).toBe(false);
  });
});
