import { useState, useRef, useEffect, useMemo } from 'react';
import type {
  AssistantTask,
  Channel,
  Collaboration,
  CommandDefinition,
  CommandArgument,
  AgentInfo,
  FileChange,
} from '../types/protocol';
import type { ChatAPI } from '../api/chatAPI';
import {
  COLLAB_SOURCE_MODE_KEY,
  COLLAB_SOURCE_PATH_KEY,
  type CollabSourceMode,
} from '../constants/collabWorkspace';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { isTauriRuntime } from '../utils/promptAttachments';
import { usePacksStore } from '../stores/packsStore';
import { PACK_CAP } from '../stores/packCapabilities';
import { MAX_COLLAB_AGENTS } from '../utils/collaborationLimits';

const CLAUDE_MODELS = ['claude-sonnet', 'claude-haiku'] as const;
const CUSTOM_EXPERT_TYPE = '__custom__';

interface CommandSelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

interface CommandFormProps {
  command: CommandDefinition;
  agents: AgentInfo[];
  channels?: Channel[];
  collaborations?: Collaboration[];
  assistantTasks?: AssistantTask[];
  pendingChanges?: FileChange[];
  api?: ChatAPI;
  onSubmit: (commandString: string, metadata?: Record<string, unknown>) => void;
  onBack: () => void;
}

function shorten(value: string, max = 48): string {
  const trimmed = value.trim();
  if (trimmed.length <= max) return trimmed;
  return `${trimmed.slice(0, max - 1)}…`;
}

function matchesIdPrefix(id: string, value: string): boolean {
  const normalized = value.trim().toLowerCase();
  if (!normalized) return false;
  return id.toLowerCase().startsWith(normalized);
}

function isCollaborationArg(arg: CommandArgument): boolean {
  const name = arg.name.toLowerCase();
  return arg.type === 'collaboration-id' || name === 'collab-id' || name === 'collaboration-id';
}

function isChannelArg(arg: CommandArgument): boolean {
  const name = arg.name.toLowerCase();
  return arg.type === 'channel-name' || name === 'channel' || name === 'channel-name';
}

function isAssistantTaskArg(arg: CommandArgument): boolean {
  const name = arg.name.toLowerCase();
  return arg.type === 'assistant-task-id' || name === 'task-id';
}

function isFileChangeArg(arg: CommandArgument): boolean {
  const name = arg.name.toLowerCase();
  return arg.type === 'file-change-id' || name === 'change-id';
}

