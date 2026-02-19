import { useState, useRef, useEffect } from 'react';
import type { CommandDefinition, CommandArgument, AgentInfo } from '../types/protocol';

interface CommandFormProps {
  command: CommandDefinition;
  agents: AgentInfo[];
  onSubmit: (commandString: string) => void;
  onBack: () => void;
}

export function CommandForm({ command, agents, onSubmit, onBack }: CommandFormProps) {
  const [values, setValues] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    for (const arg of command.arguments) {
      initial[arg.name] = arg.default ?? '';
    }
    return initial;
  });

  const firstInputRef = useRef<HTMLInputElement | HTMLSelectElement | null>(null);

  useEffect(() => {
    firstInputRef.current?.focus();
  }, []);

  const setValue = (name: string, value: string) => {
    setValues(prev => ({ ...prev, [name]: value }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

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

  const canSubmit = command.arguments
    .filter(a => a.required)
    .every(a => values[a.name]?.trim());

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
    <form onSubmit={handleSubmit} className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-slack-border">
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

      {/* Fields */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
        {command.arguments.map((arg, idx) => (
          <div key={arg.name}>
            <label htmlFor={`cmd-arg-${arg.name}`} className="block text-xs font-medium text-slack-textMuted mb-1">
              {arg.name}
              {arg.required && <span className="text-red-400 ml-0.5">*</span>}
              {!arg.required && <span className="ml-1 opacity-60">(optional)</span>}
            </label>
            {renderField(arg, idx)}
          </div>
        ))}
      </div>

      {/* Footer */}
      <div className="px-4 py-3 border-t border-slack-border flex justify-end gap-2">
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
