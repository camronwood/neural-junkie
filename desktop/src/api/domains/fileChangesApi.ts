/** FileChangesApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';
import type { FileChange, FileChangeDiff, FileChangeRequest } from '../../types/protocol';

function userIdQuery(userId?: string): string {
  const trimmed = (userId ?? '').trim();
  return trimmed ? `?user_id=${encodeURIComponent(trimmed)}` : '';
}

export class FileChangesApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async proposeFileChangeFromMessage(params: {
    channel: string;
    messageId: string;
    workspaceId: string;
    targetPath?: string;
    userId?: string;
  }): Promise<FileChange> {
    const body: Record<string, string> = {
      channel: params.channel,
      message_id: params.messageId,
      workspace_id: params.workspaceId,
      target_path: params.targetPath || '',
    };
    if (params.userId?.trim()) {
      body.user_id = params.userId.trim();
    }
    const response = await this.hubFetch(`/api/file-changes/propose-from-message`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errText = await response.text();
      throw new Error(errText || `Failed to create proposal from message: ${response.statusText}`);
    }

    return response.json();
  }

  async listPendingFileChanges(userId?: string): Promise<FileChange[]> {
    const response = await this.hubFetch(`/api/file-changes${userIdQuery(userId)}`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch file changes: ${response.statusText}`);
    }
    
    return response.json();
  }

  async approveFileChange(
    changeId: string,
    userId?: string,
    newContent?: string
  ): Promise<FileChange> {
    const body =
      newContent !== undefined && newContent !== ''
        ? JSON.stringify({ new_content: newContent })
        : undefined;
    const response = await this.hubFetch(
      `/api/file-changes/approve/${changeId}${userIdQuery(userId)}`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body,
      }
    );

    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to approve file change: ${response.statusText}`);
    }

    return response.json();
  }

  async approveFileChangeRequest(requestId: string, userId?: string): Promise<FileChangeRequest> {
    const response = await this.hubFetch(
      `/api/file-changes/requests/${encodeURIComponent(requestId)}/approve${userIdQuery(userId)}`,
      { method: 'POST' },
    );
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to approve file change request: ${response.statusText}`);
    }
    return response.json();
  }

  async rejectFileChangeRequest(
    requestId: string,
    reason: string = 'No reason provided',
    userId?: string,
  ): Promise<FileChangeRequest> {
    const body: Record<string, string> = { reason };
    if (userId?.trim()) {
      body.user_id = userId.trim();
    }
    const response = await this.hubFetch(
      `/api/file-changes/requests/${encodeURIComponent(requestId)}/reject`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      },
    );
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to reject file change request: ${response.statusText}`);
    }
    return response.json();
  }

  async updateFileChangeContent(changeId: string, newContent: string): Promise<FileChangeDiff> {
    const response = await this.hubFetch(`/api/file-changes/${changeId}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ new_content: newContent }),
    });

    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to update file change: ${response.statusText}`);
    }

    return response.json();
  }

  async rejectFileChange(changeId: string, reason: string = 'No reason provided', userId?: string): Promise<FileChange> {
    const body: Record<string, string> = { reason };
    if (userId?.trim()) {
      body.user_id = userId.trim();
    }
    const response = await this.hubFetch(`/api/file-changes/reject/${changeId}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to reject file change: ${response.statusText}`);
    }

    return response.json();
  }

  async getFileDiff(changeId: string): Promise<FileChangeDiff> {
    const response = await this.hubFetch(`/api/file-changes/${changeId}`);
    
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to get file diff: ${response.statusText}`);
    }
    
    return response.json();
  }

}
