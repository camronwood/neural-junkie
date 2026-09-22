/** PhoenixApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class PhoenixApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchPhoenixStatus(): Promise<{
    environment: string;
    credentials_path?: string;
    authenticated: boolean;
    logged_in: boolean;
    identity?: string;
    hint?: string;
  }> {
    const response = await this.hubFetch('/api/phoenix/status');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchPhoenixAnalyses(): Promise<Array<{ id: string; label: string }>> {
    const response = await this.hubFetch('/api/phoenix/analyses');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = (await response.json()) as { analyses?: Array<{ id: string; label: string }> };
    return data.analyses ?? [];
  }

  async fetchPhoenixScanResults(): Promise<Array<{ id: string; label: string }>> {
    const response = await this.hubFetch('/api/phoenix/scan-results');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = (await response.json()) as { scan_results?: Array<{ id: string; label: string }> };
    return data.scan_results ?? [];
  }

  async phoenixLoginStart(): Promise<{
    session_id: string;
    user_code: string;
    verification_url: string;
    expires_in: number;
    environment: string;
  }> {
    const response = await this.hubFetch('/api/phoenix/login/start', { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async phoenixLoginPoll(sessionId: string): Promise<{
    status: string;
    identity?: string;
    hint?: string;
    expires_in?: number;
  }> {
    const response = await this.hubFetch(
      `/api/phoenix/login/poll?session_id=${encodeURIComponent(sessionId)}`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async phoenixLogout(): Promise<void> {
    const response = await this.hubFetch('/api/phoenix/logout', { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
  }

  async phoenixImport(body: {
    workspace_id: string;
    analysis_id: string;
    scan_results_id?: string;
    output_dir?: string;
  }): Promise<{
    analysis_dir: string;
    validation_dir?: string;
    scan_export_dir?: string;
    scan_results_id?: string;
    files_written?: string[];
    attachment_notes?: string[];
  }> {
    const response = await this.hubFetch('/api/phoenix/import', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async phoenixImportScan(body: {
    workspace_id: string;
    scan_results_id: string;
    output_dir?: string;
  }): Promise<{
    analysis_dir: string;
    scan_export_dir?: string;
    scan_results_id?: string;
    files_written?: string[];
  }> {
    const response = await this.hubFetch('/api/phoenix/import-scan', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

}
