import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  confirmReplaceCollaborationExecution,
  confirmStartCollaborationWhileExecuting,
} from './collaborationConfirm';
import type { Collaboration } from '../types/protocol';

function makeCollab(overrides: Partial<Collaboration> = {}): Collaboration {
  return {
    id: 'collab-default',
    title: 'Default title',
    description: '',
    phase: 'executing',
    agents: [],
    tasks: [],
    channel: 'general',
    created_by: 'u1',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

describe('collaborationConfirm', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('confirmReplaceCollaborationExecution', () => {
    it('returns true when there is no executing collaboration', () => {
      expect(confirmReplaceCollaborationExecution(null, makeCollab({ id: 'next', phase: 'reviewing' }))).toBe(
        true
      );
      expect(
        confirmReplaceCollaborationExecution(undefined, makeCollab({ id: 'next', phase: 'approved' }))
      ).toBe(true);
    });

    it('returns true when the incoming collaboration is the same as the executing one', () => {
      const same = makeCollab({ id: 'same-id', phase: 'executing' });
      expect(confirmReplaceCollaborationExecution(same, makeCollab({ id: 'same-id', phase: 'executing' }))).toBe(
        true
      );
    });

    it('returns true when executing record is not in executing phase', () => {
      expect(
        confirmReplaceCollaborationExecution(
          makeCollab({ id: 'a', phase: 'reviewing' }),
          makeCollab({ id: 'b', phase: 'executing' })
        )
      ).toBe(true);
    });

    it('calls window.confirm and returns its result when another collaboration is executing', () => {
      const spy = vi.spyOn(window, 'confirm').mockReturnValue(true);
      const out = confirmReplaceCollaborationExecution(
        makeCollab({ id: 'run-a', title: 'Alpha run', phase: 'executing' }),
        makeCollab({ id: 'run-b', title: 'Beta run', phase: 'approved' })
      );
      expect(out).toBe(true);
      expect(spy).toHaveBeenCalledOnce();
      const prompt = String(spy.mock.calls[0][0]);
      expect(prompt).toContain('Alpha run');
      expect(prompt).toContain('Beta run');
      expect(prompt).toContain('Only one collaboration can execute');
    });

    it('returns false when the user declines the replace confirmation', () => {
      vi.spyOn(window, 'confirm').mockReturnValue(false);
      expect(
        confirmReplaceCollaborationExecution(
          makeCollab({ id: 'x', title: 'Current', phase: 'executing' }),
          makeCollab({ id: 'y', title: 'Next', phase: 'approved' })
        )
      ).toBe(false);
    });
  });

  describe('confirmStartCollaborationWhileExecuting', () => {
    it('returns true when nothing is executing', () => {
      expect(confirmStartCollaborationWhileExecuting(null)).toBe(true);
      expect(confirmStartCollaborationWhileExecuting(makeCollab({ phase: 'planning' }))).toBe(true);
    });

    it('prompts when a collaboration is executing', () => {
      const spy = vi.spyOn(window, 'confirm').mockReturnValue(true);
      expect(confirmStartCollaborationWhileExecuting(makeCollab({ title: 'In flight', phase: 'executing' }))).toBe(
        true
      );
      expect(spy).toHaveBeenCalledOnce();
      expect(String(spy.mock.calls[0][0])).toContain('In flight');
      expect(String(spy.mock.calls[0][0])).toContain('/collaborate');
    });

    it('returns false when the user cancels the start prompt', () => {
      vi.spyOn(window, 'confirm').mockReturnValue(false);
      expect(confirmStartCollaborationWhileExecuting(makeCollab({ phase: 'executing' }))).toBe(false);
    });
  });
});
