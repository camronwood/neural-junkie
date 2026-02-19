import { useState } from 'react';
import type { AgentInfo } from '../types/protocol';
import { getAgentColor } from '../types/protocol';

interface CreateChannelModalProps {
  agents: AgentInfo[];
  isOpen: boolean;
  onClose: () => void;
  onCreate: (name: string, description: string, agentIds: string[]) => void;
}

export function CreateChannelModal({ agents, isOpen, onClose, onCreate }: CreateChannelModalProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selectedAgents, setSelectedAgents] = useState<Set<string>>(new Set());

  if (!isOpen) return null;

  const toggleAgent = (agentId: string) => {
    setSelectedAgents(prev => {
      const next = new Set(prev);
      if (next.has(agentId)) {
        next.delete(agentId);
      } else {
        next.add(agentId);
      }
      return next;
    });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const slug = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
    if (!slug) return;
    onCreate(slug, description, Array.from(selectedAgents));
    setName('');
    setDescription('');
    setSelectedAgents(new Set());
    onClose();
  };

  const activeAgents = agents.filter(a => a.status === 'active');

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-md mx-4"
        onClick={e => e.stopPropagation()}
      >
        <form onSubmit={handleSubmit}>
          <div className="px-5 py-4 border-b border-slack-border">
            <h2 className="text-lg font-bold text-slack-text">Create a Channel</h2>
            <p className="text-xs text-slack-textMuted mt-1">
              Add agents to collaborate on a specific topic.
            </p>
          </div>

          <div className="px-5 py-4 space-y-4">
            <div>
              <label className="block text-xs font-medium text-slack-textMuted mb-1">
                Channel Name
              </label>
              <input
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                placeholder="e.g. rust-backend"
                className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
                autoFocus
                required
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slack-textMuted mb-1">
                Description (optional)
              </label>
              <input
                type="text"
                value={description}
                onChange={e => setDescription(e.target.value)}
                placeholder="What's this channel about?"
                className="w-full px-3 py-2 bg-slack-bgHover border border-slack-border rounded text-sm text-slack-text placeholder-slack-textMuted focus:outline-none focus:ring-1 focus:ring-slack-accent"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-slack-textMuted mb-1">
                Add Agents ({selectedAgents.size} selected)
              </label>
              <div className="max-h-48 overflow-y-auto border border-slack-border rounded bg-slack-bgHover p-1 space-y-0.5">
                {activeAgents.map(agent => {
                  const selected = selectedAgents.has(agent.id);
                  return (
                    <button
                      key={agent.id}
                      type="button"
                      onClick={() => toggleAgent(agent.id)}
                      className={`w-full text-left px-2 py-1.5 rounded text-sm flex items-center gap-2 transition-colors ${
                        selected
                          ? 'bg-slack-accent/20 text-slack-text'
                          : 'text-slack-textMuted hover:bg-white/5'
                      }`}
                    >
                      <span
                        className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                        style={{ backgroundColor: getAgentColor(agent.type) }}
                      />
                      <span className="flex-1 truncate">{agent.name}</span>
                      <span className="text-[10px] opacity-50">{agent.type}</span>
                      {selected && (
                        <svg className="w-3.5 h-3.5 text-slack-accent flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                        </svg>
                      )}
                    </button>
                  );
                })}
                {activeAgents.length === 0 && (
                  <p className="text-xs text-slack-textMuted p-2 text-center">No active agents</p>
                )}
              </div>
            </div>
          </div>

          <div className="px-5 py-3 border-t border-slack-border flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-1.5 text-sm text-slack-textMuted hover:text-slack-text rounded transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!name.trim()}
              className="px-4 py-1.5 text-sm bg-slack-accent hover:bg-slack-accentHover text-white rounded font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Create Channel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
