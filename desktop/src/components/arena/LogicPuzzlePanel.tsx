type Props = {
  prompt: string;
  title?: string;
  difficulty?: string;
  answer: string;
  onAnswerChange: (v: string) => void;
  onSubmit: () => void;
  onAskModel: () => void;
  result?: string;
  explanation?: string;
  busy?: boolean;
};

export function LogicPuzzlePanel({
  prompt,
  title,
  difficulty,
  answer,
  onAnswerChange,
  onSubmit,
  onAskModel,
  result,
  explanation,
  busy,
}: Props) {
  return (
    <div className="arena-logic-panel">
      <div className="arena-logic-title">
        {title || 'LOGIC ROUND'}
        {difficulty && <span className="arena-logic-diff">{difficulty}</span>}
      </div>
      <p className="arena-logic-prompt whitespace-pre-wrap">{prompt}</p>
      <input
        type="text"
        value={answer}
        onChange={(e) => onAnswerChange(e.target.value)}
        placeholder="TYPE YOUR ANSWER"
        className="arena-logic-input"
      />
      <div className="flex flex-wrap gap-2">
        <button type="button" disabled={busy} onClick={onSubmit} className="arena-retro-btn action">
          Submit
        </button>
        <button type="button" disabled={busy} onClick={onAskModel} className="arena-retro-btn secondary">
          Ask model
        </button>
      </div>
      {result && <div className="arena-logic-result">RESULT · {result.toUpperCase()}</div>}
      {explanation && (
        <div className="text-[0.8125rem] text-slate-300 whitespace-pre-wrap mt-2 opacity-90">{explanation}</div>
      )}
    </div>
  );
}
