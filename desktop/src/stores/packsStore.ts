import { createWithEqualityFn as create } from 'zustand/traditional';
import { ChatAPI, type PackCatalogEntry, type PackStatus, type PacksAPIResponse, type InstallPackLoRAsResponse } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import type { PackCapability } from './packCapabilities';

export const PACK_LIFE_SCIENCES = 'life-sciences';
export const PACK_SOFTWARE_DEVELOPMENT = 'software-development';

interface PacksState {
  packs: PackStatus[];
  catalog: PackCatalogEntry[];
  layoutOwner: string;
  layoutProfile: 'team' | 'ide';
  capabilities: string[];
  loading: boolean;
  error: string | null;
  applyPacksResponse: (data: PacksAPIResponse) => void;
  fetchPacks: () => Promise<void>;
  fetchPackCatalog: () => Promise<void>;
  installPack: (packId: string) => Promise<void>;
  installPackLoRAs: (packId: string) => Promise<InstallPackLoRAsResponse>;
  uninstallPack: (packId: string) => Promise<void>;
  setPackEnabled: (packId: string, enabled: boolean) => Promise<void>;
  hasCapability: (cap: PackCapability | string) => boolean;
  /** @deprecated use hasCapability(PACK_CAP.SCAN_SUMMARY_VIEWER) */
  lifeSciencesEnabled: () => boolean;
  /** @deprecated use hasCapability(PACK_CAP.IDE_V2) */
  softwareDevelopmentEnabled: () => boolean;
}

function parsePacksResponse(data: PacksAPIResponse): Pick<PacksState, 'packs' | 'layoutOwner' | 'layoutProfile' | 'capabilities'> {
  const layoutProfile = data.layout_profile === 'ide' ? 'ide' : 'team';
  return {
    packs: data.packs ?? [],
    layoutOwner: data.layout_owner ?? '',
    layoutProfile,
    capabilities: data.capabilities ?? [],
  };
}

export const usePacksStore = create<PacksState>((set, get) => ({
  packs: [],
  catalog: [],
  layoutOwner: '',
  layoutProfile: 'team',
  capabilities: [],
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
      const catalog = await api.fetchPackCatalog();
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
    const c = String(cap).trim();
    return get().capabilities.includes(c);
  },

  lifeSciencesEnabled: () => {
    const pack = get().packs.find((p) => p.id === PACK_LIFE_SCIENCES);
    return pack?.installed === true && pack?.enabled === true;
  },

  softwareDevelopmentEnabled: () => get().hasCapability('ide-v2'),
}));
