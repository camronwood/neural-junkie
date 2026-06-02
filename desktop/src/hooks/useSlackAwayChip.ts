import { useCallback, useEffect, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

export interface SlackAwayChipState {
  visible: boolean;
  awayEnabled: boolean;
  monitoringActive: boolean;
  loading: boolean;
  toggling: boolean;
  toggle: () => Promise<void>;
  refresh: () => Promise<void>;
}

export function useSlackAwayChip(): SlackAwayChipState {
  const [visible, setVisible] = useState(false);
  const [awayEnabled, setAwayEnabled] = useState(false);
  const [monitoringActive, setMonitoringActive] = useState(false);
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const api = new ChatAPI(getHubBaseURL());
      const inbox = await api.getSlackInbox();
      const h = inbox.human_dm_away;
      const show = Boolean(h?.enabled && h?.user_token_set);
      setVisible(show);
      setAwayEnabled(Boolean(h?.away_enabled));
      setMonitoringActive(h?.monitoring_status === 'monitoring_active');
    } catch {
      setVisible(false);
    } finally {
      setLoading(false);
      setToggling(false);
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

  const toggle = useCallback(async () => {
    if (toggling) return;
    setToggling(true);
    try {
      const api = new ChatAPI(getHubBaseURL());
      await api.setSlackInboxAwayEnabled(!awayEnabled);
      await refresh();
    } catch {
      setToggling(false);
    }
  }, [awayEnabled, refresh, toggling]);

  return {
    visible,
    awayEnabled,
    monitoringActive,
    loading,
    toggling,
    toggle,
    refresh,
  };
}
