import type {
  AgentCapabilityState,
  ResolvedCapability,
} from '../types/protocol';

export function capabilityKey(capability: ResolvedCapability): string {
  return capability.qualified_id || capability.id;
}

export function isSensitiveCapability(capability: ResolvedCapability): boolean {
  return capability.exposure === 'sensitive';
}

export function capabilityOverrideAfterToggle(
  capability: ResolvedCapability,
  state: Pick<AgentCapabilityState, 'allow' | 'deny' | 'effective'>,
): { allow: string[]; deny: string[] } {
  const key = capabilityKey(capability);
  const allow = new Set(state.allow ?? []);
  const deny = new Set(state.deny ?? []);
  const effective = new Set(state.effective ?? []);

  if (isSensitiveCapability(capability)) {
    if (effective.has(key)) {
      allow.delete(key);
      deny.add(key);
    } else {
      deny.delete(key);
      allow.add(key);
    }
  } else if (deny.has(key)) {
    deny.delete(key);
  } else {
    allow.delete(key);
    deny.add(key);
  }

  return {
    allow: [...allow].sort(),
    deny: [...deny].sort(),
  };
}

export type HandoffEvent = 'handoff_started' | 'handoff_completed' | 'handoff_failed';

export function handoffNavigationTarget(metadata?: Record<string, unknown>): string | null {
  const event = metadata?.handoff_event;
  const raw =
    event === 'handoff_started'
      ? metadata?.handoff_channel
      : event === 'handoff_completed' || event === 'handoff_failed'
        ? metadata?.source_channel
        : undefined;
  return typeof raw === 'string' && raw.trim() ? raw.trim() : null;
}
