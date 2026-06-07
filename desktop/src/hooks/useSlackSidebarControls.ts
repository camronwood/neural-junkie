import { useCallback, useEffect, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

export interface SlackSidebarControlsState {
  slackConnected: boolean;
  awayVisible: boolean;
  awayEnabled: boolean;
  awayMonitoringActive: boolean;
  forwardVisible: boolean;
  forwardEnabled: boolean;
  inboxEnabled: boolean;
  loading: boolean;
  awayToggling: boolean;
  forwardToggling: boolean;
  toggleAway: () => Promise<void>;
  toggleForward: () => Promise<void>;
  refresh: () => Promise<void>;
}

export function useSlackSidebarControls(): SlackSidebarControlsState {
  const [slackConnected, setSlackConnected] = useState(false);
  const [awayVisible, setAwayVisible] = useState(false);
  const [awayEnabled, setAwayEnabled] = useState(false);
  const [awayMonitoringActive, setAwayMonitoringActive] = useState(false);
  const [forwardVisible, setForwardVisible] = useState(false);
  const [forwardEnabled, setForwardEnabled] = useState(false);
  const [inboxEnabled, setInboxEnabled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [awayToggling, setAwayToggling] = useState(false);
  const [forwardToggling, setForwardToggling] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const api = new ChatAPI(getHubBaseURL());
      const [status, inbox] = await Promise.all([api.getSlackStatus(), api.getSlackInbox()]);
      const connected = Boolean(status?.configured || status?.token_set || status?.connected);
      setSlackConnected(connected);

      const h = inbox.human_dm_away;
      const showAway = Boolean(h?.enabled && h?.user_token_set);
      setAwayVisible(showAway);
      setAwayEnabled(Boolean(h?.away_enabled));
      setAwayMonitoringActive(h?.monitoring_status === 'monitoring_active');

      setInboxEnabled(Boolean(inbox.enabled));
      setForwardVisible(connected && Boolean(inbox.enabled));
      setForwardEnabled(Boolean(inbox.forward_enabled));
    } catch {
      setSlackConnected(false);
      setAwayVisible(false);
      setForwardVisible(false);
    } finally {
      setLoading(false);
      setAwayToggling(false);
      setForwardToggling(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 30_000);
    const onFocus = () => void refresh();
    const onInboxUpdated = () => void refresh();
    window.addEventListener('focus', onFocus);
    window.addEventListener('nj-slack-inbox-updated', onInboxUpdated);
    return () => {
      window.clearInterval(id);
      window.removeEventListener('focus', onFocus);
      window.removeEventListener('nj-slack-inbox-updated', onInboxUpdated);
    };
  }, [refresh]);

  const toggleAway = useCallback(async () => {
    if (awayToggling) return;
    setAwayToggling(true);
    try {
      const api = new ChatAPI(getHubBaseURL());
      await api.setSlackInboxAwayEnabled(!awayEnabled);
      window.dispatchEvent(new Event('nj-slack-inbox-updated'));
      await refresh();
    } catch {
      setAwayToggling(false);
    }
  }, [awayEnabled, awayToggling, refresh]);

  const toggleForward = useCallback(async () => {
    if (forwardToggling || !inboxEnabled) return;
    setForwardToggling(true);
    try {
      const api = new ChatAPI(getHubBaseURL());
      await api.setSlackInboxForwardEnabled(!forwardEnabled);
      window.dispatchEvent(new Event('nj-slack-inbox-updated'));
      await refresh();
    } catch {
      setForwardToggling(false);
    }
  }, [forwardEnabled, forwardToggling, inboxEnabled, refresh]);

  return {
    slackConnected,
    awayVisible,
    awayEnabled,
    awayMonitoringActive,
    forwardVisible,
    forwardEnabled,
    inboxEnabled,
    loading,
    awayToggling,
    forwardToggling,
    toggleAway,
    toggleForward,
    refresh,
  };
}

/** @deprecated use useSlackSidebarControls */
export function useSlackAwayChip() {
  const s = useSlackSidebarControls();
  return {
    visible: s.awayVisible,
    awayEnabled: s.awayEnabled,
    monitoringActive: s.awayMonitoringActive,
    loading: s.loading,
    toggling: s.awayToggling,
    toggle: s.toggleAway,
    refresh: s.refresh,
  };
}
