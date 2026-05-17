import { useState, useRef, useEffect } from 'react';
import type { CommandDefinition, CommandArgument, AgentInfo } from '../types/protocol';

interface CommandFormProps {
  command: CommandDefinition;
  agents: AgentInfo[];
  onSubmit: (commandString: string) => void;
  onBack: () => void;
}

export function CommandForm({ command, agents, onSubmit, onBack }: CommandFormProps) {
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

  const firstInputRef = useRef<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | null>(null);

  useEffect(() => {
    firstInputRef.current?.focus();
  }, []);

  const setValue = (name: string, value: string) => {
    setValues(prev => ({ ...prev, [name]: value }));
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

  const renderField = (arg: CommandArgument, idx: number) => {
    const refProp = idx === 0 ? { ref: firstInputRef as React.Ref<any> } : {};
    const id = `cmd-arg-${arg.name}`;

    switch (arg.type) {
      case 'provider':
        return (
          <select
            id={id}
            value={values[arg.name]}
            onChange={e => setValue(arg.name, e.target.value)}
            className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text focus:outline-none focus:ring-1 focus:ring-slack-accent"
            {...refProp}
          >
            <option value="">Select provider...</option>
            {(arg.options ?? ['ollama', 'claude', 'lmstudio']).map(opt => (
              <option key={opt} value={opt}>{opt}</option>
            ))}
          </select>
        );

      case 'agent-name':
        return (
          <select
            id={id}
            value={values[arg.name]}
            onChange={e => setValue(arg.name, e.target.value)}
            className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text focus:outline-none focus:ring-1 focus:ring-slack-accent"
            {...refProp}
          >
            <option value="">Select agent...</option>
            {agents
              .filter(a => a.type !== 'moderator' && a.type !== 'human')
              .map(a => (
                <option key={a.id} value={a.name}>{a.name} ({a.type})</option>
              ))}
          </select>
        );

      default:
        return (
          <input
            id={id}
            type="text"
            value={values[arg.name]}
            onChange={e => setValue(arg.name, e.target.value)}
            placeholder={arg.description}
            className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
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
                  className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
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
                  className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
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
