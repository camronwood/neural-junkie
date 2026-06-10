import { invoke } from '@tauri-apps/api/tauri';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { workspaceAbsolutePath } from './editorFileKind';
import { wellImageRelativePath } from './scanSummary';
import { ipcWorkspaceRoots } from './ipcWorkspaceRoots';

function isTauriShell(): boolean {
  return (
    typeof window !== 'undefined' &&
    Object.prototype.hasOwnProperty.call(window, '__TAURI__')
  );
}

function toDataUrl(mime: string, contentBase64: string): string {
  return `data:${mime};base64,${contentBase64}`;
}

/** Load a well TIFF as a PNG data URL for the scan summary viewer. */
export async function resolveScanSummaryWellImageSrc(options: {
  workspaceId: string;
  workspacePath: string;
  summaryDir: string;
  wellId: string;
}): Promise<string> {
  const { workspaceId, workspacePath, summaryDir, wellId } = options;
  const relativePath = wellImageRelativePath(summaryDir, wellId);
  const absolutePath = workspaceAbsolutePath(workspacePath, relativePath);

  if (isTauriShell()) {
    const result = await invoke<{ mime: string; content_base64: string }>('decode_scan_well_tiff', {
      absolutePath,
      ...ipcWorkspaceRoots(),
    });
    return toDataUrl(result.mime || 'image/png', result.content_base64);
  }

  const api = new ChatAPI(getHubBaseURL());
  return api.fetchScanSummaryWellImage(workspaceId, summaryDir, wellId);
}
