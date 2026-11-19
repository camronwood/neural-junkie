/** UserToolsApi — MCP Tool Wizard HTTP tools + grants. */
import type { HubFetchFn } from './packsApi';

export type UserMCPTool = {
  id: string;
  name: string;
  description?: string;
  url: string;
  method?: string;
  headers?: Record<string, string>;
  json_path?: string;
  granted_agents?: string[];
  created_at?: string;
};

export type UserMCPServer = {
  id: string;
  name: string;
  url?: string;
  transport_type?: string;
  created_at?: string;
};

export class UserToolsApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async listUserTools(): Promise<UserMCPTool[]> {
    const response = await this.hubFetch('/api/mcp/user-tools');
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    const data = await response.json();
    return Array.isArray(data) ? data : [];
  }

  async createUserTool(tool: Omit<UserMCPTool, 'id' | 'created_at'>): Promise<UserMCPTool> {
    const response = await this.hubFetch('/api/mcp/user-tools', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(tool),
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async deleteUserTool(id: string): Promise<void> {
    const response = await this.hubFetch(`/api/mcp/user-tools/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
    if (!response.ok && response.status !== 204) {
      throw new Error(await response.text() || response.statusText);
    }
  }

  async testUserTool(id: string, query?: string): Promise<{ success: boolean; result?: string; error?: string }> {
    const response = await this.hubFetch(`/api/mcp/user-tools/${encodeURIComponent(id)}/test`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: query ?? '' }),
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async grantUserTool(id: string, agentName: string, grant: boolean): Promise<UserMCPTool> {
    const response = await this.hubFetch(`/api/mcp/user-tools/${encodeURIComponent(id)}/grant`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ agent_name: agentName, grant }),
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async listUserServers(): Promise<UserMCPServer[]> {
    const response = await this.hubFetch('/api/mcp/user-servers');
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    const data = await response.json();
    return Array.isArray(data) ? data : [];
  }

  async createUserServer(server: Omit<UserMCPServer, 'id' | 'created_at'>): Promise<UserMCPServer> {
    const response = await this.hubFetch('/api/mcp/user-servers', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(server),
    });
    if (!response.ok) throw new Error(await response.text() || response.statusText);
    return response.json();
  }

  async deleteUserServer(id: string): Promise<void> {
    const response = await this.hubFetch(`/api/mcp/user-servers/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
    if (!response.ok && response.status !== 204) {
      throw new Error(await response.text() || response.statusText);
    }
  }
}
