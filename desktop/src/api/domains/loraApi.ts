/** LoraApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';
import type { LoraExpertContext, LoraTrainDatasetPreview, LoraTrainJob, LoraTrainStartRequest, LoraTrainingBase } from '../types/chatApiTypes';

export class LoraApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async fetchLoraExpertContext(agentId: string): Promise<LoraExpertContext> {
    const response = await this.hubFetch(
      `/api/lora/train/expert-context?agent_id=${encodeURIComponent(agentId)}`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchLoraTrainBases(): Promise<LoraTrainingBase[]> {
    const response = await this.hubFetch('/api/lora/train/bases');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return Array.isArray(data.bases) ? data.bases : [];
  }

  async previewLoraTrain(params: {
    source: string;
    source_id: string;
    thread_id?: string;
    agent_name?: string;
    agent_id?: string;
    include_learnings?: boolean;
    incremental?: boolean;
  }): Promise<number> {
    const q = new URLSearchParams({
      source: params.source,
      source_id: params.source_id,
    });
    if (params.thread_id) q.set('thread_id', params.thread_id);
    if (params.agent_name) q.set('agent_name', params.agent_name);
    if (params.agent_id) q.set('agent_id', params.agent_id);
    if (params.include_learnings) q.set('include_learnings', '1');
    if (params.incremental) q.set('incremental', '1');
    const response = await this.hubFetch(`/api/lora/train/preview?${q.toString()}`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return Number(data.row_count ?? 0);
  }

  async previewLoraTrainDataset(
    body: Omit<LoraTrainStartRequest, 'base_ollama_tag' | 'ollama_tag' | 'hyperparams'> & {
      base_ollama_tag?: string;
      ollama_tag?: string;
    },
  ): Promise<LoraTrainDatasetPreview> {
    const response = await this.hubFetch('/api/lora/train/dataset-preview', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return {
      rows: Array.isArray(data.rows) ? data.rows : [],
      count: Number(data.count ?? 0),
      min_rows: Number(data.min_rows ?? 10),
    };
  }

  async bootstrapLoraTrainFromIndex(agentId: string): Promise<LoraTrainDatasetPreview> {
    const response = await this.hubFetch('/api/lora/train/index-bootstrap', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ agent_id: agentId }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    const data = await response.json();
    return {
      rows: Array.isArray(data.rows) ? data.rows : [],
      count: Number(data.count ?? 0),
      min_rows: 10,
    };
  }

  async startLoraTrain(body: LoraTrainStartRequest): Promise<LoraTrainJob> {
    const response = await this.hubFetch(`/api/lora/train`, {
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

  async fetchLoraTrainJob(jobId: string): Promise<LoraTrainJob> {
    const response = await this.hubFetch(`/api/lora/train/${encodeURIComponent(jobId)}`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async cancelLoraTrainJob(jobId: string): Promise<LoraTrainJob> {
    const response = await this.hubFetch(`/api/lora/train/${encodeURIComponent(jobId)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

}
