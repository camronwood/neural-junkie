import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useToastStore } from './toastStore';

describe('toastStore stacking', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    useToastStore.getState().clearAllToasts();
  });

  afterEach(() => {
    useToastStore.getState().clearAllToasts();
    vi.useRealTimers();
  });

  it('coalesces identical toasts into one with a count', () => {
    const store = useToastStore.getState();
    store.addToast({ type: 'info', title: 'Slack message', message: 'one' });
    store.addToast({ type: 'info', title: 'Slack message', message: 'two' });
    store.addToast({ type: 'info', title: 'Slack message', message: 'three' });
    const toasts = useToastStore.getState().toasts;
    expect(toasts).toHaveLength(1);
    expect(toasts[0].count).toBe(3);
    expect(toasts[0].message).toBe('three');
  });

  it('keeps distinct titles separate and caps visible toasts', () => {
    const store = useToastStore.getState();
    for (let i = 0; i < 6; i++) {
      store.addToast({ type: 'info', title: `Notice ${i}`, message: String(i) });
    }
    expect(useToastStore.getState().toasts.length).toBeLessThanOrEqual(4);
  });

  it('does not coalesce actionable toasts', () => {
    const store = useToastStore.getState();
    store.addToast({
      type: 'info',
      title: 'Open thread',
      action: { label: 'View', onClick: () => undefined },
    });
    store.addToast({
      type: 'info',
      title: 'Open thread',
      action: { label: 'View', onClick: () => undefined },
    });
    expect(useToastStore.getState().toasts).toHaveLength(2);
  });
});
