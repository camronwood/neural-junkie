import type {
  SlackConfigResponse,
  SlackForwardRule,
  SlackInboxConfig,
  SlackStatus,
} from '../../types/protocol';

export function slackCanListChannelsFrom(
  status: SlackStatus | null | undefined,
  cfg: SlackConfigResponse | null | undefined
): boolean {
  return Boolean(cfg?.bot_token_set || status?.token_set || status?.configured);
}

export function defaultSlackInboxForm(): SlackInboxConfig {
  return {
    enabled: false,
    agent_id: '',
    forward_rules: [
      { id: 'mentions', type: 'mention_of_me', enabled: false, slack_channel_ids: [] },
      { id: 'nj-prefix', type: 'prefix', enabled: false, prefix: 'nj:', slack_channel_ids: ['*'] },
      { id: 'robot-react', type: 'reaction', enabled: false, emoji: 'robot_face', slack_channel_ids: [] },
    ],
    human_dm_away: {
      enabled: false,
      away_enabled: false,
      schedule_enabled: false,
      schedule_timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'America/Los_Angeles',
    },
  };
}

export function mergeSlackInboxForm(inbox: SlackInboxConfig | null | undefined): SlackInboxConfig {
  const base = defaultSlackInboxForm();
  if (!inbox) return base;
  const rules =
    (inbox.forward_rules?.length ? inbox.forward_rules : base.forward_rules) ??
    base.forward_rules ??
    [];
  const byId = new Map(rules.map((r) => [r.id ?? r.type, r]));
  for (const def of base.forward_rules ?? []) {
    if (!byId.has(def.id ?? def.type)) {
      byId.set(def.id ?? def.type, def);
    }
  }
  return {
    ...base,
    ...inbox,
    forward_rules: Array.from(byId.values()),
    human_dm_away: { ...base.human_dm_away, ...inbox.human_dm_away },
  };
}

export function updateForwardRule(
  inbox: SlackInboxConfig,
  ruleId: string,
  patch: Partial<SlackForwardRule>
): SlackInboxConfig {
  const rules = (inbox.forward_rules ?? []).map((r) =>
    (r.id ?? r.type) === ruleId ? { ...r, ...patch } : r
  );
  return { ...inbox, forward_rules: rules };
}
