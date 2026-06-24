import { createWithEqualityFn as create } from 'zustand/traditional';
import {
  ChatAPI,
  type PackCatalogEntry,
  type PackStatus,
  type PacksAPIResponse,
  type InstallPackLoRAsResponse,
  type PackValidationReport,
  type CustomerPackContextResponse,
} from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { PackCapability } from './packCapabilities';
import {
  parseCapabilityRegistry,
  registryHasCapability,
  matchFileViewer,
  toolbarActionsFromRegistry,
  settingsKeysFromRegistry,
} from './packCapabilityRegistry';
import type { ResolvedCapability } from '../api/chatAPI';

export const PACK_LIFE_SCIENCES = 'life-sciences';
export const PACK_SOFTWARE_DEVELOPMENT = 'software-development';

interface PacksState {
  packs: PackStatus[];
  catalog: PackCatalogEntry[];
  layoutOwner: string;
  layoutProfile: 'team' | 'ide';
  capabilities: string[];
  capabilityRegistry: ResolvedCapability[];
  shortIdCollisions: string[];
  loading: boolean;
  error: string | null;
  applyPacksResponse: (data: PacksAPIResponse) => void;
  fetchPacks: () => Promise<void>;
  fetchPackCatalog: () => Promise<void>;
  installPack: (packId: string) => Promise<void>;
  installPackFromZip: (packZipBase64: string) => Promise<PacksAPIResponse>;
  installPackLoRAs: (packId: string) => Promise<InstallPackLoRAsResponse>;
  uninstallPack: (packId: string) => Promise<void>;
  setPackEnabled: (packId: string, enabled: boolean) => Promise<void>;
  setLayoutOwner: (packId: string) => Promise<void>;
  validatePack: (body: {
    pack_zip_base64?: string;
    pack_dir?: string;
    pack_yaml?: string;
  }) => Promise<PackValidationReport>;
  devLinkPack: (packDir: string) => Promise<PacksAPIResponse>;
  devReloadPack: (packId: string) => Promise<PacksAPIResponse>;
  devUnlinkPack: (packId: string) => Promise<PacksAPIResponse>;
  fetchCustomerPackContext: () => Promise<CustomerPackContextResponse>;
  hasCapability: (cap: PackCapability | string) => boolean;
  getFileViewerForPath: (path: string) => ResolvedCapability | undefined;
  getToolbarActions: () => ReturnType<typeof toolbarActionsFromRegistry>;
  getPackSettingsKeys: () => string[];
  /** True when software-development is installed and enabled (pack row, not capability union). */
  softwareDevelopmentPackActive: () => boolean;
  /** @deprecated use hasCapability(PACK_CAP.SCAN_SUMMARY_VIEWER) */
  lifeSciencesEnabled: () => boolean;
  /** @deprecated use hasCapability(PACK_CAP.IDE_V2) */
  softwareDevelopmentEnabled: () => boolean;
}

function parsePacksResponse(data: PacksAPIResponse): Pick<
  PacksState,
  'packs' | 'layoutOwner' | 'layoutProfile' | 'capabilities' | 'capabilityRegistry' | 'shortIdCollisions'
> {
  const layoutProfile = data.layout_profile === 'ide' ? 'ide' : 'team';
  const reg = parseCapabilityRegistry(data);
  return {
    packs: data.packs ?? [],
    layoutOwner: data.layout_owner ?? '',
    layoutProfile,
    capabilities: reg.capabilities,
    capabilityRegistry: reg.capabilityRegistry,
    shortIdCollisions: data.short_id_collisions ?? [],
  };
}

