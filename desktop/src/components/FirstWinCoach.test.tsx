import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { FirstWinCoach } from './FirstWinCoach';
import { FIRST_WIN_DISMISS_KEY } from '../config/firstWinSteps';

afterEach(() => {
  cleanup();
  localStorage.removeItem(FIRST_WIN_DISMISS_KEY);
});

describe('FirstWinCoach', () => {
  it('renders developer checklist and calls palette / files CTAs', () => {
    const onOpenFiles = vi.fn();
    const onOpenCommandPalette = vi.fn();
    render(
      <FirstWinCoach
        track="developer"
        hasWorkspace={false}
        agents={[]}
        hasCollaboration={false}
        packsEnabled={{ ide: true }}
        onOpenFiles={onOpenFiles}
        onOpenCommandPalette={onOpenCommandPalette}
      />,
    );
    expect(screen.getByTestId('first-win-coach')).toBeTruthy();
    expect(screen.getByText('Open a workspace folder')).toBeTruthy();
    fireEvent.click(screen.getByTestId('first-win-primary-cta'));
    expect(onOpenFiles).toHaveBeenCalled();
  });

  it('prefills a repo-expert prompt once a repo agent exists', () => {
    const onPrefillComposer = vi.fn();
    render(
      <FirstWinCoach
        track="developer"
        hasWorkspace={true}
        agents={[{ id: 'r1', name: 'MyRepoExpert', type: 'repo' }]}
        hasCollaboration={false}
        packsEnabled={{ ide: true }}
        onPrefillComposer={onPrefillComposer}
      />,
    );
    fireEvent.click(screen.getByTestId('first-win-primary-cta'));
    expect(onPrefillComposer).toHaveBeenCalledWith(
      '@MyRepoExpert summarize the architecture and top risk areas',
    );
  });

  it('opens a BiologyExpert DM when that specialist is not created yet', () => {
    const onOpenAgentDM = vi.fn();
    const onOpenCommandPalette = vi.fn();
    render(
      <FirstWinCoach
        track="lifeSciences"
        hasWorkspace={false}
        agents={[]}
        hasCollaboration={false}
        packsEnabled={{ 'life-sciences': true }}
        onOpenAgentDM={onOpenAgentDM}
        onOpenCommandPalette={onOpenCommandPalette}
      />,
    );
    fireEvent.click(screen.getByTestId('first-win-primary-cta'));
    expect(onOpenAgentDM).not.toHaveBeenCalled();
    expect(onOpenCommandPalette).toHaveBeenCalledWith('');
  });

  it('prefills a BiologyExpert prompt after the specialist is present', () => {
    const onPrefillComposer = vi.fn();
    render(
      <FirstWinCoach
        track="lifeSciences"
        hasWorkspace={false}
        agents={[{ id: 'bio-1', name: 'BiologyExpert', type: 'biology' }]}
        hasCollaboration={false}
        packsEnabled={{ 'life-sciences': true }}
        onPrefillComposer={onPrefillComposer}
      />,
    );
    fireEvent.click(screen.getByTestId('first-win-primary-cta'));
    expect(onPrefillComposer).toHaveBeenCalledWith(
      '@BiologyExpert what can you help me analyze in this workspace?',
    );
  });

  it('persists dismiss', () => {
    const onDismissed = vi.fn();
    render(
      <FirstWinCoach
        track="general"
        hasWorkspace={false}
        agents={[]}
        hasCollaboration={false}
        onDismissed={onDismissed}
      />,
    );
    fireEvent.click(screen.getByTestId('first-win-dismiss'));
    expect(onDismissed).toHaveBeenCalled();
    expect(localStorage.getItem(FIRST_WIN_DISMISS_KEY)).toBe('1');
  });
});
