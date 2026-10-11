import { describe, expect, it } from 'vitest';
import {
  looksLikeExecuteImplementationAsk,
  looksLikeImplementGoAhead,
  shouldStampAgentImplementationSession,
} from './implementationIntent';

const triviaPrompt =
  "Let's design and implement a simple iOS game where users answer basic trivia questions, get points for each correct answer and lose a life for each wrong answer. Users start with 3 lives, and each question should have a 15 second answer clock — if the timer ends they lose that round. Use .swift files under an Xcode-like folder TriviaGame/ with TriviaGameApp.swift and ContentView.swift (or GameView.swift). Start with the UI and then the game logic. Keep everything local (no network backend).";

describe('implementationIntent', () => {
  it('stamps greenfield design+implement asks', () => {
    expect(looksLikeExecuteImplementationAsk(triviaPrompt)).toBe(true);
    expect(shouldStampAgentImplementationSession(triviaPrompt, false)).toBe(true);
  });

  it('does not stamp design questions', () => {
    expect(
      looksLikeExecuteImplementationAsk('go deeper on the approach — what would you implement first?')
    ).toBe(false);
  });

  it('stamps short go-aheads only when a prior implement thread exists', () => {
    expect(looksLikeImplementGoAhead('ok that sounds good')).toBe(true);
    expect(shouldStampAgentImplementationSession('ok that sounds good', false)).toBe(false);
    expect(shouldStampAgentImplementationSession('ok that sounds good', true)).toBe(true);
  });
});
