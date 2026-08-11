import { useEffect, useMemo, useState } from 'react';
import { firstWinCopyForTrack, type FirstWinAction, type FirstWinStepDef } from '../config/firstWinSteps';
import {
  computeFirstWinProgress,
  findAgentByType,
  findRepoAgent,
  FIRST_WIN_DISMISS_EVENT,
  isFirstWinCoachDismissed,
  packsEnabledFromRows,
  setFirstWinCoachDismissed,
  type FirstWinAgentLike,
} from '../utils/firstWinProgress';
import type { WizardTrack } from '../config/wizardProfiles';

export interface FirstWinCoachActions {
  onOpenFiles?: () => void;
  onOpenCommandPalette?: (filter: string) => void;
  onOpenAgentDM?: (agentId: string) => void;
  onPrefillComposer?: (text: string) => void;
  onOpenModelLibrary?: () => void;
}

export interface FirstWinCoachProps extends FirstWinCoachActions {
  track: WizardTrack;
  hasWorkspace: boolean;
  agents: FirstWinAgentLike[];
  myAgents?: Array<{ name: string; type: string }>;
  hasCollaboration: boolean;
  packsEnabled?: Record<string, boolean>;
  onDismissed?: () => void;
}

function samplePrompt(template: 'repo' | 'biology' | 'cad' | 'assistant', agents: FirstWinAgentLike[], myAgents: Array<{ name: string; type: string }>): string {
  if (template === 'repo') {
    const repo = findRepoAgent(agents, myAgents);
    const name = repo?.name || 'MyRepoExpert';
    return `@${name} summarize the architecture and top risk areas`;
  }
  if (template === 'biology') {
    const agent = findAgentByType(agents, 'biology');
    return `@${agent?.name || 'BiologyExpert'} what can you help me analyze in this workspace?`;
  }
  if (template === 'cad') {
    const agent = findAgentByType(agents, 'cad');
    return `@${agent?.name || 'CADExpert'} what should I know about the designs in this workspace?`;
  }
  const assistant = findAgentByType(agents, 'assistant');
  return `@${assistant?.name || 'Assistant'} what can you help me with in this workspace?`;
}

function runAction(
  action: FirstWinAction,
  props: FirstWinCoachProps,
): void {
  switch (action.type) {
    case 'openFiles':
      props.onOpenFiles?.();
      return;
    case 'openPalette':
      props.onOpenCommandPalette?.(action.filter);
      return;
    case 'openAgentDM': {
      const agent = findAgentByType(props.agents, action.agentType);
      if (agent?.id) {
        props.onOpenAgentDM?.(agent.id);
      } else {
        props.onOpenCommandPalette?.('');
      }
      return;
    }
    case 'prefillComposer':
      props.onPrefillComposer?.(samplePrompt(action.template, props.agents, props.myAgents ?? []));
      return;
    case 'openModelLibrary':
      props.onOpenModelLibrary?.();
      return;
  }
}

export function FirstWinCoach(props: FirstWinCoachProps) {
  const packsEnabled = props.packsEnabled ?? packsEnabledFromRows([]);
  const progress = useMemo(
    () =>
      computeFirstWinProgress({
        packsEnabled,
        track: props.track,
        hasWorkspace: props.hasWorkspace,
        agents: props.agents,
        myAgents: props.myAgents,
        hasCollaboration: props.hasCollaboration,
      }),
    [packsEnabled, props.track, props.hasWorkspace, props.agents, props.myAgents, props.hasCollaboration],
  );
  const copy = firstWinCopyForTrack(props.track);
  const completeById = useMemo(
    () => new Map(progress.steps.map((s) => [s.id, s.complete])),
    [progress.steps],
  );
  const nextStep = copy.steps.find((step) => !completeById.get(step.id));

  const handleDismiss = () => {
    setFirstWinCoachDismissed(true);
    props.onDismissed?.();
  };

  return (
    <div
      data-testid="first-win-coach"
      className="w-full max-w-md text-left rounded-lg border border-slack-border bg-slack-bgHover/40 px-5 py-5"
    >
      <p className="text-lg text-slack-text mb-1">{copy.headline}</p>
      <p className="text-sm text-slack-textMuted mb-4">{copy.lead}</p>
      <ol className="space-y-3 mb-4">
        {copy.steps.map((step, index) => (
          <FirstWinStepRow
            key={step.id}
            step={step}
            index={index + 1}
            complete={Boolean(completeById.get(step.id))}
            isNext={nextStep?.id === step.id}
            onAction={() => runAction(step.action, props)}
          />
        ))}
      </ol>
      <div className="flex items-center justify-between gap-3">
        {nextStep ? (
          <button
            type="button"
            data-testid="first-win-primary-cta"
            className="px-3 py-1.5 text-sm rounded bg-slack-accent hover:bg-slack-accentHover text-white"
            onClick={() => runAction(nextStep.action, props)}
          >
            {nextStep.ctaLabel}
          </button>
        ) : (
          <span />
        )}
        <button
          type="button"
          data-testid="first-win-dismiss"
          className="text-xs text-slack-textMuted hover:text-slack-text"
          onClick={handleDismiss}
        >
          Skip for now
        </button>
      </div>
    </div>
  );
}

function FirstWinStepRow({
  step,
  index,
  complete,
  isNext,
  onAction,
}: {
  step: FirstWinStepDef;
  index: number;
  complete: boolean;
  isNext: boolean;
  onAction: () => void;
}) {
  return (
    <li className="flex items-start gap-3">
      <span
        className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[11px] font-semibold ${
          complete
            ? 'bg-emerald-700/80 text-white'
            : isNext
              ? 'bg-slack-accent text-white'
              : 'bg-slack-border text-slack-textMuted'
        }`}
        aria-hidden
      >
        {complete ? '✓' : index}
      </span>
      <div className="min-w-0 flex-1">
        <p className={`text-sm ${complete ? 'text-slack-textMuted line-through' : 'text-slack-text'}`}>
          {step.title}
          {step.optional ? <span className="ml-1 text-[11px] no-underline text-slack-textMuted">(optional)</span> : null}
        </p>
        <p className="text-xs text-slack-textMuted mt-0.5">{step.description}</p>
        {!complete && isNext ? (
          <button
            type="button"
            className="mt-1.5 text-xs text-slack-accent hover:underline"
            onClick={onAction}
          >
            {step.ctaLabel}
          </button>
        ) : null}
      </div>
    </li>
  );
}

export function useFirstWinDismissed(): [boolean, (next: boolean) => void] {
  const [dismissed, setDismissed] = useState(() => isFirstWinCoachDismissed());
  useEffect(() => {
    const sync = () => setDismissed(isFirstWinCoachDismissed());
    window.addEventListener(FIRST_WIN_DISMISS_EVENT, sync);
    return () => window.removeEventListener(FIRST_WIN_DISMISS_EVENT, sync);
  }, []);
  return [
    dismissed,
    (next: boolean) => {
      setFirstWinCoachDismissed(next);
      setDismissed(next);
    },
  ];
}
