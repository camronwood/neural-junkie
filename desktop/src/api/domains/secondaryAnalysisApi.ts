/** SecondaryAnalysisApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';

export class SecondaryAnalysisApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async run12PlexQC(body: {
    workspace_id: string;
    analysis_dir: string;
    write_report?: boolean;
  }): Promise<import('../../utils/secondaryAnalysis').PanelQCReport> {
    const response = await this.hubFetch('/api/secondary-analysis/12plex-qc', {
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

  async runSecondaryAnalysis(body: {
    workflow: string;
    workspace_id: string;
    config?: Record<string, unknown>;
  }): Promise<import('../../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    const response = await this.hubFetch('/api/secondary-analysis/run', {
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

  async fetchSecondaryAnalysisJob(
    jobId: string
  ): Promise<import('../../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    const response = await this.hubFetch(
      `/api/secondary-analysis/jobs/${encodeURIComponent(jobId)}`
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async cancelSecondaryAnalysisJob(
    jobId: string
  ): Promise<import('../../utils/secondaryAnalysis').SecondaryAnalysisJob> {
    const response = await this.hubFetch(
      `/api/secondary-analysis/jobs/${encodeURIComponent(jobId)}`,
      { method: 'DELETE' }
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchComparatorSummary(
    workspaceId: string,
    dir: string
  ): Promise<import('../../utils/secondaryAnalysis').ComparatorSummary> {
    const params = new URLSearchParams({ workspace: workspaceId, dir });
    const response = await this.hubFetch(
      `/api/secondary-analysis/comparator-summary?${params.toString()}`
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

}
