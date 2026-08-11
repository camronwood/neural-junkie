import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { CommandPalette } from './CommandPalette';
import type { CommandDefinition } from '../types/protocol';

beforeAll(() => {
  Element.prototype.scrollIntoView = vi.fn();
});

const commands: CommandDefinition[] = [
  {
    name: '/create-repo-agent',
    description: 'Create a repo expert from a local path',
    category: 'agents',
    arguments: [],
  },
  {
    name: '/collaborate',
    description: 'Start a bounded multi-agent session',
    category: 'collaboration',
    arguments: [],
  },
  {
    name: '/status',
    description: 'Show hub status',
    category: 'system',
    arguments: [],
  },
];

afterEach(() => cleanup());

describe('CommandPalette first-win filters', () => {
  it('applies create-repo-agent initialFilter', () => {
    render(
      <CommandPalette
        commands={commands}
        agents={[]}
        isOpen
        initialFilter="create-repo-agent"
        onClose={() => {}}
        onExecute={vi.fn()}
      />,
    );
    expect((screen.getByPlaceholderText('Search actions and commands...') as HTMLInputElement).value).toBe(
      'create-repo-agent',
    );
    expect(screen.getByText('/create-repo-agent')).toBeTruthy();
    expect(screen.queryByText('/status')).toBeNull();
  });

  it('applies collaborate initialFilter', () => {
    render(
      <CommandPalette
        commands={commands}
        agents={[]}
        isOpen
        initialFilter="collaborate"
        onClose={() => {}}
        onExecute={vi.fn()}
      />,
    );
    expect((screen.getByPlaceholderText('Search actions and commands...') as HTMLInputElement).value).toBe(
      'collaborate',
    );
    expect(screen.getByText('/collaborate')).toBeTruthy();
    expect(screen.queryByText('/create-repo-agent')).toBeNull();
  });
});
