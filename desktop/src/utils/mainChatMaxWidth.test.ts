import { describe, expect, it, beforeEach, afterEach } from 'vitest';
import {
  MAIN_CHAT_MIN_WIDTH,
  mainChatMaxWidth,
  measureMainChatMaxWidth,
  reservedWidthForPanels,
  reservedWidthForRightPanels,
  type MainChatPanelVisibility,
} from './mainChatMaxWidth';

const none: MainChatPanelVisibility = {
  channelSidebarOpen: false,
  fileExplorerOpen: false,
  fileExplorerEmbedded: false,
  threadOpen: false,
  collaborationOpen: false,
  taskManagementOpen: false,
};

describe('reservedWidthForPanels', () => {
  it('returns zero when no panels are visible', () => {
    expect(reservedWidthForPanels(none)).toBe(0);
  });

  it('sums compact mins for visible panels', () => {
    expect(
      reservedWidthForPanels({
        channelSidebarOpen: true,
        fileExplorerOpen: true,
        fileExplorerEmbedded: true,
        threadOpen: true,
        collaborationOpen: false,
        taskManagementOpen: false,
      }),
    ).toBe(140 + 200 + 220);
  });

  it('uses overlay explorer min when not embedded', () => {
    expect(
      reservedWidthForPanels({
        ...none,
        fileExplorerOpen: true,
        fileExplorerEmbedded: false,
      }),
    ).toBe(160);
  });
});

describe('reservedWidthForRightPanels', () => {
  it('only reserves panels to the right of chat', () => {
    expect(
      reservedWidthForRightPanels({
        channelSidebarOpen: true,
        fileExplorerOpen: true,
        fileExplorerEmbedded: true,
        threadOpen: true,
        collaborationOpen: true,
        taskManagementOpen: false,
      }),
    ).toBe(220 + 240);
  });
});

describe('mainChatMaxWidth', () => {
  it('allows chat to use nearly full container when no chrome is visible', () => {
    expect(mainChatMaxWidth(2000, none)).toBe(2000);
  });

  it('does not apply a fixed 320px reservation cliff', () => {
    const withSidebar = mainChatMaxWidth(2000, { ...none, channelSidebarOpen: true });
    expect(withSidebar).toBe(2000 - 140);
    expect(withSidebar).toBeGreaterThan(2000 - 320);
  });

  it('clamps to min width on tiny containers', () => {
    expect(mainChatMaxWidth(300, { ...none, channelSidebarOpen: true })).toBe(MAIN_CHAT_MIN_WIDTH);
  });
});

describe('measureMainChatMaxWidth', () => {
  let container: HTMLDivElement;
  let sidebar: HTMLDivElement;
  let workspace: HTMLDivElement;
  let chat: HTMLDivElement;

  beforeEach(() => {
    container = document.createElement('div');
    sidebar = document.createElement('div');
    workspace = document.createElement('div');
    chat = document.createElement('div');

    Object.defineProperty(container, 'clientWidth', { value: 1600, configurable: true });
    Object.defineProperty(sidebar, 'offsetWidth', { value: 220, configurable: true });
    Object.defineProperty(workspace, 'offsetWidth', { value: 400, configurable: true });
    Object.defineProperty(chat, 'offsetWidth', { value: 420, configurable: true });

    workspace.style.flexGrow = '1';
    container.append(sidebar, workspace, chat);
    document.body.appendChild(container);
  });

  afterEach(() => {
    container.remove();
  });

  it('uses measured left panel widths and skips flex-grow workspace', () => {
    expect(measureMainChatMaxWidth(container, chat, none)).toBe(1600 - 220);
  });

  it('reserves compact mins for right-side panels', () => {
    expect(
      measureMainChatMaxWidth(container, chat, {
        ...none,
        threadOpen: true,
      }),
    ).toBe(1600 - 220 - 220);
  });
});
