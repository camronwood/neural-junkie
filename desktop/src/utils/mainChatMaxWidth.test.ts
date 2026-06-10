import { describe, expect, it } from 'vitest';
import {
  MAIN_CHAT_MIN_WIDTH,
  mainChatMaxWidth,
  reservedWidthForPanels,
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
