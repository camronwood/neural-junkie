import { createWithEqualityFn as create } from 'zustand/traditional';
import { ChatAPI } from '../api/chatAPI';
import { getCurrentDevicePosition, geolocationErrorMessage } from '../utils/geolocation';

export interface SharedDeviceLocation {
  lat: number;
  lon: number;
  accuracy_m: number;
  display_name: string;
  captured_at: string;
  age_s?: number;
  source?: string;
}

export interface PendingLocationRequest {
  id: string;
  agentName: string;
  channel: string;
  createdAt: string;
}

interface LocationShareState {
  sharing: boolean;
  snapshot: SharedDeviceLocation | null;
  error: string | null;
  busy: boolean;
  granted: boolean | null;
  pendingRequests: PendingLocationRequest[];
  startSharing: (api: ChatAPI) => Promise<boolean>;
  stopSharing: (api: ChatAPI) => Promise<void>;
  syncPending: (api: ChatAPI) => Promise<void>;
  fulfillPending: (api: ChatAPI, requestId: string) => Promise<void>;
  rejectPending: (api: ChatAPI, requestId: string, reason?: string) => Promise<void>;
  refreshGrant: (api: ChatAPI) => Promise<boolean>;
}

function locationCapGranted(effective: string[]): boolean {
  return effective.some((id) => id === 'maps-location' || id.endsWith('/maps-location'));
}

export async function assistantHasLocationGrant(api: ChatAPI): Promise<boolean> {
  const policy = await api.fetchCapabilityPolicy();
  if (policy.allow_sensitive_by_default) return true;
  const assistant = policy.agents.find((row) => row.agent.type === 'assistant');
  if (!assistant) return false;
  return locationCapGranted(assistant.state.effective ?? []);
}

async function readAndPublish(
  api: ChatAPI,
  shared: boolean,
  source: string,
): Promise<SharedDeviceLocation> {
  const pos = await getCurrentDevicePosition();
  let displayName = '';
  try {
    const rev = await api.reverseGeocode(pos.lat, pos.lon);
    displayName = (rev.display_name ?? '').trim();
  } catch {
    /* reverse is optional */
  }
  const published = await api.publishDeviceLocation({
    lat: pos.lat,
    lon: pos.lon,
    accuracy_m: pos.accuracy_m,
    display_name: displayName,
    captured_at: pos.captured_at,
    shared,
    source,
  });
  const loc = published.location ?? {};
  return {
    lat: typeof loc.lat === 'number' ? loc.lat : pos.lat,
    lon: typeof loc.lon === 'number' ? loc.lon : pos.lon,
    accuracy_m: typeof loc.accuracy_m === 'number' ? loc.accuracy_m : pos.accuracy_m,
    display_name: typeof loc.display_name === 'string' && loc.display_name.trim() ? loc.display_name : displayName,
    captured_at: typeof loc.captured_at === 'string' ? loc.captured_at : pos.captured_at,
    age_s: typeof loc.age_s === 'number' ? loc.age_s : 0,
    source,
  };
}

export const useLocationShareStore = create<LocationShareState>((set, get) => ({
  sharing: false,
  snapshot: null,
  error: null,
  busy: false,
  granted: null,
  pendingRequests: [],

  refreshGrant: async (api) => {
    try {
      const granted = await assistantHasLocationGrant(api);
      set({ granted });
      return granted;
    } catch {
      set({ granted: false });
      return false;
    }
  },

  startSharing: async (api) => {
    set({ busy: true, error: null });
    try {
      const granted = await get().refreshGrant(api);
      if (!granted) {
        set({
          busy: false,
          error: 'Grant Device location in Settings → Capabilities before sharing with Assistant.',
        });
        return false;
      }
      const snapshot = await readAndPublish(api, true, 'session');
      set({ sharing: true, snapshot, busy: false, error: null });
      return true;
    } catch (err) {
      set({ busy: false, error: geolocationErrorMessage(err) });
      return false;
    }
  },

  stopSharing: async (api) => {
    set({ busy: true, error: null });
    try {
      await api.clearDeviceLocation();
    } catch {
      /* still clear local state */
    }
    set({ sharing: false, snapshot: null, busy: false });
  },

  syncPending: async (api) => {
    try {
      const rows = await api.fetchPendingLocationRequests();
      set({
        pendingRequests: rows.map((row) => ({
          id: row.id,
          agentName: row.agent_name ?? 'Assistant',
          channel: row.channel ?? '',
          createdAt: row.created_at,
        })),
      });
    } catch {
      /* hub may be down or maps pack off */
    }
  },

  fulfillPending: async (api, requestId) => {
    set({ busy: true, error: null });
    try {
      const snapshot = await readAndPublish(api, get().sharing, 'locate');
      await api.fulfillLocationRequest(requestId, {
        lat: snapshot.lat,
        lon: snapshot.lon,
        accuracy_m: snapshot.accuracy_m,
        display_name: snapshot.display_name,
        captured_at: snapshot.captured_at,
      });
      set((state) => ({
        busy: false,
        snapshot: state.sharing ? snapshot : state.snapshot,
        pendingRequests: state.pendingRequests.filter((r) => r.id !== requestId),
      }));
    } catch (err) {
      set({ busy: false, error: geolocationErrorMessage(err) });
      throw err;
    }
  },

  rejectPending: async (api, requestId, reason) => {
    await api.rejectLocationRequest(requestId, reason);
    set((state) => ({
      pendingRequests: state.pendingRequests.filter((r) => r.id !== requestId),
    }));
  },
}));
