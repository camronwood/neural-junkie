import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CommandForm } from './CommandForm';
import type { ChatAPI } from '../api/chatAPI';
import type { AgentInfo, CommandDefinition } from '../types/protocol';

const createRepoAgentCommand: CommandDefinition = {
  name: '/create-repo-agent',
  description: 'Create a new repository expert agent',
  category: 'Repository Agents',
  arguments: [
    { name: 'repo-path', description: 'Path to the repository', type: 'path', required: true },
    { name: 'agent-name', description: 'Custom name for the agent', type: 'string', required: false },
    {
      name: 'provider',
      description: 'AI provider',
      type: 'provider',
      required: false,
      options: ['ollama', 'claude', 'lmstudio', 'huggingface'],
      default: 'ollama',
    },
    { name: 'model', description: 'AI model name', type: 'model', required: false },
  ],
};

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

describe('CommandForm path and model fields', () => {
  it('renders browse button for path arguments', () => {
    render(
      <CommandForm
        command={createRepoAgentCommand}
        agents={agents}
        onSubmit={() => {}}
        onBack={() => {}}
      />
    );

    expect(screen.getByRole('button', { name: /browse/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/repo-path/i)).toBeInTheDocument();
  });

  it('loads model options from the hub API for the selected provider', async () => {
    const api = {
      fetchOllamaModels: vi.fn().mockResolvedValue(['llama3.1', 'qwen2.5-coder:14b']),
      fetchLMStudioModels: vi.fn(),
      fetchHfCatalog: vi.fn(),
    } as unknown as ChatAPI;

    render(
      <CommandForm
        command={createRepoAgentCommand}
        agents={agents}
        api={api}
        onSubmit={() => {}}
        onBack={() => {}}
      />
    );

    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'llama3.1' })).toBeInTheDocument();
    });
    expect(screen.getByRole('option', { name: 'qwen2.5-coder:14b' })).toBeInTheDocument();
    expect(api.fetchOllamaModels).toHaveBeenCalled();
  });

  it('submits create-repo-agent with selected path and model', async () => {
    const onSubmit = vi.fn();
    const api = {
      fetchOllamaModels: vi.fn().mockResolvedValue(['llama3.1']),
      fetchLMStudioModels: vi.fn(),
      fetchHfCatalog: vi.fn(),
    } as unknown as ChatAPI;

    render(
      <CommandForm
        command={createRepoAgentCommand}
        agents={agents}
        api={api}
        onSubmit={onSubmit}
        onBack={() => {}}
      />
    );

    fireEvent.change(screen.getByLabelText(/repo-path/i), {
      target: { value: '/Users/me/projects/my-app' },
    });

    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'llama3.1' })).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText(/^model/i), { target: { value: 'llama3.1' } });
    fireEvent.submit(screen.getByRole('button', { name: 'Run Command' }).closest('form')!);

    expect(onSubmit).toHaveBeenCalledWith(
      '/create-repo-agent /Users/me/projects/my-app ollama llama3.1'
    );
  });

  it('repo-agent-name dropdown lists only repo agents', () => {
    const reindexCommand: CommandDefinition = {
      name: '/reindex-agent',
      description: 'Re-index a repository agent',
      category: 'Repository Agents',
      arguments: [
        {
          name: 'agent-name',
          description: 'Name of the repo agent',
          type: 'repo-agent-name',
          required: true,
        },
      ],
    };

    render(
      <CommandForm
        command={reindexCommand}
        agents={agents}
        onSubmit={() => {}}
        onBack={() => {}}
      />
    );

    expect(screen.getByRole('option', { name: 'Select repo agent...' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /RustExpert/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /Claude/ })).not.toBeInTheDocument();
  });
});
