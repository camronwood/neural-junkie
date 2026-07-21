import { useChatStore } from '../stores/chatStore';
import { useCollaborationsStore } from '../stores/collaborationsStore';
import { useEditorStore } from '../stores/editorStore';

export interface RestartBlocker {
  id: string;
  message: string;
  save?: () => Promise<boolean>;
}

type RestartBlockerProvider = () => RestartBlocker | RestartBlocker[] | null;

const providers = new Map<string, RestartBlockerProvider>();

/** Components with local draft/job state can register a restart blocker while mounted. */
export function registerRestartBlocker(id: string, provider: RestartBlockerProvider): () => void {
  providers.set(id, provider);
  return () => providers.delete(id);
}

export function getRestartBlockers(): RestartBlocker[] {
  const blockers: RestartBlocker[] = [];
  const editor = useEditorStore.getState();
  if (editor.hasUnsavedChanges()) {
    blockers.push({
      id: 'dirty-editors',
      message: 'Unsaved editor tabs must be saved before restarting.',
      save: editor.saveAllTabs,
    });
  }

  const chat = useChatStore.getState();
  if (Object.keys(chat.streamingMessages).length > 0) {
    blockers.push({
      id: 'streaming-messages',
      message: 'An agent response is still streaming.',
    });
  }
  if ([...chat.channelThinkingAgents.values()].some((agents) => agents.size > 0)) {
    blockers.push({
      id: 'thinking-agents',
      message: 'One or more agents are still working.',
    });
  }

  const collaborations = useCollaborationsStore.getState().byID;
  if (Object.values(collaborations).some((collaboration) => collaboration.phase === 'executing')) {
    blockers.push({
      id: 'executing-collaboration',
      message: 'A collaboration is currently executing.',
    });
  }

  for (const provider of providers.values()) {
    const value = provider();
    if (Array.isArray(value)) blockers.push(...value);
    else if (value) blockers.push(value);
  }
  return blockers;
}

export async function saveRestartBlockers(): Promise<RestartBlocker[]> {
  const initial = getRestartBlockers();
  for (const blocker of initial) {
    if (blocker.save && !(await blocker.save())) {
      throw new Error(blocker.message);
    }
  }
  return getRestartBlockers();
}
