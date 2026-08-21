import { isPlanFormatInvalid } from '../utils/planInvalid';

export function PlanInvalidCard() {
  return (
    <div
      data-testid="plan-invalid-card"
      className="mt-3 rounded-md border border-amber-700/50 bg-amber-950/20 p-3 text-sm"
    >
      <div className="text-xs uppercase tracking-wide text-amber-300 font-semibold">Plan format invalid</div>
      <p className="mt-1 text-slack-textMuted text-xs">
        The model reply could not be parsed into a structured plan after normalization and retry. Send the
        request again in Plan mode, or switch to a model with stronger format compliance.
      </p>
    </div>
  );
}

export function shouldShowPlanInvalidCard(metadata?: Record<string, unknown> | null): boolean {
  return isPlanFormatInvalid(metadata);
}
