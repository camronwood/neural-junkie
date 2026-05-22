import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { TypingIndicator } from './TypingIndicator';
import type { ThinkingAgent } from '../types/protocol';

const agents: ThinkingAgent[] = [
  { id: 'a1', name: 'Cursor', type: 'cli' },
];

afterEach(() => cleanup());

describe('TypingIndicator Stop control', () => {
  it('renders nothing when no agents and showStop is false', () => {
    const { container } = render(<TypingIndicator agents={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it('shows Stop when showStop even without thinking agents', () => {
    const onStop = vi.fn();
    render(<TypingIndicator agents={[]} showStop onStop={onStop} />);
    expect(screen.getByRole('button', { name: 'Stop agents' })).toBeTruthy();
  });

  it('shows thinking agents and Stop; invokes onStop on click', () => {
    const onStop = vi.fn();
    render(<TypingIndicator agents={agents} showStop onStop={onStop} />);
    expect(screen.getByText('Cursor')).toBeTruthy();
    expect(screen.getByText(/is thinking/)).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Stop agents' }));
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it('disables Stop when stopDisabled', () => {
    const onStop = vi.fn();
    render(<TypingIndicator agents={agents} showStop onStop={onStop} stopDisabled />);
    const btn = screen.getByRole('button', { name: 'Stop agents' }) as HTMLButtonElement;
    expect(btn.disabled).toBe(true);
    fireEvent.click(btn);
    expect(onStop).not.toHaveBeenCalled();
  });
});
