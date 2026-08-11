import { FIRST_WIN_BY_TRACK, FIRST_WIN_DISMISS_KEY, type FirstWinStepId } from '../config/firstWinSteps';
import { inferWizardTrackFromPacks, type WizardTrack } from '../config/wizardProfiles';

export interface FirstWinAgentLike {
  id?: string;
  name: string;
  type: string;
}

export interface FirstWinCachedAgentLike {
  name: string;
  type: string;
}

export interface FirstWinProgressInput {
  packsEnabled: Record<string, boolean> | null | undefined;
  track?: WizardTrack;
  hasWorkspace: boolean;
  agents: FirstWinAgentLike[];
  myAgents?: FirstWinCachedAgentLike[];
  hasCollaboration: boolean;
}

export interface FirstWinStepProgress {
  id: FirstWinStepId;
  complete: boolean;
  optional: boolean;
}

export interface FirstWinProgress {
  track: WizardTrack;
  steps: FirstWinStepProgress[];
  allPrimaryComplete: boolean;
}

export function getFirstWinTrack(packsEnabled: Record<string, boolean> | null | undefined): WizardTrack {
  return inferWizardTrackFromPacks(packsEnabled);
}

export function packsEnabledFromRows(
  packs: Array<{ id: string; enabled?: boolean; installed?: boolean }>,
): Record<string, boolean> {
  const enabled: Record<string, boolean> = {};
  for (const pack of packs) {
    enabled[pack.id] = pack.enabled === true && pack.installed !== false;
  }
  return enabled;
}

export function findAgentByType(agents: FirstWinAgentLike[], type: string): FirstWinAgentLike | undefined {
  return agents.find((a) => a.type === type);
}

export function findRepoAgent(
  agents: FirstWinAgentLike[],
  myAgents: FirstWinCachedAgentLike[] = [],
): FirstWinAgentLike | FirstWinCachedAgentLike | undefined {
  return agents.find((a) => a.type === 'repo') ?? myAgents.find((a) => a.type === 'repo');
}

function stepComplete(id: FirstWinStepId, input: FirstWinProgressInput): boolean {
  switch (id) {
    case 'workspace':
      return input.hasWorkspace;
    case 'repoAgent':
      return Boolean(findRepoAgent(input.agents, input.myAgents));
    case 'collaborate':
      return input.hasCollaboration;
    case 'specialist': {
      const track = input.track ?? getFirstWinTrack(input.packsEnabled);
      if (track === 'lifeSciences') return Boolean(findAgentByType(input.agents, 'biology'));
      if (track === 'cad') return Boolean(findAgentByType(input.agents, 'cad'));
      return false;
    }
    case 'askRepoExpert':
    case 'askSpecialist':
    case 'askAssistant':
    case 'modelLibrary':
    case 'discoverCommands':
      return false;
    default:
      return false;
  }
}

export function computeFirstWinProgress(input: FirstWinProgressInput): FirstWinProgress {
  const track = input.track ?? getFirstWinTrack(input.packsEnabled);
  const defs = FIRST_WIN_BY_TRACK[track].steps;
  const steps: FirstWinStepProgress[] = defs.map((def) => ({
    id: def.id,
    optional: Boolean(def.optional),
    complete: stepComplete(def.id, input),
  }));
  // Ask/prefill steps never complete from live state (empty chat unmounts the coach).
  // Auto-hide only after track setup that we can detect: developer workspace + repo expert.
  const allPrimaryComplete =
    track === 'developer' &&
    steps.filter((s) => s.id === 'workspace' || s.id === 'repoAgent').every((s) => s.complete);
  return { track, steps, allPrimaryComplete };
}

export function isFirstWinCoachDismissed(storage: Pick<Storage, 'getItem'> | null | undefined = defaultStorage()): boolean {
  if (!storage) return false;
  try {
    return storage.getItem(FIRST_WIN_DISMISS_KEY) === '1';
  } catch {
    return false;
  }
}

export const FIRST_WIN_DISMISS_EVENT = 'nj-first-win-coach-changed';

export function setFirstWinCoachDismissed(
  dismissed: boolean,
  storage: Pick<Storage, 'setItem' | 'removeItem'> | null | undefined = defaultStorage(),
): void {
  if (!storage) return;
  try {
    if (dismissed) {
      storage.setItem(FIRST_WIN_DISMISS_KEY, '1');
    } else {
      storage.removeItem(FIRST_WIN_DISMISS_KEY);
    }
  } catch {
    /* ignore quota / private mode */
  }
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(FIRST_WIN_DISMISS_EVENT));
  }
}

export function shouldShowFirstWinCoach(opts: {
  isSearching: boolean;
  dismissed: boolean;
  allPrimaryComplete: boolean;
}): boolean {
  if (opts.isSearching || opts.dismissed) return false;
  return !opts.allPrimaryComplete;
}

function defaultStorage(): Storage | null {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage;
  } catch {
    return null;
  }
}
