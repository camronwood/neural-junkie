import { create } from 'zustand';
import { Store } from '@tauri-apps/plugin-store';
import { ChatAPI } from '../api/chatAPI';

export type FontSizeScope = 'messages' | 'input' | 'global';

export interface Settings {
  fontSize: number;
  fontSizeScope: FontSizeScope;
}

export interface AnthropicSettings {
  apiKey: string;
  useAIHub: boolean;
  aiHubEndpoint: string;
  aiHubModel: string;
}

export interface GitHubSettings {
  personalAccessToken: string;
}

export interface ConfluenceSettings {
  domain: string;
  email: string;
  apiToken: string;
}

export interface OllamaSettings {
  endpoint: string;
  defaultModel: string;
  availableModels: string[];
}

export interface LMStudioSettings {
  endpoint: string;
  defaultModel: string;
  availableModels: string[];
}

export interface IntegrationSettings {
  anthropic: AnthropicSettings;
  github: GitHubSettings;
  confluence: ConfluenceSettings;
  ollama: OllamaSettings;
  lmstudio: LMStudioSettings;
}

export interface LayoutSettings {
  filesPanelVisible: boolean;
  editorPanelVisible: boolean;
  terminalPanelVisible: boolean;
  myAgentsPanelVisible: boolean;
  pendingChangesPanelVisible: boolean;
  /** When false, agent shortcuts under Direct Messages are hidden (existing DM rows stay). */
  sidebarAgentsVisible: boolean;
}

interface SettingsState {
  settings: Settings;
  integrations: IntegrationSettings;
  layoutSettings: LayoutSettings;
  isLoaded: boolean;
  store: Store | null;
  integrationsStore: Store | null;
  layoutStore: Store | null;
  
  // Actions
  loadSettings: () => Promise<void>;
  updateFontSize: (size: number) => Promise<void>;
  updateFontSizeScope: (scope: FontSizeScope) => Promise<void>;
  resetSettings: () => Promise<void>;
  
  // Integration actions
  loadIntegrations: () => Promise<void>;
  updateAnthropicSettings: (settings: Partial<AnthropicSettings>) => Promise<void>;
  updateGitHubSettings: (settings: Partial<GitHubSettings>) => Promise<void>;
  updateConfluenceSettings: (settings: Partial<ConfluenceSettings>) => Promise<void>;
  updateOllamaSettings: (settings: Partial<OllamaSettings>) => Promise<void>;
  updateLMStudioSettings: (settings: Partial<LMStudioSettings>) => Promise<void>;
  clearIntegrationSettings: () => Promise<void>;
  testAnthropicConnection: () => Promise<boolean>;
  testGitHubConnection: () => Promise<boolean>;
  testConfluenceConnection: () => Promise<boolean>;
  testOllamaConnection: () => Promise<boolean>;
  testLMStudioConnection: () => Promise<boolean>;
  fetchOllamaModels: () => Promise<string[]>;
  fetchLMStudioModels: () => Promise<string[]>;
  
  // Layout actions
  loadLayoutSettings: () => Promise<void>;
  updateLayoutSettings: (settings: Partial<LayoutSettings>) => Promise<void>;
  resetLayoutSettings: () => Promise<void>;
  
  maskCredential: (credential: string, visibleChars?: number) => string;
}

const defaultSettings: Settings = {
  fontSize: 16,
  fontSizeScope: 'messages',
};

const defaultIntegrations: IntegrationSettings = {
  anthropic: {
    apiKey: '',
    useAIHub: true,
    aiHubEndpoint: '',
    aiHubModel: 'claude-sonnet',
  },
  github: {
    personalAccessToken: '',
  },
  confluence: {
    domain: '',
    email: '',
    apiToken: '',
  },
  ollama: {
    endpoint: 'http://localhost:11434',
    defaultModel: 'llama3.1',
    availableModels: [],
  },
  lmstudio: {
    endpoint: 'http://localhost:1234/v1',
    defaultModel: '',
    availableModels: [],
  },
};

const defaultLayoutSettings: LayoutSettings = {
  filesPanelVisible: false,
  editorPanelVisible: false,
  terminalPanelVisible: false,
  myAgentsPanelVisible: false,
  pendingChangesPanelVisible: false,
  sidebarAgentsVisible: true,
};

