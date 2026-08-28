import type { MutableRefObject } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { useChatStore } from '../stores/chatStore';
import { useSettingsStore } from '../stores/settingsStore';
import type { Settings } from '../stores/settingsStore';
import type { Toast } from '../stores/toastStore';
import type { Channel } from '../types/protocol';
import {
  agentSidebarHideKey,
  dmChannelNamesForAgent,
  predictedDmChannelName,
} from '../utils/dmChannelDisplay';
import { patchRevealForChannel, patchRevealSidebarItems } from '../utils/sidebarVisibility';

export type ChatDmActionsDeps = {
  api: ChatAPI;
  username: string;
  loadChannels: () => Promise<unknown>;
  loadAgents: () => Promise<unknown>;
  handleSwitchChannel: (channelName: string) => Promise<void>;
  updateSettings: (patch: Partial<Settings>) => Promise<void>;
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => void;
  dmCreateInFlightRef: MutableRefObject<Map<string, Promise<void>>>;
  dmOpenChainRef: MutableRefObject<Promise<void>>;
};

/** DM channel open/create handlers extracted from ChatWindow. */
export function createChatDmActions(deps: ChatDmActionsDeps) {
  const handleCreateDM = async (agentId: string) => {
    const pending = deps.dmCreateInFlightRef.current.get(agentId);
    if (pending) {
      await pending;
      return;
    }

    const run = (async () => {
      const openOne = async () => {
        try {
          const st = useChatStore.getState();
          const agent = st.agents.find((a) => a.id === agentId);
          if (agent) {
            const predicted = predictedDmChannelName(deps.username, agent.name);
            const existingByName = st.channels.find((c) => c.name === predicted);
            if (existingByName) {
              await deps.handleSwitchChannel(existingByName.name);
              return;
            }
            const existingName = dmChannelNamesForAgent(st.channels, agent)[0];
            if (existingName) {
              await deps.handleSwitchChannel(existingName);
              return;
            }
            const byMembership = st.channels.find(
              (c) =>
                c.type === 'dm' &&
                (c.agents?.some((a) => a.id === agentId) || c.members?.includes(agentId))
            );
            if (byMembership) {
              await deps.handleSwitchChannel(byMembership.name);
              return;
            }
          }

          const ch = await deps.api.openDM(agentId, deps.username);
          const prevChannels = useChatStore.getState().channels;
          if (!prevChannels.some((c) => c.name === ch.name)) {
            useChatStore.getState().setChannels([...prevChannels, ch]);
          }
          const { settings, isLoaded } = useSettingsStore.getState();
          if (isLoaded) {
            const patch = patchRevealSidebarItems(settings, {
              agentIds: [agentId],
              agentSidebarKeys: agent ? [agentSidebarHideKey(agent)] : undefined,
              dmChannelNames: [ch.name],
            });
            if (patch) {
              void deps.updateSettings(patch);
            }
          }
          void deps.loadChannels();
          await deps.handleSwitchChannel(ch.name);
        } catch (error) {
          console.error('Failed to create DM channel:', error);
          const msg = error instanceof Error ? error.message : 'Failed to create DM channel.';
          deps.addToast({
            type: 'error',
            title: 'Could not open direct message',
            message: /too many requests/i.test(msg)
              ? 'Too many channel requests — wait a few seconds and try again.'
              : msg,
          });
        }
      };

      const chained = deps.dmOpenChainRef.current.then(openOne, openOne);
      deps.dmOpenChainRef.current = chained.then(
        () => undefined,
        () => undefined
      );
      await chained;
    })();

    deps.dmCreateInFlightRef.current.set(agentId, run);
    try {
      await run;
    } finally {
      deps.dmCreateInFlightRef.current.delete(agentId);
    }
  };

  const handleNewDmCreated = async (ch: Channel) => {
    try {
      deps.addToast({
        type: 'success',
        title: 'Direct message ready',
        message: `Opened ${ch.description || ch.name}`,
      });
      const channelList = await deps.api.fetchChannels();
      const merged = channelList.some((c) => c.name === ch.name)
        ? channelList
        : [...channelList, ch];
      useChatStore.getState().setChannels(merged);
      await deps.loadAgents();
      const { settings, isLoaded } = useSettingsStore.getState();
      if (isLoaded) {
        const patch = patchRevealForChannel(
          settings,
          ch.name,
          merged,
          useChatStore.getState().agents
        );
        if (patch) {
          void deps.updateSettings(patch);
        }
      }
      await deps.handleSwitchChannel(ch.name);
    } catch (e) {
      console.error('Failed after creating DM agent:', e);
      deps.addToast({
        type: 'error',
        title: 'Could not open DM',
        message: e instanceof Error ? e.message : 'Unknown error',
      });
    }
  };

  return {
    handleCreateDM,
    handleNewDmCreated,
  };
}
