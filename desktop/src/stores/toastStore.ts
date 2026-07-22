import { createWithEqualityFn as create } from 'zustand/traditional';

export interface Toast {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number; // Auto-dismiss after this many ms (0 = no auto-dismiss)
  variant?: 'default' | 'slack';
  /** How many identical notifications were coalesced into this toast. */
  count?: number;
  action?: {
    label: string;
    onClick: () => void;
  };
}

interface ToastState {
  toasts: Toast[];
  addToast: (toast: Omit<Toast, 'id' | 'count'>) => string;
  removeToast: (id: string) => void;
  clearAllToasts: () => void;
}

const MAX_VISIBLE_TOASTS = 4;
const dismissTimers = new Map<string, ReturnType<typeof setTimeout>>();

function clearDismissTimer(id: string): void {
  const timer = dismissTimers.get(id);
  if (timer) {
    clearTimeout(timer);
    dismissTimers.delete(id);
  }
}

function scheduleDismiss(id: string, duration: number, removeToast: (id: string) => void): void {
  clearDismissTimer(id);
  if (duration <= 0) return;
  dismissTimers.set(
    id,
    setTimeout(() => {
      dismissTimers.delete(id);
      removeToast(id);
    }, duration)
  );
}

function stackKey(toast: Pick<Toast, 'type' | 'title' | 'variant' | 'action'>): string {
  return `${toast.type}|${toast.variant ?? 'default'}|${toast.title}|${toast.action?.label ?? ''}`;
}

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],

  addToast: (toast) => {
    const duration = toast.duration ?? 5000;
    const incomingKey = stackKey(toast);
    const existing = get().toasts.find(
      (t) => !t.action && !toast.action && stackKey(t) === incomingKey
    );

    if (existing) {
      const id = existing.id;
      set((state) => ({
        toasts: state.toasts.map((t) =>
          t.id === id
            ? {
                ...t,
                message: toast.message ?? t.message,
                duration,
                count: (t.count ?? 1) + 1,
              }
            : t
        ),
      }));
      scheduleDismiss(id, duration, get().removeToast);
      return id;
    }

    const id = `toast_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
    const newToast: Toast = {
      ...toast,
      id,
      duration,
      count: 1,
    };

    set((state) => {
      const next = [...state.toasts, newToast];
      if (next.length <= MAX_VISIBLE_TOASTS) return { toasts: next };
      // Keep sticky (duration 0) + newest timed toasts.
      const sticky = next.filter((t) => t.duration === 0);
      const timed = next.filter((t) => t.duration !== 0);
      const keptTimed = timed.slice(-(Math.max(1, MAX_VISIBLE_TOASTS - sticky.length)));
      const dropped = timed.slice(0, Math.max(0, timed.length - keptTimed.length));
      for (const t of dropped) clearDismissTimer(t.id);
      return { toasts: [...sticky, ...keptTimed] };
    });

    scheduleDismiss(id, duration, get().removeToast);
    return id;
  },

  removeToast: (id) => {
    clearDismissTimer(id);
    set((state) => ({
      toasts: state.toasts.filter((toast) => toast.id !== id),
    }));
  },

  clearAllToasts: () => {
    for (const id of [...dismissTimers.keys()]) clearDismissTimer(id);
    set({ toasts: [] });
  },
}));

export const __toastTestUtils = { MAX_VISIBLE_TOASTS, stackKey };
