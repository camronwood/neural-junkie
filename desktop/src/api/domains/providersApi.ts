import type { ConnectionTestResult } from '../../types/protocol';
import type { HubFetchFn } from './packsApi';

/** Provider connection tests, Ollama/LM Studio/HF, and provider listing. */
export class ProvidersApi {
  constructor(private readonly hubFetch: HubFetchFn) {}

  async testAnthropicConnection(
    apiKey: string,
    useAIHub: boolean = true,
    aiHubEndpoint?: string
  ): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        anthropic_api_key: apiKey,
        use_ai_hub: useAIHub,
        ai_hub_endpoint: aiHubEndpoint,
      };

      const response = await this.hubFetch(`/api/test-anthropic-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  async testGitHubConnection(personalAccessToken: string): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        github_token: personalAccessToken,
      };

      const response = await this.hubFetch(`/api/test-github-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  async testConfluenceConnection(
    domain: string,
    email: string,
    apiToken: string
  ): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        confluence_credentials: {
          domain,
          email,
          api_token: apiToken,
        },
      };

      const response = await this.hubFetch(`/api/test-confluence-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  async testOllamaConnection(endpoint: string, model: string): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        endpoint,
        model,
      };

      const response = await this.hubFetch(`/api/test-ollama-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  async switchAllAgentProviders(provider: string, model: string): Promise<void> {
    const response = await this.hubFetch(`/api/agents/switch-all-providers`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ provider, model }),
    });

    if (!response.ok) {
      throw new Error(`Failed to switch all agents: ${response.statusText}`);
    }
  }

  async fetchOllamaStatus(): Promise<{ running: boolean; endpoint: string; error?: string }> {
    const response = await this.hubFetch(`/api/ollama/status`);

    if (!response.ok) {
      throw new Error(`Failed to fetch Ollama status: ${response.statusText}`);
    }

    return response.json();
  }

  async fetchOllamaModels(endpoint?: string): Promise<string[]> {
    const path = endpoint
      ? `/api/ollama/models?endpoint=${encodeURIComponent(endpoint)}`
      : '/api/ollama/models';
    const response = await this.hubFetch(path);

    if (!response.ok) {
      throw new Error(`Failed to fetch Ollama models: ${response.statusText}`);
    }

    const result = await response.json();
    return result.models || [];
  }

  async testLMStudioConnection(endpoint: string, model: string): Promise<ConnectionTestResult> {
    try {
      const credentials = {
        endpoint,
        model,
      };

      const response = await this.hubFetch(`/api/test-lmstudio-connection`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials),
      });

      const result = await response.json();
      return {
        success: response.ok,
        message: result.message || (response.ok ? 'Connection successful' : 'Connection failed'),
        error: result.error,
      };
    } catch (error) {
      return {
        success: false,
        message: 'Connection test failed',
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  async fetchLMStudioStatus(): Promise<{ running: boolean; endpoint: string; error?: string }> {
    const response = await this.hubFetch(`/api/lmstudio/status`);

    if (!response.ok) {
      throw new Error(`Failed to fetch LM Studio status: ${response.statusText}`);
    }

    return response.json();
  }

  async fetchLMStudioModels(endpoint?: string): Promise<string[]> {
    const path = endpoint
      ? `/api/lmstudio/models?endpoint=${encodeURIComponent(endpoint)}`
      : '/api/lmstudio/models';
    const response = await this.hubFetch(path);

    if (!response.ok) {
      throw new Error(`Failed to fetch LM Studio models: ${response.statusText}`);
    }

    const result = await response.json();
    return result.models || [];
  }

  async fetchHfCatalog(): Promise<
    {
      repo_id: string;
      title: string;
      description: string;
      tags: string[];
      modes: string[];
      files?: { filename: string; quant?: string }[];
    }[]
  > {
    const response = await this.hubFetch(`/api/hf/catalog`);
    if (!response.ok) {
      throw new Error(`Failed to fetch HF catalog: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchHfStatus(): Promise<{
    token_configured: boolean;
    router_reachable: boolean;
    cache_dir?: string;
  }> {
    const response = await this.hubFetch(`/api/hf/status`);
    if (!response.ok) {
      throw new Error(`Failed to fetch HF status: ${response.statusText}`);
    }
    return response.json();
  }

  async fetchProviders(): Promise<
    { id: string; type: string; name: string; model?: string; endpoint?: string }[]
  > {
    const response = await this.hubFetch(`/api/providers`);
    if (!response.ok) {
      throw new Error(`Failed to fetch providers: ${response.statusText}`);
    }
    return response.json();
  }

  /** Overwrite credential strings in place to clear them from memory. */
  static clearCredentials(credentials: Record<string, any>): void {
    for (const key in credentials) {
      if (typeof credentials[key] === 'string') {
        credentials[key] = 'x'.repeat(credentials[key].length);
      } else if (typeof credentials[key] === 'object' && credentials[key] !== null) {
        ProvidersApi.clearCredentials(credentials[key]);
      }
    }
  }
}
