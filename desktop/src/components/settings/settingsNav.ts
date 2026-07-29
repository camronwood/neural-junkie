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
  | 'about';

/** Map deprecated tab ids from older deep links. */
export const SETTINGS_TAB_ALIASES: Record<string, SettingsTab> = {
  'ai-providers': 'providers',
};

export function resolveSettingsTab(tab?: string): SettingsTab | undefined {
  if (!tab) return undefined;
  if (SETTINGS_TAB_ALIASES[tab]) return SETTINGS_TAB_ALIASES[tab];
  return tab as SettingsTab;
}

export type SettingsNavGroup = {
  title: string;
  items: Array<{ id: SettingsTab; label: string }>;
};

export const SETTINGS_NAV_GROUPS: SettingsNavGroup[] = [
  {
    title: 'General',
    items: [
      { id: 'appearance', label: 'Appearance' },
      { id: 'layout', label: 'Layout' },
      { id: 'keyboard', label: 'Keyboard' },
      { id: 'chat', label: 'Chat' },
      { id: 'connection', label: 'Connection' },
    ],
  },
  {
    title: 'AI',
    items: [
      { id: 'providers', label: 'Providers' },
      { id: 'models-performance', label: 'Models & performance' },
      { id: 'inference-usage', label: 'Usage & cost' },
      { id: 'collab-routing', label: 'Routing & collab' },
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
      { id: 'about', label: 'About' },
    ],
  },
];

export const ALL_SETTINGS_TABS: SettingsTab[] = SETTINGS_NAV_GROUPS.flatMap((g) =>
  g.items.map((i) => i.id)
);
