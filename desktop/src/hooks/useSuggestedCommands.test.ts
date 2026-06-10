import { describe, expect, it, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { Message } from '../types/protocol';
import { useTerminalStore } from '../stores/terminalStore';
import { useSuggestedCommands } from './useSuggestedCommands';

const runAgentTerminalCommand = vi.fn();
vi.mock('../utils/runTerminalCommand', () => ({
  runAgentTerminalCommand: (...args: unknown[]) => runAgentTerminalCommand(...args),
}));
vi.mock('../utils/terminalCwd', () => ({
  resolveTerminalCwd: () => '/workspace',
}));

describe('useSuggestedCommands', () => {
  beforeEach(() => {
    runAgentTerminalCommand.mockReset();
    useTerminalStore.setState({
      suggestedCommands: [],
      isPanelOpen: false,
    });
  });

  it('queues suggested commands and never auto-runs them', () => {
    const addToast = vi.fn();
    const collaborationsByIDRef = { current: {} as Record<string, never> };

    const { result } = renderHook(() =>
      useSuggestedCommands({ collaborationsByIDRef, addToast })
    );

    const message: Message = {
      id: 'm1',
      type: 'chat',
      channel: 'general',
      content: 'run tests',
      timestamp: new Date().toISOString(),
      from: { id: 'a1', name: 'DevAgent', type: 'agent' },
      metadata: {
        suggested_commands: [
          {
            id: 's1',
            command: 'npm test',
            plugin: 'workspace',
            description: 'Run tests',
            is_safe: true,
            agent_name: 'DevAgent',
            message_id: 'm1',
            created_at: new Date().toISOString(),
          },
        ],
      },
    };

    act(() => {
      result.current.handleSuggestedCommands(message, 'general');
    });

    expect(useTerminalStore.getState().suggestedCommands).toHaveLength(1);
    expect(useTerminalStore.getState().suggestedCommands[0]?.command).toBe('npm test');
    expect(useTerminalStore.getState().isPanelOpen).toBe(true);
    expect(addToast).toHaveBeenCalledTimes(1);
    expect(runAgentTerminalCommand).not.toHaveBeenCalled();
  });
});
