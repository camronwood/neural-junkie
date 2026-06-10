import type {
  AgentInfo,
  Collaboration,
  CollaborationAgent,
  CommandDefinition,
} from '../../types/protocol';

const CLIENT_PALETTE_COMMANDS: CommandDefinition[] = [
  {
    name: '/nj-open-model-library',
    description: 'Open model library (Ollama & Hugging Face — download, install, assign to agents)',
    category: 'Neural Junkie',
    arguments: [],
  },
];

export function withClientPaletteCommands(defs: CommandDefinition[]): CommandDefinition[] {
  const names = new Set(CLIENT_PALETTE_COMMANDS.map((c) => c.name));
  return [...CLIENT_PALETTE_COMMANDS, ...defs.filter((d) => !names.has(d.name))];
}

export function showRunbookBuilderForCollab(collab: Collaboration): boolean {
  return (
    collab.source === 'runbook' &&
    (collab.phase === 'draft' || collab.phase === 'reviewing')
  );
}

export function agentsToCollaborationAgents(agents: AgentInfo[]): CollaborationAgent[] {
  return agents.map((a) => ({
    agent_id: a.id,
    agent_name: a.name,
    agent_type: a.type,
    expertise: a.expertise ?? [],
    role: '',
  }));
}