export const usePacksStore = create<PacksState>((set, get) => ({
  packs: [],
  catalog: [],
  layoutOwner: '',
  layoutProfile: 'team',
  capabilities: [],
  capabilityRegistry: [],
  shortIdCollisions: [],
  loading: false,
  error: null,

  applyPacksResponse: (data) => {
    set(parsePacksResponse(data));
  },

  fetchPacks: async () => {
    set({ loading: true, error: null });
    try {
      const api = new ChatAPI(getHubBaseURL());
      const data = await api.fetchPacks();
      set({ ...parsePacksResponse(data), loading: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load domain packs';
      set({ error: message, loading: false });
    }
  },

  fetchPackCatalog: async () => {
    set({ loading: true, error: null });
    try {
      const api = new ChatAPI(getHubBaseURL());
      let catalog: PackCatalogEntry[];
      try {
        catalog = await api.refreshPackCatalog();
      } catch {
        catalog = await api.fetchPackCatalog();
      }
      set({ catalog, loading: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load pack catalog';
      set({ error: message, loading: false });
    }
  },

  installPack: async (packId) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.installPack(packId);
    get().applyPacksResponse(data);
    await get().fetchPackCatalog();
  },

  installPackFromZip: async (packZipBase64) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.installPackFromZip(packZipBase64);
    get().applyPacksResponse(data);
    await get().fetchPackCatalog();
    return data;
  },

  installPackLoRAs: async (packId) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.installPackLoRAs(packId);
    await get().fetchPackCatalog();
    return data;
  },

  uninstallPack: async (packId) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.uninstallPack(packId);
    get().applyPacksResponse(data);
    await get().fetchPackCatalog();
  },

  validatePack: async (body) => {
    const api = new ChatAPI(getHubBaseURL());
    return api.validatePack(body);
  },

  devLinkPack: async (packDir) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.devLinkPack(packDir);
    get().applyPacksResponse(data);
    await get().fetchPackCatalog();
    return data;
  },

  devReloadPack: async (packId) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.devReloadPack(packId);
    get().applyPacksResponse(data);
    await get().fetchPackCatalog();
    return data;
  },

  devUnlinkPack: async (packId) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.devUnlinkPack(packId);
    get().applyPacksResponse(data);
    await get().fetchPackCatalog();
    return data;
  },

  fetchCustomerPackContext: async () => {
    const api = new ChatAPI(getHubBaseURL());
    return api.fetchCustomerPackContext();
  },

  setLayoutOwner: async (packId) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.setLayoutOwner(packId);
    get().applyPacksResponse(data);
    const pack = get().packs.find((p) => p.id === packId);
    if (pack?.layout_profile === 'ide') {
      const { layoutSettings, updateLayoutSettings } = await import('./settingsStore').then(
        (m) => m.useSettingsStore.getState(),
      );
      if (!layoutSettings.devPackLayoutNudgeApplied) {
        const { panelsForPreset } = await import('../utils/layoutPresets');
        await updateLayoutSettings({
          ...panelsForPreset('ide'),
          devPackLayoutNudgeApplied: true,
        });
      }
    }
  },

  setPackEnabled: async (packId, enabled) => {
    const api = new ChatAPI(getHubBaseURL());
    const data = await api.setPackEnabled(packId, enabled);
    get().applyPacksResponse(data);
    await get().fetchPackCatalog();
    if (enabled) {
      const { layoutSettings, updateLayoutSettings, settings, updateSettings } = await import('./settingsStore').then(
        (m) => m.useSettingsStore.getState(),
      );
      if (!layoutSettings.sidebarAgentsVisible) {
        await updateLayoutSettings({ sidebarAgentsVisible: true });
      }
      if (get().layoutOwner === packId && packId === PACK_SOFTWARE_DEVELOPMENT && !layoutSettings.devPackLayoutNudgeApplied) {
        const { panelsForPreset } = await import('../utils/layoutPresets');
        await updateLayoutSettings({
          ...panelsForPreset('ide'),
          devPackLayoutNudgeApplied: true,
        });
      }
      try {
        const api = new ChatAPI(getHubBaseURL());
        const [agentList, channelList] = await Promise.all([api.fetchAgents(), api.fetchChannels()]);
        const { patchRevealActiveAgentsInSidebar } = await import('../utils/sidebarVisibility');
        const { useChatStore } = await import('./chatStore');
        const revealPatch = patchRevealActiveAgentsInSidebar(settings, agentList, channelList);
        if (revealPatch) {
          await updateSettings(revealPatch);
        }
        useChatStore.setState({ agents: agentList, channels: channelList });
      } catch {
        // non-fatal
      }
    }
  },

  hasCapability: (cap) => {
    const state = get();
    return registryHasCapability(state.capabilities, state.capabilityRegistry, cap);
  },

  getFileViewerForPath: (path) => matchFileViewer(get().capabilityRegistry, path),

  getToolbarActions: () => toolbarActionsFromRegistry(get().capabilityRegistry),

  getPackSettingsKeys: () => settingsKeysFromRegistry(get().capabilityRegistry),

  softwareDevelopmentPackActive: () => {
    const pack = get().packs.find((p) => p.id === PACK_SOFTWARE_DEVELOPMENT);
    return pack?.installed === true && pack?.enabled === true;
  },

  lifeSciencesEnabled: () => {
    const pack = get().packs.find((p) => p.id === PACK_LIFE_SCIENCES);
    return pack?.installed === true && pack?.enabled === true;
  },

  softwareDevelopmentEnabled: () =>
    get().hasCapability('ide-v2') || get().softwareDevelopmentPackActive(),
}));
