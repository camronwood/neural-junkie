import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { useHubAccessToken, useHubSessionToken } from '../config/hubUrl';
import { useChatStore } from '../stores/chatStore';
import type { LayoutSettings } from '../stores/settingsStore';
import type { Channel } from '../types/protocol';
import { isSlackHubChannelName } from '../utils/slackChannelDisplay';
import { loadAgentsFromHub, loadChannelsFromHub } from './useChatWindowData';

export interface UseChatWindowHubBootstrapArgs {
  serverAddr: string;
  channel: string;
  channels: Channel[];
  layoutSettings: LayoutSettings;
  loadLayoutSettings: () => void | Promise<void>;
  setFileExplorerOpen: (open: boolean) => void;
  setCodeEditorOpen: (open: boolean) => void;
  setPanelOpen: (open: boolean) => void;
  setMyAgentsPanelOpen: (open: boolean) => void;
}

/**
 * Hub API client, WebSocket URL, layout-panel sync, and agent/channel loaders for ChatWindow.
 * Extraction loop 5 from ChatWindowRoot.
 */
export function useChatWindowHubBootstrap({
  serverAddr,
  channel,
  channels,
  layoutSettings,
  loadLayoutSettings,
  setFileExplorerOpen,
  setCodeEditorOpen,
  setPanelOpen,
  setMyAgentsPanelOpen,
}: UseChatWindowHubBootstrapArgs) {
  const [totalAgentsCount, setTotalAgentsCount] = useState(0);
  const api = useMemo(() => new ChatAPI(serverAddr), [serverAddr]);
  const hubHttp = useMemo(
    () => (serverAddr.startsWith('http') ? serverAddr : `http://${serverAddr}`),
    [serverAddr]
  );
  const channelNamesKey = useMemo(
    () =>
      channels
        .map((c) => c.name)
        .slice()
        .sort()
        .join('\0'),
    [channels],
  );
  const hubSessionToken = useHubSessionToken();
  const hubAccessToken = useHubAccessToken();
  // Debounce slack extra-channel subscriptions so channel list hydration does not thrash the socket.
  const [stableChannelNamesKey, setStableChannelNamesKey] = useState(channelNamesKey);
  useEffect(() => {
    const timer = window.setTimeout(() => setStableChannelNamesKey(channelNamesKey), 800);
    return () => window.clearTimeout(timer);
  }, [channelNamesKey]);

  // Hub restart wipes in-memory sessions; re-mint if we reached chat without a token.
  useEffect(() => {
    if (hubSessionToken) return;
    const name = useChatStore.getState().username?.trim() || 'Anonymous';
    void api.createSession(name).catch((err) => {
      console.error('[ChatWindow] Failed to restore hub session for WebSocket:', err);
    });
  }, [api, hubSessionToken]);

  const wsURL = useMemo(() => {
    const slackExtra = stableChannelNamesKey
      .split('\0')
      .filter((name) => name && isSlackHubChannelName(name) && name !== channel);
    return api.getWebSocketURL(channel, slackExtra);
  }, [api, channel, stableChannelNamesKey, hubSessionToken, hubAccessToken]);

  const wsOutageToastShownRef = useRef(false);
  /** Primary channel from the last successful WS connect — used to skip full reload on channel-only URL changes. */
  const wsConnectedChannelRef = useRef<string | null>(null);

  // Load layout settings on mount
  useEffect(() => {
    loadLayoutSettings();
  }, [loadLayoutSettings]);

  // Apply layout settings when specific panel keys change (not whole object — avoids chat toggle resetting files/editor).
  useEffect(() => {
    if (layoutSettings) {
      setFileExplorerOpen(layoutSettings.filesPanelVisible);
    }
  }, [layoutSettings?.filesPanelVisible]);

  useEffect(() => {
    if (layoutSettings) {
      setCodeEditorOpen(layoutSettings.editorPanelVisible);
    }
  }, [layoutSettings?.editorPanelVisible]);

  useEffect(() => {
    if (layoutSettings) {
      setPanelOpen(layoutSettings.terminalPanelVisible);
    }
  }, [layoutSettings?.terminalPanelVisible, setPanelOpen]);

  useEffect(() => {
    if (layoutSettings) {
      setMyAgentsPanelOpen(layoutSettings.myAgentsPanelVisible);
    }
  }, [layoutSettings?.myAgentsPanelVisible, setMyAgentsPanelOpen]);

  // Load agents function
  const loadAgents = useCallback(async () => {
    try {
      await loadAgentsFromHub(api);
    } catch (error) {
      console.error('Failed to load agents:', error);
    }
  }, [api]);

  // Load counts for badges
  const loadCounts = useCallback(async () => {
    try {
      const [myAgents, removedAgents] = await Promise.all([
        api.fetchMyAgents(),
        api.fetchRemovedAgents()
      ]);
      setTotalAgentsCount(myAgents.length + removedAgents.length);
    } catch (error) {
      console.error('Failed to load counts:', error);
    }
  }, [api]);

  // Load channels
  const loadChannels = useCallback(async () => {
    try {
      return await loadChannelsFromHub(api);
    } catch (error) {
      console.error('Failed to load channels:', error);
      return null;
    }
  }, [api]);

  // Pick up new Slack inbox peer channels for background WS watch.
  useEffect(() => {
    void loadChannels();
    const onSlackInboxUpdated = () => void loadChannels();
    const onRoomChannelsUpdated = () => void loadChannels();
    window.addEventListener('nj-slack-inbox-updated', onSlackInboxUpdated);
    window.addEventListener('nj-room-channels-updated', onRoomChannelsUpdated);
    const id = window.setInterval(() => void loadChannels(), 5_000);
    return () => {
      window.clearInterval(id);
      window.removeEventListener('nj-slack-inbox-updated', onSlackInboxUpdated);
      window.removeEventListener('nj-room-channels-updated', onRoomChannelsUpdated);
    };
  }, [loadChannels]);

  return {
    api,
    hubHttp,
    hubSessionToken,
    wsURL,
    wsOutageToastShownRef,
    wsConnectedChannelRef,
    totalAgentsCount,
    loadAgents,
    loadCounts,
    loadChannels,
  };
}
