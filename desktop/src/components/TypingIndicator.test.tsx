import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { TypingIndicator } from './TypingIndicator';
import type { ThinkingAgent } from '../types/protocol';

const agents: ThinkingAgent[] = [
  { id: 'a1', name: 'Cursor', type: 'cli' },
];

const manyAgents: ThinkingAgent[] = [
  { id: 'a1', name: 'BackendEngineer', type: 'backend' },
  { id: 'a2', name: 'FrontendEngineer', type: 'frontend' },
  { id: 'a3', name: 'PlatformEngineer', type: 'platform' },
  { id: 'a4', name: 'SecurityReviewer', type: 'security' },
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

  it('shows tool activity in footer', () => {
    const active: ThinkingAgent[] = [
      {
        id: 'a1',
        name: 'SoftwareArchitect',
        type: 'architecture',
        activity: 'using_tool',
        activityDetail: 'read_file — package.json',
        toolSteps: [{ kind: 'start', name: 'read_file', preview: '[read_file] start' }],
      },
    ];
    render(<TypingIndicator agents={active} />);
    expect(screen.getByText('SoftwareArchitect')).toBeTruthy();
    expect(screen.getByText(/is using read_file/)).toBeTruthy();
    expect(screen.getByText(/▸/)).toBeTruthy();
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

  it('collapses many agents into a summary with expand', () => {
    render(<TypingIndicator agents={manyAgents} />);
    expect(screen.getByText(/BackendEngineer \+ 3 more responding/)).toBeTruthy();
    expect(screen.queryByText('FrontendEngineer')).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /Show all 4 agents responding/ }));
    expect(screen.getByText('FrontendEngineer')).toBeTruthy();
    expect(screen.getByText('SecurityReviewer')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Collapse agent activity list' }));
    expect(screen.queryByText('FrontendEngineer')).toBeNull();
    expect(screen.getByText(/BackendEngineer \+ 3 more responding/)).toBeTruthy();
  });
});
