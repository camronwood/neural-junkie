/** Keep in sync with internal/config/wizard_profiles.go */

export type WizardTrack = 'developer' | 'lifeSciences' | 'general';

/** Primary OpenBio chat model (Ollama Hub). Keep in sync with internal/config/wizard_profiles.go */
export const BIO_OLLAMA_CHAT_MODEL = 'koesn/llama3-openbiollm-8b:latest';
export const BIO_OLLAMA_TOOL_MODEL = 'qwen2.5:7b';
/** Optional HF GGUF import tag */
export const BIO_OLLAMA_TAG = 'nj-bio:8b';
export const DEV_OLLAMA_MODEL = 'qwen2.5-coder:14b';
export const UTILITY_OLLAMA_MODEL = 'qwen2.5:7b';

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
  if (track === 'general') {
    return [{ type: 'assistant', name: 'Assistant', enabled: true }];
  }
  return [{ type: 'assistant', name: 'Assistant', enabled: true }];
}

export function ollamaModelForTrack(track: WizardTrack): string {
  if (track === 'lifeSciences') return BIO_OLLAMA_CHAT_MODEL;
  if (track === 'general') return UTILITY_OLLAMA_MODEL;
  return DEV_OLLAMA_MODEL;
}

export function modelsToEnsureForTrack(track: WizardTrack, providerType: 'ollama' | 'cloud'): string[] {
  if (providerType !== 'ollama') return [];
  if (track === 'lifeSciences') return [BIO_OLLAMA_CHAT_MODEL, BIO_OLLAMA_TOOL_MODEL];
  if (track === 'general') return [UTILITY_OLLAMA_MODEL];
  return [DEV_OLLAMA_MODEL, UTILITY_OLLAMA_MODEL];
}

export function packsEnabledForTrack(track: WizardTrack): Record<string, boolean> {
  return {
    'life-sciences': track === 'lifeSciences',
    'software-development': track === 'developer',
  };
}
