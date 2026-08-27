import type {
  AssignSuggestion,
  Collaboration,
  CollaborationTask,
  ExecutionPolicy,
  GraphLayout,
  RunbookDefinition,
  RunbookDefinitionBundle,
  RunbookDefinitionSummary,
  RunbookRunProvenance,
  RunbookRunRecord,
} from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Runbook CRUD, definitions, templates, and pack-runbook HTTP surface. */
export class RunbooksApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async createRunbook(body: {
    description: string;
    agent_ids: string[];
    channel: string;
    created_by: string;
    tasks?: CollaborationTask[];
    execution_mode?: string;
    source_repo_path?: string;
  }): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    const response = await this.hubFetch(`/api/runbooks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async updateRunbook(
    collabId: string,
    body: {
      title?: string;
      description?: string;
      agent_ids?: string[];
      tasks?: CollaborationTask[];
      execution_policy?: ExecutionPolicy;
      graph_layout?: GraphLayout;
    }
  ): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async suggestRunbookAssignee(
    collabId: string,
    title: string,
    description: string
  ): Promise<AssignSuggestion | null> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}/suggest-assignee`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title, description }),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    const data = await response.json();
    if (data.agent_id) {
      return data as AssignSuggestion;
    }
    return null;
  }

  async parseRunbookPlan(collabId: string, markdown: string): Promise<CollaborationTask[]> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}/parse-plan`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ markdown }),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    const data = await response.json();
    return data.tasks ?? [];
  }

  async submitRunbook(collabId: string): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}/submit`,
      { method: 'POST' }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async startRunbook(collabId: string, inputs?: Record<string, string>): Promise<Collaboration> {
    const response = await this.hubFetch(`/api/runbooks/${encodeURIComponent(collabId)}/start`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(inputs ? { inputs } : {}),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async listRunbookDefinitions(): Promise<RunbookDefinitionSummary[]> {
    const response = await this.hubFetch(`/api/runbook-definitions`);
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    const data = await response.json();
    return Array.isArray(data) ? data : [];
  }

  async getRunbookDefinition(id: string, version?: number): Promise<RunbookDefinition> {
    const qs = version ? `?version=${version}` : '';
    const response = await this.hubFetch(`/api/runbook-definitions/${encodeURIComponent(id)}${qs}`);
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async saveRunbookDefinition(def: RunbookDefinition): Promise<RunbookDefinition> {
    const path = def.id
      ? `/api/runbook-definitions/${encodeURIComponent(def.id)}`
      : '/api/runbook-definitions';
    const response = await this.hubFetch(path, {
      method: def.id ? 'PUT' : 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(def),
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async exportRunbookDefinition(id: string, version?: number): Promise<RunbookDefinitionBundle> {
    const qs = version ? `?version=${version}` : '';
    const response = await this.hubFetch(`/api/runbook-definitions/${encodeURIComponent(id)}/export${qs}`);
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async importRunbookDefinition(
    bundleOrDefinition: RunbookDefinitionBundle | RunbookDefinition,
    options?: { keepId?: boolean }
  ): Promise<RunbookDefinition> {
    const qs = options?.keepId ? '?keep_id=true' : '';
    const response = await this.hubFetch(`/api/runbook-definitions/import${qs}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(bundleOrDefinition),
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async getRunbookRunProvenance(collaborationId: string): Promise<RunbookRunProvenance> {
    const response = await this.hubFetch(`/api/runbook-runs/${encodeURIComponent(collaborationId)}/provenance`);
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async instantiateRunbookDefinition(
    definitionId: string,
    body: { channel: string; created_by: string; agent_ids: string[]; inputs?: Record<string, string> }
  ): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    const response = await this.hubFetch(
      `/api/runbook-definitions/${encodeURIComponent(definitionId)}/instantiate`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async listRunbookRuns(definitionId?: string): Promise<RunbookRunRecord[]> {
    const qs = definitionId ? `?definition_id=${encodeURIComponent(definitionId)}` : '';
    const response = await this.hubFetch(`/api/runbook-runs${qs}`);
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async replayRunbookRun(collabId: string): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    const response = await this.hubFetch(`/api/runbook-runs/${encodeURIComponent(collabId)}/replay`, {
      method: 'POST',
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async listPackRunbooks(): Promise<{ pack_id: string; path: string; title: string }[]> {
    const response = await this.hubFetch('/api/packs/runbooks');
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    const data = await response.json();
    return data.runbooks ?? [];
  }

  async importPackRunbook(packId: string, path: string): Promise<RunbookDefinition> {
    const response = await this.hubFetch('/api/packs/runbooks/import', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_id: packId, path }),
    });
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async listRunbookTemplates(): Promise<RunbookDefinitionSummary[]> {
    const response = await this.hubFetch(`/api/runbook-templates`);
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }

  async createRunbookFromTemplate(
    templateName: string,
    body: { channel: string; created_by: string; agent_ids: string[] }
  ): Promise<{ collaboration_id: string; collaboration_channel: string; collaboration: Collaboration }> {
    const response = await this.hubFetch(`/api/runbook-templates/${encodeURIComponent(templateName)}/instantiate`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }
    );
    if (!response.ok) {
      throw new Error(await response.text() || response.statusText);
    }
    return response.json();
  }
}
