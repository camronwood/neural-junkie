export interface ChatWindowProps {
  onOpenSettings?: (tab?: import('../SettingsModal').SettingsTab) => void;
  onLogout?: () => void;
}