export function CommandForm({
  command,
  agents,
  channels = [],
  collaborations = [],
  assistantTasks = [],
  pendingChanges = [],
  api,
  onSubmit,
  onBack,
}: CommandFormProps) {
  const isCollaborateCommand = command.name === '/collaborate';
  const hasLoRACompose = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_COMPOSE));
  const visibleArguments = useMemo(() => {
    if (command.name === '/create-repo-agent' && !hasLoRACompose) {
      return command.arguments.filter((a) => a.name !== 'adapter-repo');
    }
    return command.arguments;
  }, [command.arguments, command.name, hasLoRACompose]);
  const [collabRounds, setCollabRounds] = useState('');
  const [collabMessages, setCollabMessages] = useState('');
  const [allowAgentAdds, setAllowAgentAdds] = useState(false);
  const [collabWorkspaceMode, setCollabWorkspaceMode] = useState<CollabSourceMode>('active');
  const [collabRepoPath, setCollabRepoPath] = useState('');
  const activeExplorerWorkspace = useFileExplorerStore((s) => {
    const id = s.activeWorkspaceId;
    return s.workspaces.find((w) => w.id === id) ?? s.workspaces[0];
  });
  const [values, setValues] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    for (const arg of command.arguments) {
      initial[arg.name] = arg.default ?? '';
    }
    return initial;
  });
  const [selectedCollaborators, setSelectedCollaborators] = useState<Set<string>>(new Set());
  const [modelOptions, setModelOptions] = useState<string[]>([]);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [pathBrowseError, setPathBrowseError] = useState<string | null>(null);
  const [expertTypeMode, setExpertTypeMode] = useState('assistant');
  const [customExpertType, setCustomExpertType] = useState('');
  const [expertPresetSlugs, setExpertPresetSlugs] = useState<string[]>([]);

  const providerArg = useMemo(
    () => command.arguments.find(a => a.type === 'provider'),
    [command.arguments]
  );
  const modelArg = useMemo(
    () => command.arguments.find(a => a.type === 'model'),
    [command.arguments]
  );
  const providerValue =
    (providerArg ? values[providerArg.name] : '') ||
    providerArg?.default ||
    'ollama';

  const firstInputRef = useRef<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | null>(null);

  useEffect(() => {
    firstInputRef.current?.focus();
  }, []);

  useEffect(() => {
    if (command.name !== '/create-expert' || !api) return;
    let cancelled = false;
    (async () => {
      try {
        const presets = await api.fetchExpertPresets();
        if (cancelled) return;
        const slugs = presets.map((p) => p.slug).filter(Boolean);
        setExpertPresetSlugs(slugs.length > 0 ? slugs : ['assistant']);
        setExpertTypeMode(slugs[0] ?? 'assistant');
      } catch {
        if (!cancelled) {
          setExpertPresetSlugs(['assistant']);
          setExpertTypeMode('assistant');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [api, command.name]);

  useEffect(() => {
    if (!api || !modelArg) {
      setModelOptions([]);
      return;
    }

    let cancelled = false;
    setModelsLoading(true);

    (async () => {
      try {
        let models: string[] = [];
        if (providerValue === 'ollama') {
          models = await api.fetchOllamaModels();
        } else if (providerValue === 'lmstudio') {
          models = await api.fetchLMStudioModels();
        } else if (providerValue === 'huggingface' || providerValue === 'hf') {
          const catalog = await api.fetchHfCatalog();
          models = catalog
            .filter(entry => entry.modes?.includes('hosted'))
            .map(entry => entry.repo_id);
        } else if (providerValue === 'claude') {
          models = [...CLAUDE_MODELS];
        }
        if (!cancelled) {
          setModelOptions(models);
        }
      } catch {
        if (!cancelled) {
          setModelOptions([]);
        }
      } finally {
        if (!cancelled) {
          setModelsLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [api, modelArg, providerValue]);

  const setValue = (name: string, value: string) => {
    setValues(prev => ({ ...prev, [name]: value }));
  };

  const updateArgValue = (arg: CommandArgument, value: string) => {
    setValues(prev => {
      const next = { ...prev, [arg.name]: value };
      if (command.name === '/collab-task-done' && isCollaborationArg(arg)) {
        next.task = '';
      }
      return next;
    });
  };

  const handleProviderChange = (argName: string, value: string) => {
    setValue(argName, value);
    if (modelArg) {
      setValue(modelArg.name, '');
    }
  };

  const handleBrowsePath = async (arg: CommandArgument) => {
    setPathBrowseError(null);
    if (!isTauriRuntime()) {
      setPathBrowseError('Folder picker requires the desktop app');
      return;
    }
    try {
      const { open } = await import('@tauri-apps/api/dialog');
      const selected = await open({
        directory: true,
        multiple: false,
        title: arg.description || 'Select directory',
      });
      if (selected && typeof selected === 'string') {
        setValue(arg.name, selected);
      }
    } catch (error) {
      setPathBrowseError(error instanceof Error ? error.message : String(error));
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (isCollaborateCommand) {
      const description = values.description?.trim() || '';
      if (selectedCollaborators.size < 2 || !description) {
        return;
      }
      const mentions = agents
        .filter(agent => selectedCollaborators.has(agent.id))
        .map(agent => `@${agent.name}`);
      const flags: string[] = [];
      const r = collabRounds.trim();
      if (r !== '') {
        if (!/^\d+$/.test(r)) return;
        flags.push('--rounds', r);
      }
      const m = collabMessages.trim();
      if (m !== '') {
        if (!/^\d+$/.test(m)) return;
        flags.push('--messages', m);
      }
      if (allowAgentAdds) {
        flags.push('--allow-agent-adds');
      }
      const meta: Record<string, unknown> = {
        [COLLAB_SOURCE_MODE_KEY]: collabWorkspaceMode,
      };
      if (collabWorkspaceMode === 'none') {
        flags.push('--no-workspace');
      } else if (collabWorkspaceMode === 'path') {
        const p = collabRepoPath.trim();
        if (!p) return;
        flags.push('--repo', p);
        meta[COLLAB_SOURCE_PATH_KEY] = p;
        flags.push('--workspace');
      } else if (collabWorkspaceMode === 'active') {
        flags.push('--workspace');
        if (activeExplorerWorkspace?.path) {
          meta[COLLAB_SOURCE_PATH_KEY] = activeExplorerWorkspace.path;
        }
      }
      onSubmit([command.name, ...flags, ...mentions, description].join(' '), meta);
      return;
    }

    if (command.name === '/create-repo-agent') {
      const repoPath = values['repo-path']?.trim();
      if (!repoPath) return;
      const parts = [command.name, repoPath];
      const agentName = values['agent-name']?.trim();
      const provider = values.provider?.trim();
      const model = values.model?.trim();
      const adapterRepo = values['adapter-repo']?.trim();
      if (agentName) parts.push(agentName);
      if (provider) parts.push(provider);
      if (model) parts.push('--model', model);
      if (adapterRepo) parts.push('--adapter-repo', adapterRepo);
      onSubmit(parts.join(' '));
      return;
    }

    if (command.name === '/create-expert') {
      const expertSlug =
        expertTypeMode === CUSTOM_EXPERT_TYPE
          ? customExpertType.trim()
          : expertTypeMode.trim();
      if (!expertSlug) {
        return;
      }
      const parts = [command.name, expertSlug];
      for (const arg of visibleArguments) {
        if (arg.name === 'type') continue;
        const v = values[arg.name]?.trim();
        if (v) {
          parts.push(v);
        } else if (arg.required) {
          return;
        }
      }
      onSubmit(parts.join(' '));
      return;
    }

    const parts = [command.name];
    for (const arg of visibleArguments) {
      const v = values[arg.name]?.trim();
      if (v) {
        parts.push(v);
      } else if (arg.required) {
        return; // prevent submission with missing required fields
      }
    }

    onSubmit(parts.join(' '));
  };

  const collabNumericOptsOk = (() => {
    const r = collabRounds.trim();
    const m = collabMessages.trim();
    if (r !== '' && !/^\d+$/.test(r)) return false;
    if (m !== '' && !/^\d+$/.test(m)) return false;
    return true;
  })();

  const collabWorkspaceOk =
    collabWorkspaceMode !== 'path' || collabRepoPath.trim().length > 0;

  const createExpertTypeOk =
    command.name !== '/create-expert' ||
    (expertTypeMode !== CUSTOM_EXPERT_TYPE
      ? expertTypeMode.trim().length > 0
      : customExpertType.trim().length > 0);

  const canSubmit = isCollaborateCommand
    ? selectedCollaborators.size >= 2 &&
      selectedCollaborators.size <= MAX_COLLAB_AGENTS &&
      !!values.description?.trim() &&
      collabNumericOptsOk &&
      collabWorkspaceOk
    : command.name === '/create-expert'
      ? createExpertTypeOk &&
        visibleArguments
          .filter((a) => a.required && a.name !== 'type')
          .every((a) => values[a.name]?.trim())
      : visibleArguments
          .filter(a => a.required)
          .every(a => values[a.name]?.trim());

  const toggleCollaborator = (agentID: string) => {
    setSelectedCollaborators(prev => {
      const next = new Set(prev);
      if (next.has(agentID)) {
        next.delete(agentID);
      } else if (next.size < MAX_COLLAB_AGENTS) {
        next.add(agentID);
      }
      return next;
    });
  };

  const selectableCollaborators = agents.filter(
    a => a.status === 'active' && a.type !== 'human' && a.type !== 'moderator'
  );

  const fieldClass =
    'w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text focus:outline-none focus:ring-1 focus:ring-slack-accent';

  const collaborationOptions = useMemo<CommandSelectOption[]>(
    () =>
      collaborations.map((collab) => {
        const shortId = collab.id.slice(0, 8);
        const title = collab.title?.trim() || collab.description?.trim() || 'Untitled collaboration';
        return {
          value: shortId,
          label: `${shorten(title, 44)} (${collab.phase}, ${shortId})`,
        };
      }),
    [collaborations]
  );

  const channelOptions = useMemo<CommandSelectOption[]>(
    () =>
      channels.map((ch) => ({
        value: ch.name,
        label: ch.display_name?.trim()
          ? `${ch.display_name} (${ch.name})`
          : `${ch.name} (${ch.type})`,
      })),
    [channels]
  );

  const assistantTaskOptions = useMemo<CommandSelectOption[]>(
    () =>
      assistantTasks
        .filter(task => task.status !== 'done')
        .map(task => ({
          value: task.id,
          label: `${shorten(task.title, 48)} (${task.status}, ${task.id.slice(0, 8)})`,
        })),
    [assistantTasks]
  );

  const fileChangeOptions = useMemo<CommandSelectOption[]>(
    () =>
      pendingChanges
        .filter(change => change.status === 'pending')
        .map(change => ({
          value: change.id,
          label: `${change.operation} ${shorten(change.file_path || change.new_path || change.old_path || change.id, 42)} (${change.id.slice(0, 8)})`,
        })),
    [pendingChanges]
  );

  const selectedCollaborationForTask = useMemo(() => {
    const selected = values['collab-id'] || values['collaboration-id'] || '';
    if (!selected) return null;
    return collaborations.find(collab => matchesIdPrefix(collab.id, selected)) ?? null;
  }, [collaborations, values]);

  const taskOptionsForSelectedCollaboration = useMemo<CommandSelectOption[]>(
    () =>
      (selectedCollaborationForTask?.tasks ?? [])
        .map((task, index) => ({
          value: String(index + 1),
          label: `#${index + 1}: ${shorten(task.title, 44)} (${task.status})`,
        })),
    [selectedCollaborationForTask]
  );

  const knownSelectConfig = (arg: CommandArgument): { placeholder: string; options: CommandSelectOption[]; disabled?: boolean } | null => {
    if (command.name === '/collab-task-done' && arg.name.toLowerCase() === 'task') {
      return {
        placeholder: selectedCollaborationForTask
          ? 'Select collaboration task...'
          : 'Select collaboration first...',
        options: taskOptionsForSelectedCollaboration,
        disabled: !selectedCollaborationForTask,
      };
    }

    if (isCollaborationArg(arg)) {
      return {
        placeholder: arg.required ? 'Select collaboration...' : 'All collaborations (optional)',
        options: collaborationOptions,
      };
    }

    if (isChannelArg(arg)) {
      return {
        placeholder: 'Select channel...',
        options: channelOptions,
      };
    }

    if (isAssistantTaskArg(arg)) {
      return {
        placeholder: 'Select task...',
        options: assistantTaskOptions,
      };
    }

    if (isFileChangeArg(arg)) {
      return {
        placeholder: 'Select file change...',
        options: fileChangeOptions,
      };
    }

    if (arg.options?.length) {
      return {
        placeholder: arg.required ? `Select ${arg.name}...` : `${arg.name} (optional)`,
        options: arg.options.map(opt => ({ value: opt, label: opt })),
      };
    }

    return null;
  };

  const renderKnownSelect = (
    arg: CommandArgument,
    config: { placeholder: string; options: CommandSelectOption[]; disabled?: boolean },
    refProp: { ref?: React.Ref<any> }
  ) => (
    <select
      id={`cmd-arg-${arg.name}`}
      value={values[arg.name]}
      onChange={e => updateArgValue(arg, e.target.value)}
      disabled={config.disabled || (arg.required && config.options.length === 0)}
      className={fieldClass}
      {...refProp}
    >
      <option value="">{config.placeholder}</option>
      {config.options.map(opt => (
        <option key={opt.value} value={opt.value} disabled={opt.disabled}>
          {opt.label}
        </option>
      ))}
    </select>
  );

  const renderField = (arg: CommandArgument, idx: number) => {
    const refProp: { ref?: React.Ref<any> } =
      idx === 0 ? { ref: firstInputRef as React.Ref<any> } : {};
    const id = `cmd-arg-${arg.name}`;

    switch (arg.type) {
      case 'path':
        return (
          <div>
            <div className="flex gap-2">
              <input
                id={id}
                type="text"
                value={values[arg.name]}
                onChange={e => setValue(arg.name, e.target.value)}
                placeholder={arg.description}
                className={`flex-1 ${fieldClass} placeholder-slack-textMuted`}
                {...refProp}
              />
              <button
                type="button"
                onClick={() => void handleBrowsePath(arg)}
                disabled={!isTauriRuntime()}
                title={
                  isTauriRuntime()
                    ? 'Browse for directory'
                    : 'Folder picker requires the desktop app'
                }
                className="shrink-0 px-3 py-2 text-sm border border-slack-border rounded text-slack-text hover:bg-slack-bgHover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Browse…
              </button>
            </div>
            {pathBrowseError && (
              <p className="mt-1 text-xs text-red-400">{pathBrowseError}</p>
            )}
          </div>
        );

      case 'provider':
        return (
          <select
            id={id}
            value={values[arg.name]}
            onChange={e => handleProviderChange(arg.name, e.target.value)}
            className={fieldClass}
            {...refProp}
          >
            <option value="">Select provider...</option>
            {(arg.options ?? ['ollama', 'claude', 'lmstudio', 'huggingface']).map(opt => (
              <option key={opt} value={opt}>
                {opt}
              </option>
            ))}
          </select>
        );

      case 'model':
        return (
          <select
            id={id}
            value={values[arg.name]}
            onChange={e => setValue(arg.name, e.target.value)}
            disabled={modelsLoading && modelOptions.length === 0}
            className={fieldClass}
            {...refProp}
          >
            <option value="">
              {modelsLoading ? 'Loading models…' : 'Server default (optional)'}
            </option>
            {modelOptions.map(model => (
              <option key={model} value={model}>
                {model}
              </option>
            ))}
          </select>
        );

      case 'agent-name':
      case 'repo-agent-name': {
        const selectable =
          arg.type === 'repo-agent-name'
            ? agents.filter(a => a.type === 'repo')
            : agents.filter(a => a.type !== 'moderator' && a.type !== 'human');
        const placeholder =
          arg.type === 'repo-agent-name' ? 'Select repo agent...' : 'Select agent...';
        return (
          <select
            id={id}
            value={values[arg.name]}
            onChange={e => setValue(arg.name, e.target.value)}
            className={fieldClass}
            {...refProp}
          >
            <option value="">{placeholder}</option>
            {selectable.map(a => (
              <option key={a.id} value={a.name}>
                {a.name}
                {arg.type === 'agent-name' ? ` (${a.type})` : ''}
              </option>
            ))}
          </select>
        );
      }

      default:
        {
          const selectConfig = knownSelectConfig(arg);
          if (selectConfig) {
            return renderKnownSelect(arg, selectConfig, refProp);
          }
        }
        return (
          <input
            id={id}
            type="text"
            value={values[arg.name]}
            onChange={e => updateArgValue(arg, e.target.value)}
            placeholder={arg.description}
            className={`${fieldClass} placeholder-slack-textMuted`}
            {...refProp}
          />
        );
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2 px-4 py-3 border-b border-slack-border">
        <button
          type="button"
          onClick={onBack}
          className="text-slack-textMuted hover:text-slack-text transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <div>
          <div className="text-sm font-semibold text-slack-text font-mono">{command.name}</div>
          <div className="text-xs text-slack-textMuted">{command.description}</div>
        </div>
      </div>

      {/* Fields — scrollable; footer stays pinned below */}
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-3 space-y-4">
        {isCollaborateCommand ? (
          <>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label htmlFor="cmd-collab-rounds" className="block text-xs font-medium text-slack-textMuted mb-1">
                  max discussion rounds <span className="opacity-60">(optional, default 3)</span>
                </label>
                <input
                  id="cmd-collab-rounds"
                  type="number"
                  min={1}
                  max={10}
                  inputMode="numeric"
                  value={collabRounds}
                  onChange={e => setCollabRounds(e.target.value)}
                  placeholder="3"
                  className={`${fieldClass} placeholder-slack-textMuted`}
                />
              </div>
              <div>
                <label htmlFor="cmd-collab-messages" className="block text-xs font-medium text-slack-textMuted mb-1">
                  max agent messages <span className="opacity-60">(optional, default 20)</span>
                </label>
                <input
                  id="cmd-collab-messages"
                  type="number"
                  min={1}
                  max={50}
                  inputMode="numeric"
                  value={collabMessages}
                  onChange={e => setCollabMessages(e.target.value)}
                  placeholder="20"
                  className={`${fieldClass} placeholder-slack-textMuted`}
                />
              </div>
            </div>
            <div>
              <span className="block text-xs font-medium text-slack-textMuted mb-1">
                project workspace
              </span>
              <div className="space-y-2 rounded border border-slack-border bg-slack-bgHover p-2 text-sm">
                <label className="flex items-center gap-2 text-slack-text cursor-pointer">
                  <input
                    type="radio"
                    name="collab-ws"
                    checked={collabWorkspaceMode === 'active'}
                    onChange={() => setCollabWorkspaceMode('active')}
                  />
                  <span>
                    Active workspace
                    {activeExplorerWorkspace?.path ? (
                      <span className="block text-xs text-slack-textMuted truncate">
                        {activeExplorerWorkspace.path}
                      </span>
                    ) : (
                      <span className="block text-xs text-amber-400/90">No workspace open in Files</span>
                    )}
                  </span>
                </label>
                <label className="flex items-center gap-2 text-slack-text cursor-pointer">
                  <input
                    type="radio"
                    name="collab-ws"
                    checked={collabWorkspaceMode === 'path'}
                    onChange={() => setCollabWorkspaceMode('path')}
                  />
                  <span>Choose folder</span>
                </label>
                {collabWorkspaceMode === 'path' && (
                  <div className="flex gap-2 pl-6">
                    <input
                      type="text"
                      value={collabRepoPath}
                      onChange={(e) => setCollabRepoPath(e.target.value)}
                      placeholder="/path/to/repo"
                      className={`${fieldClass} flex-1 placeholder-slack-textMuted`}
                    />
                    <button
                      type="button"
                      onClick={async () => {
                        setPathBrowseError(null);
                        if (!isTauriRuntime()) {
                          setPathBrowseError('Folder picker requires the desktop app');
                          return;
                        }
                        try {
                          const { open } = await import('@tauri-apps/api/dialog');
                          const selected = await open({
                            directory: true,
                            multiple: false,
                            title: 'Collaboration project root',
                          });
                          if (selected && typeof selected === 'string') {
                            setCollabRepoPath(selected);
                          }
                        } catch (error) {
                          setPathBrowseError(
                            error instanceof Error ? error.message : String(error)
                          );
                        }
                      }}
                      className="px-2 py-1 rounded border border-slack-border text-xs text-slack-text hover:bg-white/5"
                    >
                      Browse
                    </button>
                  </div>
                )}
                <label className="flex items-center gap-2 text-slack-text cursor-pointer">
                  <input
                    type="radio"
                    name="collab-ws"
                    checked={collabWorkspaceMode === 'none'}
                    onChange={() => setCollabWorkspaceMode('none')}
                  />
                  <span>
                    No workspace
                    <span className="block text-xs text-slack-textMuted">
                      Research / discussion only — no repo paths
                    </span>
                  </span>
                </label>
              </div>
              {pathBrowseError && collabWorkspaceMode === 'path' && (
                <p className="text-xs text-red-400 mt-1">{pathBrowseError}</p>
              )}
            </div>
            <div>
              <label htmlFor="cmd-arg-description" className="block text-xs font-medium text-slack-textMuted mb-1">
                prompt<span className="text-red-400 ml-0.5">*</span>
              </label>
              <textarea
                id="cmd-arg-description"
                rows={3}
                value={values.description ?? ''}
                onChange={e => setValue('description', e.target.value)}
                placeholder="Describe what you want the agents to collaborate on..."
                className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent resize-y min-h-[5rem]"
                ref={firstInputRef as React.Ref<HTMLTextAreaElement>}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slack-textMuted mb-1">
                agents<span className="text-red-400 ml-0.5">*</span>
                <span className="ml-1 opacity-60">({selectedCollaborators.size} selected, 2–{MAX_COLLAB_AGENTS})</span>
              </label>
              <div className="max-h-28 sm:max-h-40 overflow-y-auto overscroll-contain border border-slack-border rounded bg-slack-bgHover p-1 space-y-0.5">
                {selectableCollaborators.map(agent => {
                  const selected = selectedCollaborators.has(agent.id);
                  return (
                    <button
                      key={agent.id}
                      type="button"
                      onClick={() => toggleCollaborator(agent.id)}
                      className={`w-full text-left px-2 py-1.5 rounded text-sm flex items-center gap-2 transition-colors ${
                        selected
                          ? 'bg-slack-accent/20 text-slack-text'
                          : 'text-slack-textMuted hover:bg-white/5'
                      }`}
                    >
                      <span className="flex-1 truncate">{agent.name}</span>
                      <span className="text-xs opacity-50">{agent.type}</span>
                    </button>
                  );
                })}
                {selectableCollaborators.length === 0 && (
                  <div className="text-xs text-slack-textMuted p-2 text-center">No active agents available</div>
                )}
              </div>
            </div>
            <label className="flex items-start gap-3 rounded border border-slack-border bg-slack-bgHover px-3 py-2 text-sm text-slack-text">
              <input
                type="checkbox"
                checked={allowAgentAdds}
                onChange={e => setAllowAgentAdds(e.target.checked)}
                className="mt-0.5"
              />
              <span>
                <span className="block font-medium">Allow agent expansion requests</span>
                <span className="block text-xs text-slack-textMuted">
                  Agents may suggest adding other agents. You approve each request before anyone joins.
                </span>
              </span>
            </label>
          </>
        ) : (
          visibleArguments.map((arg, idx) => (
            <div key={arg.name}>
              <label htmlFor={`cmd-arg-${arg.name}`} className="block text-xs font-medium text-slack-textMuted mb-1">
                {arg.name}
                {arg.required && <span className="text-red-400 ml-0.5">*</span>}
                {!arg.required && <span className="ml-1 opacity-60">(optional)</span>}
                {arg.type === 'model' && modelsLoading && (
                  <span className="ml-1 opacity-60">(loading…)</span>
                )}
              </label>
              {command.name === '/create-expert' && arg.name === 'type' ? (
                <>
                  <select
                    id={`cmd-arg-${arg.name}`}
                    value={expertTypeMode}
                    onChange={(e) => setExpertTypeMode(e.target.value)}
                    className={fieldClass}
                    ref={idx === 0 ? (firstInputRef as React.Ref<HTMLSelectElement>) : undefined}
                  >
                    {expertPresetSlugs.map((slug) => (
                      <option key={slug} value={slug}>
                        {slug}
                      </option>
                    ))}
                    <option value={CUSTOM_EXPERT_TYPE}>Custom…</option>
                  </select>
                  {expertTypeMode === CUSTOM_EXPERT_TYPE && (
                    <input
                      type="text"
                      value={customExpertType}
                      onChange={(e) => setCustomExpertType(e.target.value)}
                      placeholder="e.g. guitar, legal-advice, cooking"
                      className={`${fieldClass} mt-2 placeholder-slack-textMuted`}
                    />
                  )}
                </>
              ) : (
                renderField(arg, idx)
              )}
            </div>
          ))
        )}
      </div>

      {/* Footer — always visible */}
      <div className="shrink-0 border-t border-slack-border bg-slack-bg px-4 py-3 flex justify-end gap-2">
        <button
          type="button"
          onClick={onBack}
          className="px-3 py-1.5 text-sm text-slack-textMuted hover:text-slack-text rounded transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={!canSubmit}
          className="px-4 py-1.5 text-sm bg-slack-accent hover:bg-slack-accentHover text-white rounded transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          Run Command
        </button>
      </div>
    </form>
  );
}
