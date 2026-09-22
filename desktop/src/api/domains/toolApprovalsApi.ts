/** ToolApprovalsApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class ToolApprovalsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchPendingToolApprovals(): Promise<
    Array<{
      id: string;
      agent_id: string;
      agent_name: string;
      session_id?: string;
      tool_name: string;
      tool_input?: Record<string, unknown>;
      channel: string;
      created_at: string;
    }>
  > {
    const response = await this.hubFetch('/api/tool-approvals/pending');
    if (!response.ok) {
      throw new Error(`Failed to list pending tool approvals: ${response.statusText}`);
    }
    return response.json();
  }

  async approveToolCall(approvalId: string, scope: 'once' | 'always' = 'once'): Promise<void> {
    const response = await this.hubFetch(`/api/tool-approvals/approve/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scope }),
    });

    if (!response.ok) {
      throw new Error(`Failed to approve tool call: ${response.statusText}`);
    }
  }

  async rejectToolCall(approvalId: string, reason: string = 'User rejected'): Promise<void> {
    const response = await this.hubFetch(`/api/tool-approvals/reject/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    });

    if (!response.ok) {
      throw new Error(`Failed to reject tool call: ${response.statusText}`);
    }
  }

}
