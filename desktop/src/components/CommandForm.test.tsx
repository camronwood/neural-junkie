import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CommandForm } from './CommandForm';
import type { AgentInfo, CommandDefinition } from '../types/protocol';

const collaborateCommand: CommandDefinition = {
  name: '/collaborate',
  description: 'Start a multi-agent collaboration',
  category: 'Collaboration',
  arguments: [
    {
      name: 'description',
      description: '@Agent1 @Agent2 collaboration goal',
      type: 'string',
      required: true,
    },
  ],
};

const agents: AgentInfo[] = [
  {
    id: 'a1',
    name: 'RustExpert',
    type: 'rust',
    expertise: ['rust'],
    status: 'active',
    model: 'mock',
    is_paused: false,
  },
  {
    id: 'a2',
    name: 'ReactExpert',
    type: 'frontend',
    expertise: ['react'],
    status: 'active',
    model: 'mock',
    is_paused: false,
  },
  {
    id: 'a3',
    name: 'DevOpsPro',
    type: 'devops',
    expertise: ['k8s'],
    status: 'active',
    model: 'mock',
    is_paused: false,
  },
];

afterEach(() => {
  cleanup();
});

describe('CommandForm /collaborate quick select', () => {
  it('builds collaborate command from selected agents and prompt', () => {
    const onSubmit = vi.fn();

    render(
      <CommandForm
        command={collaborateCommand}
        agents={agents}
        onSubmit={onSubmit}
        onBack={() => {}}
      />
    );

    fireEvent.change(screen.getByLabelText(/prompt/i), {
      target: { value: 'Build and deploy collaboration UI updates' },
    });
    fireEvent.click(screen.getByText('RustExpert'));
    fireEvent.click(screen.getByText('ReactExpert'));
    fireEvent.submit(screen.getByRole('button', { name: 'Run Command' }).closest('form')!);

    expect(onSubmit).toHaveBeenCalledWith(
      '/collaborate @RustExpert @ReactExpert Build and deploy collaboration UI updates'
    );
  });

  it('includes optional --rounds and --messages when set', () => {
    const onSubmit = vi.fn();

    render(
      <CommandForm
        command={collaborateCommand}
        agents={agents}
        onSubmit={onSubmit}
        onBack={() => {}}
      />
    );

    fireEvent.change(screen.getByLabelText(/prompt/i), {
      target: { value: 'Ship the feature' },
    });
    fireEvent.change(screen.getByPlaceholderText('3'), { target: { value: '6' } });
    fireEvent.change(screen.getByPlaceholderText('20'), { target: { value: '35' } });
    fireEvent.click(screen.getByText('RustExpert'));
    fireEvent.click(screen.getByText('ReactExpert'));
    fireEvent.submit(screen.getByRole('button', { name: 'Run Command' }).closest('form')!);

    expect(onSubmit).toHaveBeenCalledWith(
      '/collaborate --rounds 6 --messages 35 @RustExpert @ReactExpert Ship the feature'
    );
  });
});
