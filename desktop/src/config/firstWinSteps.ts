import type { WizardTrack } from './wizardProfiles';

export const FIRST_WIN_DISMISS_KEY = 'nj-first-win-coach-dismissed';

export type FirstWinStepId =
  | 'workspace'
  | 'repoAgent'
  | 'askRepoExpert'
  | 'collaborate'
  | 'specialist'
  | 'askSpecialist'
  | 'modelLibrary'
  | 'askAssistant'
  | 'discoverCommands';

export type FirstWinAction =
  | { type: 'openFiles' }
  | { type: 'openPalette'; filter: string }
  | { type: 'openAgentDM'; agentType: string }
  | { type: 'prefillComposer'; template: 'repo' | 'biology' | 'cad' | 'assistant' }
  | { type: 'openModelLibrary' };

export interface FirstWinStepDef {
  id: FirstWinStepId;
  title: string;
  description: string;
  ctaLabel: string;
  action: FirstWinAction;
  optional?: boolean;
}

export interface FirstWinTrackCopy {
  headline: string;
  lead: string;
  steps: FirstWinStepDef[];
}

export const FIRST_WIN_BY_TRACK: Record<WizardTrack, FirstWinTrackCopy> = {
  developer: {
    headline: 'Get to a first win',
    lead: 'Link a folder, create a repo expert, then ask it something real. Collaboration is optional but that’s the multi-agent loop.',
    steps: [
      {
        id: 'workspace',
        title: 'Open a workspace folder',
        description: 'Give agents a project tree they can see and edit with your approval.',
        ctaLabel: 'Open Files',
        action: { type: 'openFiles' },
      },
      {
        id: 'repoAgent',
        title: 'Create a repo expert',
        description: 'A specialist indexed on your codebase — the fastest path to a useful answer.',
        ctaLabel: 'Create repo expert',
        action: { type: 'openPalette', filter: 'create-repo-agent' },
      },
      {
        id: 'askRepoExpert',
        title: 'Ask your repo expert',
        description: 'Try a real question about architecture or risk in this repo.',
        ctaLabel: 'Prefill a prompt',
        action: { type: 'prefillComposer', template: 'repo' },
      },
      {
        id: 'collaborate',
        title: 'Start a collaboration',
        description: 'Optional: a bounded session with specialists, a shared plan, and human approval.',
        ctaLabel: 'Collaborate',
        action: { type: 'openPalette', filter: 'collaborate' },
        optional: true,
      },
    ],
  },
  lifeSciences: {
    headline: 'Get to a first lab win',
    lead: 'Talk to BiologyExpert, then pull models if you need local sequence or tool routing.',
    steps: [
      {
        id: 'specialist',
        title: 'Meet BiologyExpert',
        description: 'Your lab specialist is ready after setup. Open a DM to start.',
        ctaLabel: 'Message BiologyExpert',
        action: { type: 'openAgentDM', agentType: 'biology' },
      },
      {
        id: 'askSpecialist',
        title: 'Ask a lab question',
        description: 'Try a real research or sequence question in that DM.',
        ctaLabel: 'Prefill a prompt',
        action: { type: 'prefillComposer', template: 'biology' },
      },
      {
        id: 'modelLibrary',
        title: 'Check the model library',
        description: 'Optional: confirm OpenBioLLM and a tool-routing model are available.',
        ctaLabel: 'Open Model Library',
        action: { type: 'openModelLibrary' },
        optional: true,
      },
    ],
  },
  cad: {
    headline: 'Get to a first CAD win',
    lead: 'Open a folder if you have drawings, then ask CADExpert about the design.',
    steps: [
      {
        id: 'workspace',
        title: 'Open a workspace folder',
        description: 'Link a directory with CAD files so the expert can see context.',
        ctaLabel: 'Open Files',
        action: { type: 'openFiles' },
      },
      {
        id: 'specialist',
        title: 'Meet CADExpert',
        description: 'Your CAD specialist is ready after setup. Open a DM to start.',
        ctaLabel: 'Message CADExpert',
        action: { type: 'openAgentDM', agentType: 'cad' },
      },
      {
        id: 'askSpecialist',
        title: 'Ask a design question',
        description: 'Try a real question about a part, drawing, or assembly.',
        ctaLabel: 'Prefill a prompt',
        action: { type: 'prefillComposer', template: 'cad' },
      },
    ],
  },
  general: {
    headline: 'Get to a first win',
    lead: 'Ask Assistant what it can do here, or open the command palette to discover actions.',
    steps: [
      {
        id: 'askAssistant',
        title: 'Ask Assistant',
        description: 'Start with a real question about this workspace.',
        ctaLabel: 'Prefill a prompt',
        action: { type: 'prefillComposer', template: 'assistant' },
      },
      {
        id: 'discoverCommands',
        title: 'Discover commands',
        description: 'Browse slash commands without memorizing them.',
        ctaLabel: 'Open command palette',
        action: { type: 'openPalette', filter: '' },
      },
    ],
  },
};

export function firstWinCopyForTrack(track: WizardTrack): FirstWinTrackCopy {
  return FIRST_WIN_BY_TRACK[track] ?? FIRST_WIN_BY_TRACK.general;
}
