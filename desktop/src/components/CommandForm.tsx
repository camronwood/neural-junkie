import { useState, useRef, useEffect, useMemo } from 'react';
import type { CommandDefinition, CommandArgument, AgentInfo } from '../types/protocol';
import type { ChatAPI } from '../api/chatAPI';
import { isTauriRuntime } from '../utils/promptAttachments';

const CLAUDE_MODELS = ['claude-sonnet', 'claude-haiku'] as const;

interface CommandFormProps {
  command: CommandDefinition;
  agents: AgentInfo[];
  api?: ChatAPI;
  onSubmit: (commandString: string) => void;
  onBack: () => void;
}

export function CommandForm({ command, agents, api, onSubmit, onBack }: CommandFormProps) {
  const isCollaborateCommand = command.name === '/collaborate';
  const [collabRounds, setCollabRounds] = useState('');
  const [collabMessages, setCollabMessages] = useState('');
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
      onSubmit([command.name, ...flags, ...mentions, description].join(' '));
      return;
    }

    const parts = [command.name];
    for (const arg of command.arguments) {
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

  const canSubmit = isCollaborateCommand
    ? selectedCollaborators.size >= 2 && !!values.description?.trim() && collabNumericOptsOk
    : command.arguments
        .filter(a => a.required)
        .every(a => values[a.name]?.trim());

  const toggleCollaborator = (agentID: string) => {
    setSelectedCollaborators(prev => {
      const next = new Set(prev);
      if (next.has(agentID)) {
        next.delete(agentID);
      } else {
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

  const renderField = (arg: CommandArgument, idx: number) => {
    const refProp = idx === 0 ? { ref: firstInputRef as React.Ref<any> } : {};
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
        return (
          <input
            id={id}
            type="text"
            value={values[arg.name]}
            onChange={e => setValue(arg.name, e.target.value)}
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
                <span className="ml-1 opacity-60">({selectedCollaborators.size} selected, min 2)</span>
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
          </>
        ) : (
          command.arguments.map((arg, idx) => (
            <div key={arg.name}>
              <label htmlFor={`cmd-arg-${arg.name}`} className="block text-xs font-medium text-slack-textMuted mb-1">
                {arg.name}
                {arg.required && <span className="text-red-400 ml-0.5">*</span>}
                {!arg.required && <span className="ml-1 opacity-60">(optional)</span>}
                {arg.type === 'model' && modelsLoading && (
                  <span className="ml-1 opacity-60">(loading…)</span>
                )}
              </label>
              {renderField(arg, idx)}
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
