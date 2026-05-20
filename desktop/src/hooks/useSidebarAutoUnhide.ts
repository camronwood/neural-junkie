import { useEffect, useRef } from 'react';
import type { AgentInfo, Channel } from '../types/protocol';
import { useSettingsStore } from '../stores/settingsStore';
import {
  patchMigrateHiddenAgentSidebarKeys,
  patchUnhideAgentShortcutsOnActivation,
} from '../utils/sidebarVisibility';

/**
 * Restores hidden agent shortcuts (and matching DMs) when agents become active again.
 * Does not run on first load so "hidden from sidebar" survives app restarts.
 */
export function useSidebarAutoUnhide(agents: AgentInfo[], channels: Channel[]) {
  const settings = useSettingsStore((s) => s.settings);
  const isLoaded = useSettingsStore((s) => s.isLoaded);
  const updateSettings = useSettingsStore((s) => s.updateSettings);
  const prevStatusRef = useRef<Record<string, string | undefined>>({});
  const seededRef = useRef(false);

  useEffect(() => {
    if (!isLoaded) return;

    const migratePatch = patchMigrateHiddenAgentSidebarKeys(settings, agents);
    if (migratePatch) {
      void updateSettings(migratePatch);
      return;
    }

    const { patch, nextStatusById } = patchUnhideAgentShortcutsOnActivation(
      settings,
      agents,
      channels,
      prevStatusRef.current,
      { seeded: seededRef.current }
    );
    prevStatusRef.current = nextStatusById;
    seededRef.current = true;

    if (patch) {
      void updateSettings(patch);
    }
  }, [agents, channels, settings, isLoaded, updateSettings]);
}
