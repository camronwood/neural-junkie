import { useState, useRef } from 'react';
import type { Message } from '../types/protocol';
import { ChatAPI } from '../api/chatAPI';

interface UserQuestionCardProps {
  message: Message;
}

export function UserQuestionCard({ message }: UserQuestionCardProps) {
  const [loading, setLoading] = useState(false);
  const [freeText, setFreeText] = useState('');
  const apiRef = useRef(new ChatAPI());

  const questionId = message.metadata?.question_id as string | undefined;
  const question = (message.metadata?.question as string) || message.content;
  const options = (message.metadata?.options as string[] | undefined) ?? [];
  const status = message.metadata?.status as string | undefined;
  const answer = message.metadata?.answer as string | undefined;

  const isPending = status === 'pending';
  const isAnswered = status === 'answered';

  const submitAnswer = async (value: string) => {
    if (!questionId || !value.trim()) return;
    setLoading(true);
    try {
      await apiRef.current.answerUserQuestion(questionId, value.trim());
    } catch (err) {
      console.error('Failed to answer question:', err);
    } finally {
      setLoading(false);
    }
  };

  if (!isPending && !isAnswered) {
    return null;
  }

  return (
    <div className="mt-2 rounded-lg border border-slack-border bg-slack-bgAlt p-3 text-sm">
      <div className="font-medium text-slack-text mb-2">Agent question</div>
      <p className="text-slack-text whitespace-pre-wrap mb-3">{question}</p>

      {isAnswered && answer ? (
        <div className="text-slack-textMuted">
          <span className="font-medium text-slack-text">Your answer:</span> {answer}
        </div>
      ) : isPending ? (
        <div className="space-y-2">
          {options.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {options.map((opt) => (
                <button
                  key={opt}
                  type="button"
                  disabled={loading}
                  onClick={() => void submitAnswer(opt)}
                  className="px-3 py-1.5 rounded-md border border-slack-border bg-slack-bg hover:bg-slack-bgHover text-slack-text disabled:opacity-50"
                >
                  {opt}
                </button>
              ))}
            </div>
          ) : null}
          <div className="flex gap-2">
            <input
              type="text"
              value={freeText}
              onChange={(e) => setFreeText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') void submitAnswer(freeText);
              }}
              placeholder="Type your answer…"
              className="flex-1 px-2 py-1.5 rounded-md border border-slack-border bg-slack-bg text-slack-text text-sm"
              disabled={loading}
            />
            <button
              type="button"
              disabled={loading || !freeText.trim()}
              onClick={() => void submitAnswer(freeText)}
              className="px-3 py-1.5 rounded-md bg-slack-accent text-white font-medium disabled:opacity-50"
            >
              Reply
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
