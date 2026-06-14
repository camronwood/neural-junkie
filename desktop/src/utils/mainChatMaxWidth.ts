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

/** Compact minimums for panels rendered to the right of the main chat column. */
export function reservedWidthForRightPanels(visibility: MainChatPanelVisibility): number {
  let reserved = 0;
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

function isShrinkableFlexRegion(el: HTMLElement): boolean {
  const style = window.getComputedStyle(el);
  return parseFloat(style.flexGrow) > 0;
}

/**
 * Max draggable chat width based on measured siblings in the flex row.
 * Skips flex-grow regions (workspace slot, team spacer) so chat can reclaim that space.
 */
export function measureMainChatMaxWidth(
  container: HTMLElement,
  chatEl: HTMLElement,
  visibility: MainChatPanelVisibility,
  minWidth: number = MAIN_CHAT_MIN_WIDTH,
): number {
  const containerWidth = container.clientWidth;
  if (!Number.isFinite(containerWidth) || containerWidth <= 0) {
    return minWidth;
  }

  let leftWidth = 0;
  for (const child of Array.from(container.children)) {
    if (child === chatEl) {
      break;
    }
    const el = child as HTMLElement;
    if (isShrinkableFlexRegion(el)) {
      continue;
    }
    leftWidth += el.offsetWidth;
  }

  const rightReserved = reservedWidthForRightPanels(visibility);
  return Math.max(minWidth, containerWidth - leftWidth - rightReserved);
}

/** Fallback max width when the chat element is not mounted yet. */
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
