import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CollabPhaseStatusBanner } from './CollabPhaseStatusBanner';

describe('CollabPhaseStatusBanner', () => {
  it('renders nothing outside planning or without errors', () => {
    const { container } = render(
      <CollabPhaseStatusBanner phase="reviewing" planningGenErrors={[]} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it('renders generation error banner in planning', () => {
    render(
      <CollabPhaseStatusBanner
        phase="planning"
        planningGenErrors={[
          {
            agentId: 'a1',
            agentName: 'Backend',
            errorCode: 'generation_error',
            count: 2,
            turnAdvanced: true,
          },
        ]}
      />,
    );
    expect(screen.getByTestId('collaboration-generation-error-banner')).toBeTruthy();
    expect(screen.getByText(/@Backend/)).toBeTruthy();
    expect(screen.getByText(/turn skipped/)).toBeTruthy();
  });
});
