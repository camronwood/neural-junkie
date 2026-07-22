export const THINKING_ACTIVITY_GENERATING_IMAGE = 'generating_image';
export const THINKING_ACTIVITY_GENERATING_MUSIC = 'generating_music';
export const THINKING_ACTIVITY_USING_TOOL = 'using_tool';
export const THINKING_ACTIVITY_REASONING = 'reasoning';
export const THINKING_ACTIVITY_WRITING = 'writing';
export const THINKING_ACTIVITY_VERIFYING = 'verifying';
export const THINKING_ACTIVITY_PROPOSING_EDIT = 'proposing_edit';
export const THINKING_ACTIVITY_IMPLEMENTATION = 'implementation';

/** Human-readable line for the typing indicator footer. */
export function formatThinkingActivityLabel(activity?: string, detail?: string): string {
  const d = detail?.trim();
  switch (activity) {
    case THINKING_ACTIVITY_GENERATING_IMAGE:
      return d ? `is generating an image — ${d}` : 'is generating an image';
    case THINKING_ACTIVITY_GENERATING_MUSIC:
      return d ? `is generating music — ${d}` : 'is generating music';
    case THINKING_ACTIVITY_USING_TOOL:
      return d ? `is using ${d}` : 'is using a tool';
    case THINKING_ACTIVITY_REASONING:
      return d ? `is reasoning — ${d}` : 'is reasoning';
    case THINKING_ACTIVITY_WRITING:
      return 'is writing a response';
    case THINKING_ACTIVITY_VERIFYING:
      return d ? `is verifying — ${d}` : 'is running verification';
    case THINKING_ACTIVITY_PROPOSING_EDIT:
      return d ? `is proposing edit to ${d}` : 'is proposing a file edit';
    case THINKING_ACTIVITY_IMPLEMENTATION:
      return d ? `is implementing — ${d}` : 'is running an implementation session';
    case 'routing':
      return d ? d : 'is delivering your message';
    default:
      if (activity && d) return `${activity}: ${d}`;
      if (activity) return `is ${activity.replace(/_/g, ' ')}`;
      return 'is thinking';
  }
}

/** Compact label for a tool-step history row. */
export function formatToolStepLabel(step: {
  kind: string;
  name: string;
  preview?: string;
}): string {
  const preview = step.preview?.trim();
  if (preview) return preview;
  const name = step.name || 'tool';
  if (step.kind === 'error') return `${name} failed`;
  if (step.kind === 'start') return `${name}…`;
  return name;
}
