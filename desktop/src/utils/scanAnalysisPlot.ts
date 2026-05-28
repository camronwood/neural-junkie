import { convertFileSrc } from '@tauri-apps/api/tauri';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { workspaceAbsolutePath } from './editorFileKind';
import { resolveEditorImageSrc } from './chatImageSrc';

function isTauriShell(): boolean {
  return (
    typeof window !== 'undefined' &&
    Object.prototype.hasOwnProperty.call(window, '__TAURI__')
  );
}

/** Resolve a plot JPG or process report path to a loadable image/src URL. */
export async function resolveScanAnalysisAssetSrc(options: {
  workspaceId: string;
  workspacePath: string;
  relativePath: string;
}): Promise<string> {
  const { workspaceId, workspacePath, relativePath } = options;
  const absolutePath = workspaceAbsolutePath(workspacePath, relativePath);
  return resolveEditorImageSrc({ workspaceId, relativePath, absolutePath });
}

/** Synchronous src for Tauri only (returns empty in browser until async fetch). */
export function scanAnalysisAssetSrcSync(workspacePath: string, relativePath: string): string {
  if (!isTauriShell()) return '';
  const absolutePath = workspaceAbsolutePath(workspacePath, relativePath);
  return convertFileSrc(absolutePath);
}

/** Fetch process report text from workspace. */
export async function fetchScanAnalysisProcessReport(options: {
  workspaceId: string;
  relativePath: string;
}): Promise<string> {
  const api = new ChatAPI(getHubBaseURL());
  return api.fetchFileContent(options.workspaceId, options.relativePath);
}
