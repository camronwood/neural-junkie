/** FilesApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class FilesApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchFiles(workspaceId: string, path: string = '/'): Promise<any[]> {
    const response = await this.hubFetch(`/api/files?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`
    );
    
    if (!response.ok) {
      throw new Error(`Failed to fetch files: ${response.statusText}`);
    }
    
    return response.json();
  }

  async fetchFileContent(workspaceId: string, path: string): Promise<string> {
    const response = await this.hubFetch(`/api/file-content?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`
    );
    
    if (!response.ok) {
      const body = await response.text().catch(() => '');
      const detail = body.trim();
      if (response.status === 403) {
        throw new Error(
          detail || 'Forbidden: path is outside the workspace. Use a path relative to the workspace root.'
        );
      }
      if (response.status === 404) {
        throw new Error(detail || `Not Found: ${path}`);
      }
      throw new Error(
        detail
          ? `Failed to fetch file content (${response.status}): ${detail}`
          : `Failed to fetch file content: ${response.statusText}`
      );
    }
    
    const data = await response.json();
    return data.content;
  }

  async fetchScanSummaryWellImage(
    workspaceId: string,
    summaryDir: string,
    well: string
  ): Promise<string> {
    const params = new URLSearchParams({
      workspace: workspaceId,
      dir: summaryDir,
      well,
    });
    const response = await this.hubFetch(`/api/scan-summary/well-image?${params.toString()}`
    );
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Failed to load well image: ${response.statusText}`);
    }
    const data = (await response.json()) as { mime?: string; content_base64?: string };
    const b64 = data.content_base64 ?? '';
    if (!b64) {
      throw new Error('Empty well image payload from hub');
    }
    const mime = data.mime || 'image/png';
    return `data:${mime};base64,${b64}`;
  }

  async fetchWorkspaceBinaryDataUrl(
    workspaceId: string,
    path: string,
    fallbackMime = 'application/octet-stream',
  ): Promise<string> {
    const response = await this.hubFetch(
      `/api/file-content?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}&binary=1`,
    );
    if (!response.ok) {
      throw new Error(`Failed to load file: ${response.statusText}`);
    }
    const data = (await response.json()) as { mime?: string; content_base64?: string };
    const b64 = data.content_base64 ?? '';
    if (!b64) {
      throw new Error('Empty binary payload from hub');
    }
    const mime = data.mime || fallbackMime;
    return `data:${mime};base64,${b64}`;
  }

  async saveFileContent(workspaceId: string, path: string, content: string): Promise<void> {
    const response = await this.hubFetch(`/api/file-content`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
        path,
        content,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to save file content: ${response.statusText}`);
    }
  }

  async createFile(workspaceId: string, path: string, content: string = '', isDir = false): Promise<void> {
    const response = await this.hubFetch(`/api/file-create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
        path,
        content,
        is_dir: isDir,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to create file: ${response.statusText}`);
    }
  }

  async renameFile(workspaceId: string, oldPath: string, newPath: string): Promise<void> {
    const response = await this.hubFetch(`/api/file-rename`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workspace_id: workspaceId,
        old_path: oldPath,
        new_path: newPath,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to rename file: ${response.statusText}`);
    }
  }

  async deleteFile(workspaceId: string, path: string): Promise<void> {
    const response = await this.hubFetch(`/api/file-delete?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(path)}`,
      {
        method: 'DELETE',
      }
    );

    if (!response.ok) {
      throw new Error(`Failed to delete file: ${response.statusText}`);
    }
  }

}
