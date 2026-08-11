/** Keep in sync with internal/config/wizard_profiles.go */

export type WizardTrack = 'developer' | 'lifeSciences' | 'cad' | 'general';

/** Primary OpenBio chat model (Ollama Hub). Keep in sync with internal/config/wizard_profiles.go */
export const BIO_OLLAMA_CHAT_MODEL = 'koesn/llama3-openbiollm-8b:latest';
export const BIO_OLLAMA_TOOL_MODEL = 'qwen3.5:9b';
/** Optional HF GGUF import tag */
export const BIO_OLLAMA_TAG = 'nj-bio:8b';
export const DEV_OLLAMA_MODEL = 'qwen3.5:27b';
export const CAD_OLLAMA_CHAT_MODEL = 'qwen3.5:27b';
export const CAD_OLLAMA_CHAT_MODEL_LIGHT = 'qwen3.5:9b';
export const CAD_OLLAMA_TOOL_MODEL = 'qwen3.5:9b';
export const CAD_OLLAMA_TAG = 'nj-cad:27b';
export const UTILITY_OLLAMA_MODEL = 'qwen3.5:9b';

export interface WizardAgentChoice {
  type: string;
  name: string;
  enabled: boolean;
}

export function agentsForTrack(track: WizardTrack): WizardAgentChoice[] {
  if (track === 'lifeSciences') {
    return [
      { type: 'biology', name: 'BiologyExpert', enabled: true },
      { type: 'assistant', name: 'Assistant', enabled: true },
    ];
  }
  if (track === 'cad') {
    return [
      { type: 'cad', name: 'CADExpert', enabled: true },
      { type: 'assistant', name: 'Assistant', enabled: true },
    ];
  }
  if (track === 'general') {
    return [{ type: 'assistant', name: 'Assistant', enabled: true }];
  }
  return [
    { type: 'assistant', name: 'Assistant', enabled: true },
    { type: 'backend', name: 'BackendEngineer', enabled: true },
  ];
}

export function ollamaModelForTrack(track: WizardTrack): string {
  if (track === 'lifeSciences') return BIO_OLLAMA_CHAT_MODEL;
  if (track === 'cad') return CAD_OLLAMA_CHAT_MODEL;
  if (track === 'general') return UTILITY_OLLAMA_MODEL;
  return UTILITY_OLLAMA_MODEL;
}

export function modelsToEnsureForTrack(track: WizardTrack, providerType: 'ollama' | 'cloud'): string[] {
  if (providerType !== 'ollama') return [];
  if (track === 'lifeSciences') return [BIO_OLLAMA_CHAT_MODEL, BIO_OLLAMA_TOOL_MODEL];
  if (track === 'cad') return [CAD_OLLAMA_CHAT_MODEL, CAD_OLLAMA_CHAT_MODEL_LIGHT, CAD_OLLAMA_TOOL_MODEL];
  if (track === 'general') return [UTILITY_OLLAMA_MODEL];
  return [UTILITY_OLLAMA_MODEL];
}

export function packsEnabledForTrack(track: WizardTrack): Record<string, boolean> {
  return {
    ide: track === 'developer',
    'life-sciences': track === 'lifeSciences',
    'software-development': track === 'developer',
    cad: track === 'cad',
  };
}

/** Infer focus track from hub packs.enabled (re-run setup prefill). */
export function inferWizardTrackFromPacks(
  enabled: Record<string, boolean> | null | undefined,
): WizardTrack {
  if (!enabled) return 'general';
  if (enabled['life-sciences']) return 'lifeSciences';
  if (enabled.cad) return 'cad';
  if (enabled.ide || enabled['software-development']) return 'developer';
  return 'general';
}
