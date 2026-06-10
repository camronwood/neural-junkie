export const MAIN_CHAT_MIN_WIDTH = 260;

export interface MainChatPanelVisibility {
  channelSidebarOpen: boolean;
  fileExplorerOpen: boolean;
  fileExplorerEmbedded: boolean;
  threadOpen: boolean;
  collaborationOpen: boolean;
  taskManagementOpen: boolean;
}

export const PANEL_COMPACT_MINS = {
  channelSidebar: 140,
  fileExplorerEmbedded: 200,
  fileExplorerOverlay: 160,
  thread: 220,
  collaboration: 240,
  taskManagement: 260,
} as const;

/** Sum compact minimum widths for visible chrome that should not be crushed by chat resize. */
export function reservedWidthForPanels(visibility: MainChatPanelVisibility): number {
  let reserved = 0;
  if (visibility.channelSidebarOpen) {
    reserved += PANEL_COMPACT_MINS.channelSidebar;
  }
  if (visibility.fileExplorerOpen) {
    reserved += visibility.fileExplorerEmbedded
      ? PANEL_COMPACT_MINS.fileExplorerEmbedded
      : PANEL_COMPACT_MINS.fileExplorerOverlay;
  }
  if (visibility.threadOpen) {
    reserved += PANEL_COMPACT_MINS.thread;
  }
  if (visibility.collaborationOpen) {
    reserved += PANEL_COMPACT_MINS.collaboration;
  }
  if (visibility.taskManagementOpen) {
    reserved += PANEL_COMPACT_MINS.taskManagement;
  }
  return reserved;
}

/** Max draggable width for the main chat panel; workspace/editor flex region is not reserved. */
export function mainChatMaxWidth(
  containerWidth: number,
  visibility: MainChatPanelVisibility,
  minWidth: number = MAIN_CHAT_MIN_WIDTH,
): number {
  if (!Number.isFinite(containerWidth) || containerWidth <= 0) {
    return minWidth;
  }
  const reserved = reservedWidthForPanels(visibility);
  return Math.max(minWidth, containerWidth - reserved);
}
