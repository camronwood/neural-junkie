import { createWithEqualityFn as create } from 'zustand/traditional';

export type ActivityKind =
  | 'channel'
  | 'file'
  | 'terminal'
  | 'command'
  | 'settings'
  | 'agent'
  | 'other';

export interface ActivityEvent {
  id: string;
  ts: number;
  kind: ActivityKind;
  title: string;
  detail?: string;
  channel?: string;
  path?: string;
}

const STORAGE_KEY = 'nj-activity-log-v1';
const MAX_EVENTS = 500;

function loadEvents(): ActivityEvent[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as ActivityEvent[];
    return Array.isArray(parsed) ? parsed.slice(0, MAX_EVENTS) : [];
  } catch {
    return [];
  }
}

function persist(events: ActivityEvent[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(events.slice(0, MAX_EVENTS)));
  } catch {
    /* ignore quota */
  }
}

interface ActivityLogState {
  events: ActivityEvent[];
  append: (partial: Omit<ActivityEvent, 'id' | 'ts'> & { ts?: number }) => void;
  clear: () => void;
}

export const useActivityLogStore = create<ActivityLogState>((set, get) => ({
  events: typeof localStorage !== 'undefined' ? loadEvents() : [],
  append: (partial) => {
    const event: ActivityEvent = {
      id: `act_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
      ts: partial.ts ?? Date.now(),
      kind: partial.kind,
      title: partial.title,
      detail: partial.detail,
      channel: partial.channel,
      path: partial.path,
    };
    const next = [event, ...get().events].slice(0, MAX_EVENTS);
    persist(next);
    set({ events: next });
  },
  clear: () => {
    persist([]);
    set({ events: [] });
  },
}));

export function logActivity(partial: Omit<ActivityEvent, 'id' | 'ts'> & { ts?: number }): void {
  useActivityLogStore.getState().append(partial);
}
