/** PackExtrasApi — extracted from ChatAPI for smaller modules. */
import type { HubFetchFn } from './packsApi';
import type { ResolvedCapability } from '../../types/protocol';
import type { ACEStepStatus, AIInterviewProgressResponse, ArenaSidecarStatus, CustomerPackContextResponse, ExpertPresetOption, ImageGenStatus, InstallACEStepResponse, InstallArenaSidecarResponse, PackStatus, PackValidationReport, PacksAPIResponse } from '../types/chatApiTypes';

export class PackExtrasApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async installPackFromZip(packZipBase64: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/install-zip`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_zip_base64: packZipBase64 }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async fetchACEStepStatus(packId = 'music-creation'): Promise<ACEStepStatus> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/acestep-status`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async installACEStep(
    packId = 'music-creation',
    modelVariant?: string,
  ): Promise<InstallACEStepResponse> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/install-acestep`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model_variant: modelVariant ?? 'sft' }),
      },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async restartMusicSidecar(packId = 'music-creation'): Promise<{ status: string; acestep: ACEStepStatus }> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/restart-sidecar`,
      { method: 'POST' },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchArenaSidecarStatus(packId = 'model-arena'): Promise<ArenaSidecarStatus> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/sidecar-status`,
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async installArenaSidecarDeps(
    packId = 'model-arena',
  ): Promise<InstallArenaSidecarResponse> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/install-sidecar-deps`,
      { method: 'POST' },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async restartArenaSidecar(
    packId = 'model-arena',
  ): Promise<{ status: string; sidecar: ArenaSidecarStatus }> {
    const response = await this.hubFetch(
      `/api/packs/${encodeURIComponent(packId)}/arena-restart-sidecar`,
      { method: 'POST' },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchAIInterviewProgress(): Promise<AIInterviewProgressResponse> {
    const response = await this.hubFetch('/api/ai-interview/progress');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async startAIInterviewDay(): Promise<AIInterviewProgressResponse> {
    const response = await this.hubFetch('/api/ai-interview/start', { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async completeAIInterviewDay(
    day: number,
    body: { concept?: boolean; drill?: boolean; complete?: boolean; advance?: boolean },
  ): Promise<AIInterviewProgressResponse> {
    const response = await this.hubFetch(`/api/ai-interview/days/${day}/complete`, {
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

  async submitAIInterviewGate(
    gateId: string,
    body: { eval_notes?: string; mock_notes?: string; score?: number },
  ): Promise<AIInterviewProgressResponse> {
    const response = await this.hubFetch(
      `/api/ai-interview/gates/${encodeURIComponent(gateId)}/submit`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      },
    );
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async unlockAIInterviewCert(): Promise<{
    certification: Record<string, unknown>;
    credential: Record<string, unknown>;
  }> {
    const response = await this.hubFetch('/api/ai-interview/cert/unlock', { method: 'POST' });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchImageGenStatus(): Promise<ImageGenStatus> {
    const response = await this.hubFetch('/api/image-gen/status');
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async uninstallPack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async setPackEnabled(packId: string, enabled: boolean): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/${encodeURIComponent(packId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async validatePack(body: {
    pack_zip_base64?: string;
    pack_dir?: string;
    pack_yaml?: string;
  }): Promise<PackValidationReport> {
    const response = await this.hubFetch(`/api/packs/validate`, {
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

  async devLinkPack(packDir: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/dev-link`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_dir: packDir }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async devReloadPack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/dev-reload`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_id: packId }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async devUnlinkPack(packId: string): Promise<PacksAPIResponse> {
    const response = await this.hubFetch(`/api/packs/dev-unlink`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pack_id: packId }),
    });
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return this.parsePacksMutationResponse(await response.json());
  }

  async fetchCustomerPackContext(): Promise<CustomerPackContextResponse> {
    const response = await this.hubFetch(`/api/packs/customer-context`);
    if (!response.ok) {
      const t = await response.text();
      throw new Error(t.trim() || response.statusText);
    }
    return response.json();
  }

  async fetchExpertPresets(): Promise<ExpertPresetOption[]> {
    const response = await this.hubFetch(`/api/expert-presets`);
    if (!response.ok) {
      throw new Error(`Failed to fetch expert presets: ${response.statusText}`);
    }
    return response.json();
  }

  parsePacksMutationResponse(data: Record<string, unknown>): PacksAPIResponse {
    return {
      packs: (data.packs as PackStatus[]) ?? [],
      pack_id: data.pack_id as string | undefined,
      layout_owner: data.layout_owner as string | undefined,
      layout_profile: data.layout_profile as string | undefined,
      capabilities: (data.capabilities as string[]) ?? [],
      capability_registry: (data.capability_registry as ResolvedCapability[]) ?? [],
      short_id_collisions: (data.short_id_collisions as string[]) ?? [],
    };
  }

}
