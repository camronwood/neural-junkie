import { useState } from 'react';
import type { AgentInfo, Channel } from '../types/protocol';
import { getAgentColor } from '../types/protocol';

interface ChannelInfoModalProps {
  channel: Channel;
  agents: AgentInfo[];
  onClose: () => void;
  onClearHistory?: (channelName: string) => Promise<void>;
}

function typeLabel(t: Channel['type'] | undefined): string {
  switch (t) {
    case 'dm':
      return 'Direct message';
    case 'custom':
      return 'Custom';
    case 'collaboration':
      return 'Collaboration';
    case 'public':
      return 'Public';
    default:
      return t || 'Public';
  }
}

function formatWhen(iso: string | undefined): string {
  if (!iso) return '—';
  const d = Date.parse(iso);
  if (Number.isNaN(d)) return iso;
  return new Date(d).toLocaleString();
}

export function ChannelInfoModal({ channel: ch, agents: globalAgents, onClose, onClearHistory }: ChannelInfoModalProps) {
  const [clearing, setClearing] = useState(false);
  const [clearError, setClearError] = useState<string | null>(null);

  const inRoom = new Map<string, AgentInfo>();
  for (const a of ch.agents ?? []) {
    inRoom.set(a.id, a);
  }

  const memberOnlyRows: { id: string; agent?: AgentInfo }[] = [];
  const seen = new Set(inRoom.keys());
  for (const id of ch.members ?? []) {
    if (seen.has(id)) continue;
    seen.add(id);
    memberOnlyRows.push({ id, agent: globalAgents.find((a) => a.id === id) });
  }

  const canClearHistory =
    !!onClearHistory && (ch.type === 'dm' || ch.type === 'custom' || ch.type === 'collaboration');

  const handleClearHistory = async () => {
    if (!onClearHistory || clearing) return;
    const label = ch.type === 'dm' ? 'this DM' : `#${ch.name}`;
    if (
      !window.confirm(
        `Clear all message history for ${label}? This cannot be undone. Agents will not replay old messages after hub restart.`
      )
    ) {
      return;
    }
    setClearing(true);
    setClearError(null);
    try {
      await onClearHistory(ch.name);
      onClose();
    } catch (e) {
      setClearError(e instanceof Error ? e.message : String(e));
    } finally {
      setClearing(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-slack-bg border border-slack-border rounded-lg shadow-2xl w-full max-w-md mx-4 max-h-[85vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-5 py-4 border-b border-slack-border shrink-0">
          <h2 className="text-lg font-bold text-slack-text">#{ch.name}</h2>
          <p className="text-xs text-slack-textMuted mt-1">{typeLabel(ch.type)}</p>
        </div>

        <div className="px-5 py-4 space-y-4 overflow-y-auto text-sm text-slack-text">
          {ch.description ? (
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wide text-slack-textMuted mb-1">
                Description
              </div>
              <p className="text-slack-text whitespace-pre-wrap">{ch.description}</p>
            </div>
          ) : null}

          {ch.project ? (
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wide text-slack-textMuted mb-1">
                Project
              </div>
              <p>{ch.project}</p>
            </div>
          ) : null}

          <div className="grid grid-cols-2 gap-3 text-xs">
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wide text-slack-textMuted mb-1">
                Created
              </div>
              <p>{formatWhen(ch.created)}</p>
            </div>
            {ch.created_by ? (
              <div>
                <div className="text-[11px] font-semibold uppercase tracking-wide text-slack-textMuted mb-1">
                  Created by
                </div>
                <p>{ch.created_by}</p>
              </div>
            ) : null}
          </div>

          <div>
            <div className="text-[11px] font-semibold uppercase tracking-wide text-slack-textMuted mb-2">
              In this channel ({inRoom.size + memberOnlyRows.length})
            </div>
            <ul className="space-y-2">
              {Array.from(inRoom.values()).map((a) => (
                <li key={a.id} className="flex items-center gap-2 min-w-0">
                  <span
                    className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                    style={{ backgroundColor: getAgentColor(a.type) }}
                  />
                  <span className="truncate font-medium">{a.name}</span>
                  <span className="text-xs text-slack-textMuted shrink-0">{a.type}</span>
                  <span className="text-xs text-slack-textMuted shrink-0 ml-auto">{a.status}</span>
                </li>
              ))}
              {memberOnlyRows.map(({ id, agent }) => (
                <li key={id} className="flex items-center gap-2 min-w-0 text-slack-textMuted">
                  <span className="w-2.5 h-2.5 rounded-full flex-shrink-0 bg-white/30" />
                  {agent ? (
                    <>
                      <span className="truncate font-medium text-slack-text">{agent.name}</span>
                      <span className="text-xs shrink-0">{agent.type}</span>
                      <span className="text-xs shrink-0 ml-auto">member</span>
                    </>
                  ) : (
                    <span className="truncate">Member ID: {id}</span>
                  )}
                </li>
              ))}
              {inRoom.size === 0 && memberOnlyRows.length === 0 && (
                <li className="text-slack-textMuted text-xs">No agents listed for this channel yet.</li>
              )}
            </ul>
          </div>

          {ch.tags && ch.tags.length > 0 ? (
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wide text-slack-textMuted mb-1">
                Tags
              </div>
              <p className="text-xs">{ch.tags.join(', ')}</p>
            </div>
          ) : null}

          {canClearHistory ? (
            <div className="pt-2 border-t border-slack-border">
              <div className="text-[11px] font-semibold uppercase tracking-wide text-slack-textMuted mb-2">
                History
              </div>
              <p className="text-xs text-slack-textMuted mb-2">
                Remove all messages in this channel so agents stop replaying stale errors on restore.
              </p>
              {clearError ? <p className="text-xs text-red-400 mb-2">{clearError}</p> : null}
              <button
                type="button"
                disabled={clearing}
                onClick={() => void handleClearHistory()}
                className="px-3 py-1.5 text-sm border border-red-500/50 text-red-300 hover:bg-red-500/10 rounded font-medium transition-colors disabled:opacity-50"
              >
                {clearing ? 'Clearing…' : 'Clear message history'}
              </button>
            </div>
          ) : null}
        </div>

        <div className="px-5 py-3 border-t border-slack-border flex justify-end shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-1.5 text-sm bg-slack-accent hover:bg-slack-accentHover text-white rounded font-medium transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
