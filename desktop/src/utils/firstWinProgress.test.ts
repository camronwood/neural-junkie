import { afterEach, describe, expect, it } from 'vitest';
import { FIRST_WIN_DISMISS_KEY } from '../config/firstWinSteps';
import {
  computeFirstWinProgress,
  findAgentByType,
  findRepoAgent,
  getFirstWinTrack,
  isFirstWinCoachDismissed,
  packsEnabledFromRows,
  setFirstWinCoachDismissed,
  shouldShowFirstWinCoach,
} from './firstWinProgress';

const storage = (): Storage => {
  const map = new Map<string, string>();
  return {
    get length() {
      return map.size;
    },
    clear: () => map.clear(),
    getItem: (key) => map.get(key) ?? null,
    key: (i) => [...map.keys()][i] ?? null,
    removeItem: (key) => {
      map.delete(key);
    },
    setItem: (key, value) => {
      map.set(key, value);
    },
  };
};

afterEach(() => {
  localStorage.removeItem(FIRST_WIN_DISMISS_KEY);
});

describe('getFirstWinTrack', () => {
  it('infers developer from ide / software-development packs', () => {
    expect(getFirstWinTrack({ ide: true })).toBe('developer');
    expect(getFirstWinTrack({ 'software-development': true })).toBe('developer');
  });

  it('prefers life-sciences, then cad, then general', () => {
    expect(getFirstWinTrack({ 'life-sciences': true, ide: true })).toBe('lifeSciences');
    expect(getFirstWinTrack({ cad: true })).toBe('cad');
    expect(getFirstWinTrack({})).toBe('general');
  });
});

describe('packsEnabledFromRows', () => {
  it('maps installed+enabled pack rows', () => {
    expect(
      packsEnabledFromRows([
        { id: 'ide', enabled: true, installed: true },
        { id: 'cad', enabled: true, installed: false },
        { id: 'life-sciences', enabled: false, installed: true },
      ]),
    ).toEqual({
      ide: true,
      cad: false,
      'life-sciences': false,
    });
  });
});

describe('computeFirstWinProgress', () => {
  it('marks developer workspace and repo agent from live state', () => {
    const progress = computeFirstWinProgress({
      packsEnabled: { ide: true },
      hasWorkspace: true,
      agents: [{ id: 'r1', name: 'MyRepoExpert', type: 'repo' }],
      hasCollaboration: false,
    });
    expect(progress.track).toBe('developer');
    expect(progress.steps.find((s) => s.id === 'workspace')?.complete).toBe(true);
    expect(progress.steps.find((s) => s.id === 'repoAgent')?.complete).toBe(true);
    expect(progress.steps.find((s) => s.id === 'collaborate')?.complete).toBe(false);
    expect(progress.steps.find((s) => s.id === 'collaborate')?.optional).toBe(true);
    expect(progress.allPrimaryComplete).toBe(true);
  });

  it('treats cached myAgents repo type as a repo expert', () => {
    const progress = computeFirstWinProgress({
      packsEnabled: { 'software-development': true },
      hasWorkspace: true,
      agents: [],
      myAgents: [{ name: 'CachedRepo', type: 'repo' }],
      hasCollaboration: false,
    });
    expect(progress.steps.find((s) => s.id === 'repoAgent')?.complete).toBe(true);
    expect(progress.allPrimaryComplete).toBe(true);
  });

  it('does not auto-complete developer setup without a workspace', () => {
    const progress = computeFirstWinProgress({
      packsEnabled: { ide: true },
      hasWorkspace: false,
      agents: [{ id: 'r1', name: 'MyRepoExpert', type: 'repo' }],
      hasCollaboration: true,
    });
    expect(progress.allPrimaryComplete).toBe(false);
    expect(progress.steps.find((s) => s.id === 'collaborate')?.complete).toBe(true);
  });

  it('marks BiologyExpert present on the life-sciences track without auto-hiding', () => {
    const progress = computeFirstWinProgress({
      packsEnabled: { 'life-sciences': true },
      hasWorkspace: false,
      agents: [{ id: 'b1', name: 'BiologyExpert', type: 'biology' }],
      hasCollaboration: false,
    });
    expect(progress.track).toBe('lifeSciences');
    expect(progress.steps.find((s) => s.id === 'specialist')?.complete).toBe(true);
    expect(progress.allPrimaryComplete).toBe(false);
  });

  it('marks CADExpert present on the cad track without auto-hiding', () => {
    const progress = computeFirstWinProgress({
      packsEnabled: { cad: true },
      hasWorkspace: true,
      agents: [{ id: 'c1', name: 'CADExpert', type: 'cad' }],
      hasCollaboration: false,
    });
    expect(progress.track).toBe('cad');
    expect(progress.steps.find((s) => s.id === 'workspace')?.complete).toBe(true);
    expect(progress.steps.find((s) => s.id === 'specialist')?.complete).toBe(true);
    expect(progress.allPrimaryComplete).toBe(false);
  });

  it('keeps general track visible after setup', () => {
    const progress = computeFirstWinProgress({
      packsEnabled: {},
      hasWorkspace: false,
      agents: [{ id: 'a1', name: 'Assistant', type: 'assistant' }],
      hasCollaboration: false,
    });
    expect(progress.track).toBe('general');
    expect(progress.allPrimaryComplete).toBe(false);
  });
});

describe('agent finders', () => {
  it('finds specialists and repo agents', () => {
    const agents = [
      { id: 'a', name: 'Assistant', type: 'assistant' },
      { id: 'b', name: 'BiologyExpert', type: 'biology' },
    ];
    expect(findAgentByType(agents, 'biology')?.name).toBe('BiologyExpert');
    expect(findRepoAgent(agents, [{ name: 'Repo', type: 'repo' }])?.name).toBe('Repo');
  });
});

describe('dismiss persistence', () => {
  it('reads and writes the dismiss flag', () => {
    const mem = storage();
    expect(isFirstWinCoachDismissed(mem)).toBe(false);
    setFirstWinCoachDismissed(true, mem);
    expect(mem.getItem(FIRST_WIN_DISMISS_KEY)).toBe('1');
    expect(isFirstWinCoachDismissed(mem)).toBe(true);
    setFirstWinCoachDismissed(false, mem);
    expect(isFirstWinCoachDismissed(mem)).toBe(false);
  });
});

describe('shouldShowFirstWinCoach', () => {
  it('hides while searching, when dismissed, or when developer setup is done', () => {
    expect(shouldShowFirstWinCoach({ isSearching: true, dismissed: false, allPrimaryComplete: false })).toBe(false);
    expect(shouldShowFirstWinCoach({ isSearching: false, dismissed: true, allPrimaryComplete: false })).toBe(false);
    expect(shouldShowFirstWinCoach({ isSearching: false, dismissed: false, allPrimaryComplete: true })).toBe(false);
    expect(shouldShowFirstWinCoach({ isSearching: false, dismissed: false, allPrimaryComplete: false })).toBe(true);
  });
});
