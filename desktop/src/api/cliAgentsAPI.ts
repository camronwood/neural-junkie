export type CLIAuthState = 'not_applicable' | 'unknown' | 'needs_auth' | 'authed';

export interface CLIInstallSpec {
  method: string;
  command: string;
  prereqs?: string[];
}

export interface CLIAuthSpec {
  method: string;
  env_vars?: string[];
  login_command?: string[];
  probe_command?: string[];
  credential_paths?: string[];
}

export interface CLIAgentStatus {
  type: string;
  name: string;
  provider_name: string;
  featured: boolean;
  installed: boolean;
  binary?: string;
  binary_path?: string;
  version?: string;
  auth_state: CLIAuthState;
  auth_method?: string;
  login_command?: string;
  install_hint?: string;
  can_install: boolean;
  missing_prereqs?: string[];
  install?: CLIInstallSpec;
  auth?: CLIAuthSpec;
}

export interface CLIAgentsResponse {
  agents: CLIAgentStatus[];
}

export interface CLIAuthInfo {
  mode: string;
  command?: string;
  env_var?: string;
}

export interface CLIActivateResponse {
  activated: boolean;
  already_active?: boolean;
  type: string;
  name: string;
  error?: string;
}

async function readSSEText(
  resp: Response,
  onLine: (line: string) => void
): Promise<void> {
  const reader = resp.body?.getReader();
  if (!reader) {
    throw new Error('No response body');
  }
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split('\n\n');
    buffer = parts.pop() ?? '';
    for (const chunk of parts) {
      for (const line of chunk.split('\n')) {
        if (line.startsWith('data: ')) {
          onLine(line.slice(6));
        }
      }
    }
  }
}

export async function fetchCLIAgents(serverAddr: string): Promise<CLIAgentStatus[]> {
  const resp = await fetch(`${serverAddr}/api/cli-agents`);
  if (!resp.ok) {
    throw new Error(await resp.text());
  }
  const data = (await resp.json()) as CLIAgentsResponse;
  return data.agents ?? [];
}

export async function installCLIAgent(
  serverAddr: string,
  cliType: string,
  onProgress: (message: string) => void
): Promise<void> {
  const resp = await fetch(`${serverAddr}/api/cli-agents/${encodeURIComponent(cliType)}/install`, {
    method: 'POST',
  });
  if (!resp.ok) {
    throw new Error(await resp.text());
  }
  let error: string | null = null;
  await readSSEText(resp, (line) => {
    if (line.startsWith('ERROR: ')) {
      error = line.slice(7);
      onProgress(error);
      return;
    }
    if (line === 'DONE') {
      onProgress('Install complete');
      return;
    }
    onProgress(line);
  });
  if (error) {
    throw new Error(error);
  }
}

export async function fetchCLIAuthInfo(
  serverAddr: string,
  cliType: string
): Promise<CLIAuthInfo> {
  const resp = await fetch(`${serverAddr}/api/cli-agents/${encodeURIComponent(cliType)}/auth`);
  if (!resp.ok) {
    throw new Error(await resp.text());
  }
  return resp.json();
}

export async function saveCLIAPIKey(
  serverAddr: string,
  cliType: string,
  apiKey: string
): Promise<CLIAgentStatus> {
  const resp = await fetch(`${serverAddr}/api/cli-agents/${encodeURIComponent(cliType)}/auth`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ api_key: apiKey }),
  });
  if (!resp.ok) {
    throw new Error(await resp.text());
  }
  return resp.json();
}

export async function probeCLIAgent(
  serverAddr: string,
  cliType: string
): Promise<CLIAgentStatus> {
  const resp = await fetch(`${serverAddr}/api/cli-agents/${encodeURIComponent(cliType)}/probe`, {
    method: 'POST',
  });
  if (!resp.ok) {
    throw new Error(await resp.text());
  }
  return resp.json();
}

export async function activateCLIAgent(
  serverAddr: string,
  cliType: string
): Promise<CLIActivateResponse> {
  const resp = await fetch(`${serverAddr}/api/cli-agents/${encodeURIComponent(cliType)}/activate`, {
    method: 'POST',
  });
  const data = (await resp.json()) as CLIActivateResponse;
  if (!resp.ok) {
    throw new Error(data.error || `Activate failed (${resp.status})`);
  }
  return data;
}

export function cliReadyStatus(agent: CLIAgentStatus): 'ready' | 'needs_install' | 'needs_auth' | 'unknown' {
  if (!agent.installed) return 'needs_install';
  if (agent.auth_state === 'needs_auth') return 'needs_auth';
  if (agent.auth_state === 'unknown') return 'unknown';
  return 'ready';
}

export function statusLabel(agent: CLIAgentStatus): string {
  const state = cliReadyStatus(agent);
  switch (state) {
    case 'ready':
      return 'Ready';
    case 'needs_install':
      return 'Not installed';
    case 'needs_auth':
      return 'Needs sign-in';
    default:
      return agent.installed ? 'Installed' : 'Not installed';
  }
}

export function statusDotClass(agent: CLIAgentStatus): string {
  const state = cliReadyStatus(agent);
  if (state === 'ready') return 'bg-green-400';
  if (state === 'needs_auth' || state === 'unknown') return 'bg-yellow-400';
  return 'bg-red-400';
}
