import { describe, expect, it } from 'vitest';
import type { Settings } from '../stores/settingsStore';
import type { AgentInfo, Channel } from '../types/protocol';
import { agentSidebarHideKey } from './dmChannelDisplay';
import {
  isAgentShortcutHidden,
  isDmChannelVisibleInSidebar,
  patchRevealActiveAgentsInSidebar,
  patchRevealSidebarItems,
  patchUnhideAgentShortcutsOnActivation,
} from './sidebarVisibility';

const baseSettings: Settings = {
  fontSize: 16,
  fontSizeScope: 'messages',
  hiddenAgentSidebarKeys: ['backend:GoExpert'],
  hiddenAgentIdsForSidebar: ['legacy-uuid'],
  hiddenDmChannelNames: ['dm-u-goexpert'],
  hiddenCollaborationChannelNames: ['collab-abc'],
};

const agent = (id: string, name: string, status: string, type = 'backend'): AgentInfo => ({
  id,
  name,
  type: type as AgentInfo['type'],
  expertise: [],
  status,
  model: 'test',
  is_paused: false,
});

const dmChannel: Channel = {
  id: '1',
  name: 'dm-u-goexpert',
  type: 'dm',
  description: 'Direct message with GoExpert',
  agents: [{ id: 'go-2', name: 'GoExpert', type: 'backend' } as AgentInfo],
};

describe('isAgentShortcutHidden', () => {
  it('matches stable key and legacy id', () => {
    const a = agent('legacy-uuid', 'GoExpert', 'active');
    expect(isAgentShortcutHidden(baseSettings, a)).toBe(true);
    expect(isAgentShortcutHidden(baseSettings, agent('new-uuid', 'GoExpert', 'active'))).toBe(true);
    expect(isAgentShortcutHidden(baseSettings, agent('x', 'RustExpert', 'active', 'rust'))).toBe(false);
  });
});

describe('patchRevealSidebarItems', () => {
  it('removes matching hidden entries', () => {
    const patch = patchRevealSidebarItems(baseSettings, {
      agentSidebarKeys: ['backend:GoExpert'],
      dmChannelNames: ['dm-u-goexpert'],
      collabChannelNames: ['collab-abc'],
    });
    expect(patch?.hiddenAgentSidebarKeys).toEqual([]);
    expect(patch?.hiddenDmChannelNames).toEqual([]);
    expect(patch?.hiddenCollaborationChannelNames).toEqual([]);
  });
});

describe('patchUnhideAgentShortcutsOnActivation', () => {
  it('does not unhide before seeding', () => {
    const agents = [agent('go-2', 'GoExpert', 'active')];
    const { patch } = patchUnhideAgentShortcutsOnActivation(
      baseSettings,
      agents,
      [dmChannel],
      {},
      { seeded: false }
    );
    expect(patch).toBeNull();
  });

  it('unhides by stable key when agent returns with new id', () => {
    const agents = [agent('go-2', 'GoExpert', 'active')];
    const { patch } = patchUnhideAgentShortcutsOnActivation(
      baseSettings,
      agents,
      [dmChannel],
      {},
      { seeded: true }
    );
    expect(patch?.hiddenAgentSidebarKeys).toEqual([]);
    expect(patch?.hiddenDmChannelNames).toEqual([]);
  });

  it('does not unhide when user hid an already-active shortcut', () => {
    const agents = [agent('go-2', 'GoExpert', 'active')];
    const { patch } = patchUnhideAgentShortcutsOnActivation(
      baseSettings,
      agents,
      [],
      { 'go-2': 'active' },
      { seeded: true }
    );
    expect(patch).toBeNull();
  });
});

describe('patchRevealActiveAgentsInSidebar', () => {
  it('reveals hidden idle agents', () => {
    const patch = patchRevealActiveAgentsInSidebar(
      baseSettings,
      [agent('go-2', 'GoExpert', 'idle')],
      [dmChannel]
    );
    expect(patch?.hiddenAgentSidebarKeys).toEqual([]);
    expect(patch?.hiddenDmChannelNames).toEqual([]);
  });

  it('reveals hidden DMs for active agents even when shortcut key is not hidden', () => {
    const settings: Settings = {
      ...baseSettings,
      hiddenAgentSidebarKeys: [],
      hiddenAgentIdsForSidebar: [],
      hiddenDmChannelNames: ['dm-u-goexpert'],
    };
    const patch = patchRevealActiveAgentsInSidebar(
      settings,
      [agent('go-2', 'GoExpert', 'active')],
      [dmChannel]
    );
    expect(patch?.hiddenDmChannelNames).toEqual([]);
  });

  it('reveals shortcut and dm for hidden active agent', () => {
    const patch = patchRevealActiveAgentsInSidebar(
      baseSettings,
      [agent('go-2', 'GoExpert', 'active')],
      [dmChannel]
    );
    expect(patch?.hiddenAgentSidebarKeys).toEqual([]);
    expect(patch?.hiddenDmChannelNames).toEqual([]);
  });
});

describe('agentSidebarHideKey', () => {
  it('uses type and name', () => {
    expect(agentSidebarHideKey(agent('1', 'GoExpert', 'active'))).toBe('backend:GoExpert');
  });
});

describe('isDmChannelVisibleInSidebar', () => {
  it('hides DM when matching specialist is absent', () => {
    expect(isDmChannelVisibleInSidebar(dmChannel, [])).toBe(false);
    expect(isDmChannelVisibleInSidebar(dmChannel, [agent('go-2', 'GoExpert', 'active')])).toBe(
      true
    );
  });

  it('hides DM when agent is removed or paused', () => {
    expect(
      isDmChannelVisibleInSidebar(dmChannel, [agent('go-2', 'GoExpert', 'removed')])
    ).toBe(false);
    expect(
      isDmChannelVisibleInSidebar(dmChannel, [
        { ...agent('go-2', 'GoExpert', 'active'), is_paused: true },
      ])
    ).toBe(false);
  });

  it('matches by display name when channel has no agent ids', () => {
    const orphan: Channel = {
      id: '2',
      name: 'dm-u-goexpert',
      type: 'dm',
      description: 'Direct message with GoExpert',
    };
    expect(isDmChannelVisibleInSidebar(orphan, [agent('go-2', 'GoExpert', 'idle')])).toBe(true);
    expect(isDmChannelVisibleInSidebar(orphan, [])).toBe(false);
  });
});
