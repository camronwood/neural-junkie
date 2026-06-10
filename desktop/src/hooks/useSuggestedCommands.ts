import { useCallback, type RefObject } from 'react';
import type { Collaboration, Message } from '../types/protocol';
import type { CommandSuggestion } from '../stores/terminalStore';
import { useTerminalStore } from '../stores/terminalStore';
import { resolveTerminalCwd } from '../utils/terminalCwd';

export interface SuggestedCommandToast {
  type: 'warning';
  title: string;
  message: string;
  duration: number;
  action: {
    label: string;
    onClick: () => void;
  };
}

interface UseSuggestedCommandsOptions {
  collaborationsByIDRef: RefObject<Record<string, Collaboration>>;
  addToast: (toast: SuggestedCommandToast) => void;
}

/** Queue agent command suggestions for user review — never auto-execute. */
export function useSuggestedCommands({
  collaborationsByIDRef,
  addToast,
}: UseSuggestedCommandsOptions) {
  const addSuggestedCommand = useTerminalStore((s) => s.addSuggestedCommand);

  const handleSuggestedCommands = useCallback(
    (message: Message, activeChannel: string) => {
      if (!message.metadata?.suggested_commands) return;

      const suggestions = message.metadata.suggested_commands as CommandSuggestion[];
      const msgCh = message.channel || activeChannel;
      const collabCtx = Object.values(collaborationsByIDRef.current ?? {}).find(
        (c) => c.channel === msgCh
      );

      for (const suggestion of suggestions) {
        const enriched: CommandSuggestion = {
          ...suggestion,
          cwd:
            suggestion.cwd?.trim() ||
            resolveTerminalCwd({ collaboration: collabCtx ?? null }),
        };
        addSuggestedCommand(enriched);
        useTerminalStore.getState().setPanelOpen(true);
        addToast({
          type: 'warning',
          title: `${enriched.agent_name} wants to run a command`,
          message: enriched.command,
          duration: 0,
          action: {
            label: 'Review in terminal',
            onClick: () => useTerminalStore.getState().setPanelOpen(true),
          },
        });
      }
    },
    [addSuggestedCommand, addToast, collaborationsByIDRef]
  );

  return { handleSuggestedCommands };
}
