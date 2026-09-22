import type { FileNode } from '../../stores/fileExplorerStore';
import { fileExplorerIcon } from './fileExplorerUtils';

export interface FileExplorerTreeProps {
  files: FileNode[];
  level?: number;
  expandedPaths: Record<string, boolean>;
  isPathSelected: (path: string) => boolean;
  onFileClick: (file: FileNode, e?: React.MouseEvent) => void;
  onContextMenu: (e: React.MouseEvent, file: FileNode) => void;
  onDragStart: (e: React.DragEvent, file: FileNode) => void;
  onDragEnd: (e: React.DragEvent) => void;
}

export function FileExplorerTree({
  files,
  level = 0,
  expandedPaths,
  isPathSelected,
  onFileClick,
  onContextMenu,
  onDragStart,
  onDragEnd,
}: FileExplorerTreeProps) {
  return (
    <>
      {files.map((file) => (
        <div key={file.path}>
          <div
            className={`flex items-center gap-2 py-1 px-2 cursor-pointer hover:bg-slack-bgHover rounded ${
              isPathSelected(file.path) ? 'bg-slack-accent text-white' : 'text-slack-text'
            } ${!file.is_dir ? 'cursor-grab active:cursor-grabbing' : ''}`}
            style={{ paddingLeft: `${level * 16 + 8}px` }}
            draggable={!file.is_dir}
            onDragStart={(e) => onDragStart(e, file)}
            onDragEnd={onDragEnd}
            onClick={(e) => void onFileClick(file, e)}
            onContextMenu={(e) => onContextMenu(e, file)}
            title={
              file.is_dir
                ? undefined
                : 'Click to preview · Cmd/Ctrl+click to multi-select · drag selection to chat'
            }
          >
            <span className="text-sm">{fileExplorerIcon(file, expandedPaths)}</span>
            <span className="text-sm truncate flex-1">{file.name}</span>
            {file.is_dir && (
              <span className="text-xs text-slack-textMuted">
                {expandedPaths[file.path] ? '▼' : '▶'}
              </span>
            )}
          </div>
          {file.is_dir && expandedPaths[file.path] && file.children && (
            <div>
              <FileExplorerTree
                files={file.children}
                level={level + 1}
                expandedPaths={expandedPaths}
                isPathSelected={isPathSelected}
                onFileClick={onFileClick}
                onContextMenu={onContextMenu}
                onDragStart={onDragStart}
                onDragEnd={onDragEnd}
              />
            </div>
          )}
        </div>
      ))}
    </>
  );
}
