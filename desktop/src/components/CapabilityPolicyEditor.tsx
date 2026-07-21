import type {
  AgentCapabilityState,
  ResolvedCapability,
} from '../types/protocol';
import {
  capabilityKey,
  capabilityOverrideAfterToggle,
  isSensitiveCapability,
} from '../utils/capabilityPolicy';

interface CapabilityPolicyEditorProps {
  capabilities: ResolvedCapability[];
  state: AgentCapabilityState;
  onChange: (override: { allow: string[]; deny: string[] }) => void;
  disabled?: boolean;
  compact?: boolean;
}

export function CapabilityPolicyEditor({
  capabilities,
  state,
  onChange,
  disabled = false,
  compact = false,
}: CapabilityPolicyEditorProps) {
  const available = new Set(state.available ?? []);
  const effective = new Set(state.effective ?? []);
  const denied = new Set(state.deny ?? []);
  const unavailable = new Set(state.unavailable ?? []);

  if (!capabilities.length) {
    return <p className="text-sm text-slack-textMuted">No executable capabilities are discoverable.</p>;
  }

  return (
    <div className={compact ? 'space-y-1.5' : 'space-y-2'}>
      {capabilities.map((capability) => {
        const key = capabilityKey(capability);
        const sensitive = isSensitiveCapability(capability);
        const isAvailable = available.has(key) && !unavailable.has(key);
        const isEffective = effective.has(key);
        const explicitlyDenied = denied.has(key);
        const status = !isAvailable
          ? 'Unavailable'
          : sensitive
            ? isEffective
              ? 'Granted'
              : 'Revoked'
            : explicitlyDenied
              ? 'Revoked'
              : 'Inherited';

        return (
          <div
            key={key}
            className="flex items-start justify-between gap-3 rounded border border-slack-border bg-slack-bgHover px-3 py-2"
          >
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium text-slack-text">
                  {capability.label || capability.id}
                </span>
                <span
                  className={`rounded px-1.5 py-0.5 text-[10px] uppercase ${
                    sensitive
                      ? 'bg-amber-500/15 text-amber-300'
                      : 'bg-emerald-500/15 text-emerald-300'
                  }`}
                >
                  {sensitive ? 'sensitive' : 'safe'}
                </span>
                <span className="text-xs text-slack-textMuted">{status}</span>
              </div>
              {capability.description && (
                <p className="mt-0.5 text-xs text-slack-textMuted">{capability.description}</p>
              )}
              {!isAvailable && (
                <p className="mt-1 text-xs text-amber-400">
                  Required pack, MCP agent, or tool is not currently available.
                </p>
              )}
            </div>
            <button
              type="button"
              disabled={disabled || !isAvailable}
              onClick={() => onChange(capabilityOverrideAfterToggle(capability, state))}
              className="shrink-0 rounded border border-slack-border px-2 py-1 text-xs text-slack-text hover:bg-slack-bg disabled:cursor-not-allowed disabled:opacity-40"
            >
              {sensitive ? (isEffective ? 'Revoke' : 'Grant') : explicitlyDenied ? 'Inherit' : 'Revoke'}
            </button>
          </div>
        );
      })}
    </div>
  );
}
