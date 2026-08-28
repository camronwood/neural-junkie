import type { GitChangeProposal } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Git change proposal approval HTTP surface. */
export class GitChangesApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchGitChanges(userId: string): Promise<GitChangeProposal[]> {
    const response = await this.hubFetch(
      `/api/git-changes?user_id=${encodeURIComponent(userId)}`
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch git changes: ${response.statusText}`);
    }
    return response.json();
  }

  async approveGitChange(changeId: string): Promise<GitChangeProposal> {
    const response = await this.hubFetch(
      `/api/git-changes/approve/${encodeURIComponent(changeId)}`,
      { method: 'POST' }
    );
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to approve git change: ${response.statusText}`);
    }
    return response.json();
  }

  async rejectGitChange(changeId: string, reason?: string): Promise<GitChangeProposal> {
    const response = await this.hubFetch(
      `/api/git-changes/reject/${encodeURIComponent(changeId)}`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: reason ?? '' }),
      }
    );
    if (!response.ok) {
      const detail = (await response.text()).trim();
      throw new Error(detail || `Failed to reject git change: ${response.statusText}`);
    }
    return response.json();
  }
}
