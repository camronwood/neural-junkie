export type SettingsTab =
  | 'appearance'
  | 'layout'
  | 'keyboard'
  | 'chat'
  | 'connection'
  | 'providers'
  | 'models-performance'
  | 'inference-usage'
  | 'collab-routing'
  | 'memory-learning'
  | 'capabilities'
  | 'api-credentials'
  | 'integrations'
  | 'web-search'
  | 'assistant-tools'
  | 'slack'
  | 'streams'
  | 'security'
  | 'server-network'
  | 'automation'
  | 'activity'
  | 'about'
  | 'domain-packs';

/** Opens the Domain packs modal (toolbar) — not an inline settings panel. */
export const DOMAIN_PACKS_SETTINGS_ACTION = 'domain-packs' as const;

/** Map deprecated tab ids from older deep links. */
export const SETTINGS_TAB_ALIASES: Record<string, SettingsTab> = {
  'ai-providers': 'providers',
};

export function resolveSettingsTab(tab?: string): SettingsTab | undefined {
  if (!tab) return undefined;
  if (SETTINGS_TAB_ALIASES[tab]) return SETTINGS_TAB_ALIASES[tab];
  return tab as SettingsTab;
}

export function isDomainPacksSettingsAction(tab?: string): boolean {
  return tab === DOMAIN_PACKS_SETTINGS_ACTION;
}

export type SettingsNavGroup = {
  title: string;
  items: Array<{ id: SettingsTab; label: string; action?: 'open-domain-packs' }>;
};

/** Essentials — shown by default in Settings nav. */
export const SETTINGS_ESSENTIALS_GROUP: SettingsNavGroup = {
  title: 'Essentials',
  items: [
    { id: 'appearance', label: 'Appearance' },
    { id: 'connection', label: 'Connection' },
    { id: 'providers', label: 'Providers' },
    { id: 'collab-routing', label: 'Routing & collab' },
    { id: 'domain-packs', label: 'Domain packs', action: 'open-domain-packs' },
    { id: 'about', label: 'About' },
  ],
};

/** Advanced — collapsed by default. */
export const SETTINGS_ADVANCED_GROUPS: SettingsNavGroup[] = [
  {
    title: 'General',
    items: [
      { id: 'layout', label: 'Layout' },
      { id: 'keyboard', label: 'Keyboard' },
      { id: 'chat', label: 'Chat' },
    ],
  },
  {
    title: 'AI',
    items: [
      { id: 'models-performance', label: 'Models & performance' },
      { id: 'inference-usage', label: 'Usage & cost' },
      { id: 'memory-learning', label: 'Memory & learning' },
      { id: 'capabilities', label: 'Capabilities' },
    ],
  },
  {
    title: 'External',
    items: [
      { id: 'api-credentials', label: 'API credentials' },
      { id: 'integrations', label: 'Integrations' },
      { id: 'web-search', label: 'Web search' },
      { id: 'assistant-tools', label: 'Assistant tools' },
      { id: 'slack', label: 'Slack' },
      { id: 'streams', label: 'Streams' },
    ],
  },
  {
    title: 'Advanced',
    items: [
      { id: 'security', label: 'Security' },
      { id: 'server-network', label: 'Server & network' },
      { id: 'automation', label: 'Automation & testing' },
      { id: 'activity', label: 'Activity' },
    ],
  },
];

/** @deprecated use SETTINGS_ESSENTIALS_GROUP + SETTINGS_ADVANCED_GROUPS */
export const SETTINGS_NAV_GROUPS: SettingsNavGroup[] = [
  SETTINGS_ESSENTIALS_GROUP,
  ...SETTINGS_ADVANCED_GROUPS,
];

export const ALL_SETTINGS_TABS: SettingsTab[] = [
  ...SETTINGS_ESSENTIALS_GROUP.items.map((i) => i.id),
  ...SETTINGS_ADVANCED_GROUPS.flatMap((g) => g.items.map((i) => i.id)),
].filter((id) => id !== 'domain-packs');
