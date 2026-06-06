import { beforeEach, describe, expect, it, vi } from 'vitest';
import { dispatchShortcut } from './dispatcher';
import { useShortcutHandlersStore } from '../stores/shortcutHandlersStore';

describe('dispatchShortcut', () => {
  beforeEach(() => {
    useShortcutHandlersStore.setState({
      gates: {
        devPackEnabled: true,
        ideLayout: true,
        codeEditorOpen: true,
        showAgentStop: false,
        hasPendingApprovals: false,
        threadOpen: false,
        terminalFocused: false,
        monacoFocused: false,
        chatConnected: true,
      },
    });
  });

  it('dispatches higher-priority terminal clear over fast edit for mod+k', async () => {
    const clearTerminal = vi.fn();
    const fastEdit = vi.fn();
    useShortcutHandlersStore.getState().registerHandlers({ clearTerminal, fastEdit });
    useShortcutHandlersStore.getState().setGates({
      terminalFocused: true,
      codeEditorOpen: true,
      devPackEnabled: true,
    });

    vi.stubGlobal('navigator', { platform: 'MacIntel' });
    const event = {
      key: 'k',
      metaKey: true,
      ctrlKey: false,
      shiftKey: false,
      altKey: false,
      target: document.body,
      preventDefault: vi.fn(),
      stopPropagation: vi.fn(),
    } as unknown as KeyboardEvent;

    const handled = await dispatchShortcut(event);
    expect(handled).toBe(true);
    expect(clearTerminal).toHaveBeenCalledTimes(1);
    expect(fastEdit).not.toHaveBeenCalled();
    vi.unstubAllGlobals();
  });

  it('dispatches toggle channel sidebar for mod+b', async () => {
    const toggleChannelSidebar = vi.fn();
    useShortcutHandlersStore.getState().registerHandlers({ toggleChannelSidebar });

    vi.stubGlobal('navigator', { platform: 'Win32' });
    const event = {
      key: 'b',
      metaKey: false,
      ctrlKey: true,
      shiftKey: false,
      altKey: false,
      target: document.body,
      preventDefault: vi.fn(),
      stopPropagation: vi.fn(),
    } as unknown as KeyboardEvent;

    const handled = await dispatchShortcut(event);
    expect(handled).toBe(true);
    expect(toggleChannelSidebar).toHaveBeenCalledTimes(1);
    vi.unstubAllGlobals();
  });
});
