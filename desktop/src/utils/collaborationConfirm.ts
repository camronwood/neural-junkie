import type { Collaboration } from '../types/protocol';

/** True if there is nothing to confirm or the user accepted replacing the running execution. */
export function confirmReplaceCollaborationExecution(
  executing: Collaboration | null | undefined,
  incoming: Collaboration
): boolean {
  if (!executing || executing.phase !== 'executing' || executing.id === incoming.id) {
    return true;
  }
  const runTitle = executing.title || 'Running collaboration';
  const nextTitle = incoming.title || 'This collaboration';
  return window.confirm(
    `Only one collaboration can execute at a time.\n\n` +
      `Stop the running collaboration:\n  "${runTitle}"\n\n` +
      `and start execution on:\n  "${nextTitle}"?\n\n` +
      `OK stops the current run and continues. Cancel leaves everything unchanged.`
  );
}

/** Warn before /collaborate when another collaboration is already executing in this channel. */
export function confirmStartCollaborationWhileExecuting(executing: Collaboration | null | undefined): boolean {
  if (!executing || executing.phase !== 'executing') {
    return true;
  }
  const runTitle = executing.title || 'Running collaboration';
  return window.confirm(
    `A collaboration is still executing:\n  "${runTitle}"\n\n` +
      `You can start another plan in parallel. When you approve the new plan, the current execution will be stopped automatically so only one collaboration runs at a time.\n\n` +
      `Continue with this /collaborate?`
  );
}
