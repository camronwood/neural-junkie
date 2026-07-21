import { describe, expect, it } from 'vitest';
import { pendingUserQuestionIds, pendingUserQuestionMessages } from './pendingUserQuestions';

describe('pendingUserQuestions', () => {
  it('filters pending ask_user cards', () => {
    const messages = [
      { type: 'chat', metadata: {} },
      { type: 'user_question', metadata: { question_id: 'q1', status: 'pending' } },
      { type: 'user_question', metadata: { question_id: 'q2', status: 'answered' } },
      { type: 'user_question', metadata: { question_id: 'q3', status: 'pending' } },
    ];
    expect(pendingUserQuestionMessages(messages)).toHaveLength(2);
    expect(pendingUserQuestionIds(messages)).toEqual(['q1', 'q3']);
  });

  it('deduplicates question IDs before the composer submits answers', () => {
    const messages = [
      { type: 'user_question', metadata: { question_id: 'q1', status: 'pending' } },
      { type: 'user_question', metadata: { question_id: 'q1', status: 'pending' } },
    ];
    expect(pendingUserQuestionIds(messages)).toEqual(['q1']);
  });
});