export const useSettingsStore = create<SettingsState>((set, get) => ({
  settings: defaultSettings,
  integrations: defaultIntegrations,
  layoutSettings: defaultLayoutSettings,
  isLoaded: false,
  store: null,
  integrationsStore: null,
  layoutStore: null,
  
  loadSettings: async () => {
    try {
      // Initialize Tauri store
      const store = new Store('.neural-junkie-settings.dat');
      
      // Load settings from store
      const savedSettings = await store.get<Settings>('settings');
      
      if (savedSettings) {
        set({ 
          settings: { ...defaultSettings, ...savedSettings },
          isLoaded: true,
          store 
        });
      } else {
        // No saved settings, use defaults
        set({ 
          settings: defaultSettings,
          isLoaded: true,
          store 
        });
        // Save defaults to store
        await store.set('settings', defaultSettings);
        await store.save();
      }
    } catch (error) {
      console.error('Failed to load settings:', error);
      // Fallback to defaults if store fails
      set({ 
        settings: defaultSettings,
        isLoaded: true,
        store: null 
      });
    }
  },
  
  updateFontSize: async (size: number) => {
    const { store } = get();
    const newSettings = { ...get().settings, fontSize: size };
    
    set({ settings: newSettings });
    
    if (store) {
      try {
        await store.set('settings', newSettings);
        await store.save();
      } catch (error) {
        console.error('Failed to save font size:', error);
      }
    }
  },
  
  updateFontSizeScope: async (scope: FontSizeScope) => {
    const { store } = get();
    const newSettings = { ...get().settings, fontSizeScope: scope };
    
    set({ settings: newSettings });
    
    if (store) {
      try {
        await store.set('settings', newSettings);
        await store.save();
      } catch (error) {
        console.error('Failed to save font size scope:', error);
      }
    }
  },
  
  resetSettings: async () => {
    const { store } = get();
    
    set({ settings: defaultSettings });
    
    if (store) {
      try {
        await store.set('settings', defaultSettings);
        await store.save();
      } catch (error) {
        console.error('Failed to reset settings:', error);
      }
    }
  },

  // Integration methods
  loadIntegrations: async () => {
    try {
      // Initialize integrations store
      const integrationsStore = new Store('.neural-junkie-integrations.dat');
      
      // Load integrations from store
      const savedIntegrations = await integrationsStore.get<IntegrationSettings>('integrations');
      
      if (savedIntegrations) {
        set({ 
          integrations: { ...defaultIntegrations, ...savedIntegrations },
          integrationsStore 
        });
      } else {
        // No saved integrations, use defaults
        set({ 
          integrations: defaultIntegrations,
          integrationsStore 
        });
        // Save defaults to store
        await integrationsStore.set('integrations', defaultIntegrations);
        await integrationsStore.save();
      }
    } catch (error) {
      console.error('Failed to load integrations:', error);
      // Fallback to defaults if store fails
      set({ 
        integrations: defaultIntegrations,
        integrationsStore: null 
      });
    }
  },

  updateAnthropicSettings: async (settings: Partial<AnthropicSettings>) => {
    const { integrationsStore } = get();
    
    // Validate API key format
    if (settings.apiKey && !settings.apiKey.startsWith('sk-ant-') && !settings.apiKey.startsWith('aihub-')) {
      throw new Error('Invalid API key format. Anthropic keys should start with "sk-ant-" or AI Hub keys with "aihub-"');
    }
    
    // Validate AI Hub endpoint format
    if (settings.aiHubEndpoint && !settings.aiHubEndpoint.startsWith('https://')) {
      throw new Error('AI Hub endpoint must be a valid HTTPS URL');
    }
    
    const newIntegrations = { 
      ...get().integrations, 
      anthropic: { ...get().integrations.anthropic, ...settings } 
    };
    
    set({ integrations: newIntegrations });
    
    if (integrationsStore) {
      try {
        await integrationsStore.set('integrations', newIntegrations);
        await integrationsStore.save();
      } catch (error) {
        console.error('Failed to save Anthropic settings:', error);
        throw error;
      }
    }
  },

  updateGitHubSettings: async (settings: Partial<GitHubSettings>) => {
    const { integrationsStore } = get();
    
    // Validate GitHub token format
    if (settings.personalAccessToken && !settings.personalAccessToken.startsWith('ghp_') && !settings.personalAccessToken.startsWith('github_pat_')) {
      throw new Error('Invalid GitHub token format. Tokens should start with "ghp_" or "github_pat_"');
    }
    
    const newIntegrations = { 
      ...get().integrations, 
      github: { ...get().integrations.github, ...settings } 
    };
    
    set({ integrations: newIntegrations });
    
    if (integrationsStore) {
      try {
        await integrationsStore.set('integrations', newIntegrations);
        await integrationsStore.save();
      } catch (error) {
        console.error('Failed to save GitHub settings:', error);
        throw error;
      }
    }
  },

  updateConfluenceSettings: async (settings: Partial<ConfluenceSettings>) => {
    const { integrationsStore } = get();
    
    // Validate email format
    if (settings.email && !settings.email.includes('@')) {
      throw new Error('Invalid email format');
    }
    
    // Validate domain format
    if (settings.domain && !settings.domain.includes('.')) {
      throw new Error('Invalid domain format. Should be like "yourcompany.atlassian.net"');
    }
    
    // Validate API token is not empty if provided
    if (settings.apiToken && settings.apiToken.trim() === '') {
      throw new Error('API token cannot be empty');
    }
    
    const newIntegrations = { 
      ...get().integrations, 
      confluence: { ...get().integrations.confluence, ...settings } 
    };
    
    set({ integrations: newIntegrations });
    
    if (integrationsStore) {
      try {
        await integrationsStore.set('integrations', newIntegrations);
        await integrationsStore.save();
      } catch (error) {
        console.error('Failed to save Confluence settings:', error);
        throw error;
      }
    }
  },

  clearIntegrationSettings: async () => {
    const { integrationsStore } = get();
    
    set({ integrations: defaultIntegrations });
    
    if (integrationsStore) {
      try {
        await integrationsStore.set('integrations', defaultIntegrations);
        await integrationsStore.save();
      } catch (error) {
        console.error('Failed to clear integration settings:', error);
      }
    }
  },

  testAnthropicConnection: async () => {
    const { integrations } = get();
    const api = new ChatAPI();
    
    try {
      const result = await api.testAnthropicConnection(
        integrations.anthropic.apiKey,
        integrations.anthropic.useAIHub,
        integrations.anthropic.aiHubEndpoint
      );
      return result.success;
    } catch (error) {
      console.error('Anthropic connection test failed:', error);
      return false;
    }
  },

  testGitHubConnection: async () => {
    const { integrations } = get();
    const api = new ChatAPI();
    
    try {
      const result = await api.testGitHubConnection(integrations.github.personalAccessToken);
      return result.success;
    } catch (error) {
      console.error('GitHub connection test failed:', error);
      return false;
    }
  },

  testConfluenceConnection: async () => {
    const { integrations } = get();
    const api = new ChatAPI();
    
    try {
      const result = await api.testConfluenceConnection(
        integrations.confluence.domain,
        integrations.confluence.email,
        integrations.confluence.apiToken
      );
      return result.success;
    } catch (error) {
      console.error('Confluence connection test failed:', error);
      return false;
    }
  },

  updateOllamaSettings: async (settings: Partial<OllamaSettings>) => {
    const { integrationsStore } = get();
    
    // Validate endpoint format
    if (settings.endpoint && !settings.endpoint.startsWith('http://') && !settings.endpoint.startsWith('https://')) {
      throw new Error('Endpoint must be a valid HTTP/HTTPS URL');
    }
    
    const newIntegrations = { 
      ...get().integrations, 
      ollama: { ...get().integrations.ollama, ...settings } 
    };
    
    set({ integrations: newIntegrations });
    
    if (integrationsStore) {
      try {
        await integrationsStore.set('integrations', newIntegrations);
        await integrationsStore.save();
      } catch (error) {
        console.error('Failed to save Ollama settings:', error);
        throw error;
      }
    }
  },

  testOllamaConnection: async () => {
    const { integrations } = get();
    const api = new ChatAPI();
    
    try {
      const result = await api.testOllamaConnection(integrations.ollama.endpoint, integrations.ollama.defaultModel);
      return result.success;
    } catch (error) {
      console.error('Ollama connection test failed:', error);
      return false;
    }
  },

  fetchOllamaModels: async () => {
    const { integrations } = get();
    const api = new ChatAPI();
    
    try {
      const models = await api.fetchOllamaModels(integrations.ollama.endpoint);
      // Update the available models in the store
      await get().updateOllamaSettings({ availableModels: models });
      return models;
    } catch (error) {
      console.error('Failed to fetch Ollama models:', error);
      return [];
    }
  },

  updateLMStudioSettings: async (settings: Partial<LMStudioSettings>) => {
    const { integrationsStore } = get();
    
    // Validate endpoint format
    if (settings.endpoint && !settings.endpoint.startsWith('http://') && !settings.endpoint.startsWith('https://')) {
      throw new Error('Endpoint must be a valid HTTP/HTTPS URL');
    }
    
    const newIntegrations = { 
      ...get().integrations, 
      lmstudio: { ...get().integrations.lmstudio, ...settings } 
    };
    
    set({ integrations: newIntegrations });
    
    if (integrationsStore) {
      try {
        await integrationsStore.set('integrations', newIntegrations);
        await integrationsStore.save();
      } catch (error) {
        console.error('Failed to save LM Studio settings:', error);
        throw error;
      }
    }
  },

  testLMStudioConnection: async () => {
    const { integrations } = get();
    const api = new ChatAPI();
    
    try {
      const result = await api.testLMStudioConnection(integrations.lmstudio.endpoint, integrations.lmstudio.defaultModel);
      return result.success;
    } catch (error) {
      console.error('LM Studio connection test failed:', error);
      return false;
    }
  },

  fetchLMStudioModels: async () => {
    const { integrations } = get();
    const api = new ChatAPI();
    
    try {
      const models = await api.fetchLMStudioModels(integrations.lmstudio.endpoint);
      // Update the available models in the store
      await get().updateLMStudioSettings({ availableModels: models });
      return models;
    } catch (error) {
      console.error('Failed to fetch LM Studio models:', error);
      return [];
    }
  },

  // Layout methods
  loadLayoutSettings: async () => {
    try {
      // Initialize layout store
      const layoutStore = new Store('.neural-junkie-layout.dat');
      
      // Load layout settings from store
      const savedLayoutSettings = await layoutStore.get<LayoutSettings>('layoutSettings');
      
      if (savedLayoutSettings) {
        set({ 
          layoutSettings: { ...defaultLayoutSettings, ...savedLayoutSettings },
          layoutStore 
        });
      } else {
        // No saved layout settings, use defaults
        set({ 
          layoutSettings: defaultLayoutSettings,
          layoutStore 
        });
        // Save defaults to store
        await layoutStore.set('layoutSettings', defaultLayoutSettings);
        await layoutStore.save();
      }
    } catch (error) {
      console.error('Failed to load layout settings:', error);
      // Fallback to defaults if store fails
      set({ 
        layoutSettings: defaultLayoutSettings,
        layoutStore: null 
      });
    }
  },

  updateLayoutSettings: async (settings: Partial<LayoutSettings>) => {
    const { layoutStore } = get();
    const newLayoutSettings = { 
      ...get().layoutSettings, 
      ...settings 
    };
    
    set({ layoutSettings: newLayoutSettings });
    
    if (layoutStore) {
      try {
        await layoutStore.set('layoutSettings', newLayoutSettings);
        await layoutStore.save();
      } catch (error) {
        console.error('Failed to save layout settings:', error);
      }
    }
  },

  resetLayoutSettings: async () => {
    const { layoutStore } = get();
    
    set({ layoutSettings: defaultLayoutSettings });
    
    if (layoutStore) {
      try {
        await layoutStore.set('layoutSettings', defaultLayoutSettings);
        await layoutStore.save();
      } catch (error) {
        console.error('Failed to reset layout settings:', error);
      }
    }
  },

  // Utility function to mask credentials for display
  maskCredential: (credential: string, visibleChars: number = 4): string => {
    if (!credential || credential.length <= visibleChars) {
      return '*'.repeat(credential.length);
    }
    return credential.slice(0, visibleChars) + '*'.repeat(credential.length - visibleChars);
  },
}));
