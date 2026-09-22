/** CadApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';
import type { CadParam } from '../types/chatApiTypes';

export class CadApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async renderCAD(body: {
    workspace: string;
    path: string;
    project_id?: string;
    params?: Record<string, string>;
    output_path?: string;
  }): Promise<{ content_base64: string; params?: unknown[]; scad_path?: string; stl_path?: string }> {
    const response = await this.hubFetch('/api/cad/render', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `CAD render failed: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchCADMesh(
    workspaceId: string,
    scadPath: string,
    projectId?: string
  ): Promise<{ content_base64: string }> {
    const params = new URLSearchParams({ workspace: workspaceId, path: scadPath });
    if (projectId) params.set('project_id', projectId);
    const response = await this.hubFetch(`/api/cad/mesh?${params.toString()}`);
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async fetchCADParams(
    workspaceId: string,
    scadPath: string,
    projectId?: string
  ): Promise<{ params: CadParam[] }> {
    const params = new URLSearchParams({ workspace: workspaceId, path: scadPath });
    if (projectId) params.set('project_id', projectId);
    const response = await this.hubFetch(`/api/cad/params?${params.toString()}`);
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async fetchCADVersions(projectId: string): Promise<{ versions: Array<{ id: string; label: string; created_at: string }> }> {
    const response = await this.hubFetch(`/api/cad/versions?project_id=${encodeURIComponent(projectId)}`);
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async saveCADVersion(body: {
    workspace: string;
    path: string;
    project_id: string;
    label: string;
    params?: Record<string, string>;
  }): Promise<unknown> {
    const response = await this.hubFetch('/api/cad/versions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async restoreCADVersion(
    projectId: string,
    versionId: string
  ): Promise<{ content?: string; scad_path?: string }> {
    const response = await this.hubFetch('/api/cad/versions/restore', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ project_id: projectId, version_id: versionId }),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async testOpenSCAD(path?: string): Promise<{ ok: boolean; message: string }> {
    const response = await this.hubFetch('/api/cad/test-openscad', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: path ?? '' }),
    });
    const data = (await response.json()) as { ok: boolean; message: string };
    return data;
  }

  async checkCADPrintability(body: {
    stl_path: string;
    min_wall_mm?: number;
  }): Promise<{
    printable?: boolean;
    warnings?: string[];
    overhang?: { max_angle_deg?: number; faces_over_limit?: number };
    estimated_min_wall_mm?: number;
  }> {
    const response = await this.hubFetch('/api/cad/printability', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

  async validateCADAssembly(body: {
    manifest_path: string;
    clearance_mm?: number;
  }): Promise<{ ok?: boolean; bom?: Array<{ part_id: string; name: string }>; fit_issues?: unknown[] }> {
    const response = await this.hubFetch('/api/cad/assembly/validate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }

}
