import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { TurnTelemetryDrawer } from './TurnTelemetryDrawer';
import { useChatStore } from '../stores/chatStore';

afterEach(() => {
  cleanup();
  useChatStore.setState({ turnTelemetryByChannel: new Map() });
});

describe('TurnTelemetryDrawer', () => {
  it('renders structured routing row from payload', () => {
    useChatStore.getState().appendTurnTelemetryEvent('general', {
      agentId: 'a1',
      agentName: 'Dev',
      kind: 'routing',
      detail: 'chat: qwen3.5:9b',
      payload: {
        chat_model: 'qwen3.5:9b',
        cost_tier: 'standard',
        knowledge_route: 'codebase',
        reason: 'capability_routing',
        source: 'capabilities',
        governance: { composer_mode: 'agent' },
      },
    });

    render(<TurnTelemetryDrawer channel="general" enabled />);
    expect(screen.getByTestId('turn-telemetry-drawer')).toBeTruthy();
    expect(screen.getByTestId('turn-telemetry-routing-row')).toBeTruthy();
    expect(screen.getByText(/qwen3\.5:9b · Standard · Codebase/)).toBeTruthy();
    expect(screen.getByText(/capability_routing · capabilities · agent/)).toBeTruthy();
  });

  it('returns null when disabled', () => {
    const { container } = render(<TurnTelemetryDrawer channel="general" enabled={false} />);
    expect(container.firstChild).toBeNull();
  });
});
