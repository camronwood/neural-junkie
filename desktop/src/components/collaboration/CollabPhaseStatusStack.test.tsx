import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CollabPhaseStatusStack } from './CollabPhaseStatusStack';

describe('CollabPhaseStatusStack', () => {
  it('renders stall banner when planning is stalled without gen errors', () => {
    render(
      <CollabPhaseStatusStack
        phase="planning"
        collabIdPrefix="abcd1234"
        planningGenErrors={[]}
        planningStalled={['Backend']}
        planningAwaitingFirstTurn={false}
        executingStuck={false}
      />,
    );
    expect(screen.getByTestId('collaboration-planning-stall-banner')).toBeTruthy();
    expect(screen.getByText(/@Backend/)).toBeTruthy();
  });

  it('renders wait banner for first turn', () => {
    render(
      <CollabPhaseStatusStack
        phase="planning"
        collabIdPrefix="abcd1234"
        planningGenErrors={[]}
        planningStalled={[]}
        planningAwaitingFirstTurn
        executingStuck={false}
      />,
    );
    expect(screen.getByTestId('collaboration-planning-wait-banner')).toBeTruthy();
  });

  it('renders executing stuck banner', () => {
    render(
      <CollabPhaseStatusStack
        phase="executing"
        collabIdPrefix="abcd1234"
        planningGenErrors={[]}
        planningStalled={[]}
        planningAwaitingFirstTurn={false}
        executingStuck
      />,
    );
    expect(screen.getByTestId('collaboration-executing-stuck-banner')).toBeTruthy();
  });

  it('renders gen-error banner when planning errors are present', () => {
    render(
      <CollabPhaseStatusStack
        phase="planning"
        collabIdPrefix="abcd1234"
        planningGenErrors={[
          {
            agentId: 'a1',
            agentName: 'Backend',
            errorCode: 'generation_error',
            count: 1,
            turnAdvanced: false,
          },
        ]}
        planningStalled={[]}
        planningAwaitingFirstTurn={false}
        executingStuck={false}
      />,
    );
    expect(screen.getByTestId('collaboration-generation-error-banner')).toBeTruthy();
  });
});
