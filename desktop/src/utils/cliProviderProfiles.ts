export interface CLIProfilePreset {
  id: string;
  label: string;
  model: string;
  buttonClass: string;
}

export interface CLIProviderProfileConfig {
  providerType: string;
  title: string;
  description: string;
  presets: CLIProfilePreset[];
}

/** Preset model profiles for auto-detected CLI providers (Settings → AI Providers). */
export const CLI_PROVIDER_PROFILES: CLIProviderProfileConfig[] = [
  {
    providerType: 'gemini-cli',
    title: 'Gemini',
    description: 'Fast uses Flash for low latency. Deep uses Pro for higher quality.',
    presets: [
      { id: 'fast', label: 'Fast (Flash)', model: 'gemini-2.5-flash', buttonClass: 'bg-emerald-600 hover:bg-emerald-500' },
      { id: 'deep', label: 'Deep (Pro)', model: 'gemini-2.5-pro', buttonClass: 'bg-indigo-600 hover:bg-indigo-500' },
    ],
  },
  {
    providerType: 'cursor-cli',
    title: 'Cursor',
    description: 'Pick the model passed to Cursor Agent (--model). Auto lets Cursor choose.',
    presets: [
      { id: 'auto', label: 'Auto', model: 'auto', buttonClass: 'bg-gray-600 hover:bg-gray-500' },
      { id: 'fast', label: 'Fast (Sonnet)', model: 'sonnet', buttonClass: 'bg-emerald-600 hover:bg-emerald-500' },
      { id: 'deep', label: 'Deep (Opus)', model: 'opus', buttonClass: 'bg-indigo-600 hover:bg-indigo-500' },
    ],
  },
  {
    providerType: 'claude-cli',
    title: 'Claude Code CLI',
    description: 'Model for the Claude Code subprocess (claude --model). Not the same as Anthropic API below.',
    presets: [
      { id: 'fast', label: 'Fast (Haiku)', model: 'haiku', buttonClass: 'bg-emerald-600 hover:bg-emerald-500' },
      { id: 'balanced', label: 'Balanced (Sonnet)', model: 'sonnet', buttonClass: 'bg-blue-600 hover:bg-blue-500' },
      { id: 'deep', label: 'Deep (Opus)', model: 'opus', buttonClass: 'bg-indigo-600 hover:bg-indigo-500' },
    ],
  },
  {
    providerType: 'codex-cli',
    title: 'Codex',
    description: 'Model for OpenAI Codex CLI (codex exec --model).',
    presets: [
      { id: 'fast', label: 'Fast (Mini)', model: 'o4-mini', buttonClass: 'bg-emerald-600 hover:bg-emerald-500' },
      { id: 'deep', label: 'Deep (Full)', model: 'o3', buttonClass: 'bg-indigo-600 hover:bg-indigo-500' },
    ],
  },
  {
    providerType: 'copilot-cli',
    title: 'Copilot CLI',
    description: 'Model for GitHub Copilot CLI (copilot --model).',
    presets: [
      { id: 'fast', label: 'Fast', model: 'gpt-4.1', buttonClass: 'bg-emerald-600 hover:bg-emerald-500' },
      { id: 'deep', label: 'Deep', model: 'claude-sonnet-4', buttonClass: 'bg-indigo-600 hover:bg-indigo-500' },
    ],
  },
];

export function detectCLIProfileLabel(
  config: CLIProviderProfileConfig,
  model?: string
): string {
  const normalized = (model || '').toLowerCase();
  const match = config.presets.find(
    (p) => p.model.toLowerCase() === normalized || normalized.includes(p.model.toLowerCase())
  );
  return match?.label ?? (model || 'default');
}
