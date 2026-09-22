export interface FileExplorerPanelProps {
  onClose: () => void;
  onFileOpen?: () => void;
  variant?: 'overlay' | 'embedded';
}
