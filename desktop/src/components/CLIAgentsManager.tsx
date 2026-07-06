import { useCallback, useEffect, useMemo, useState } from 'react';
import { XTerminal } from './XTerminal';
import { terminalAPI } from '../api/terminalAPI';
import {
  activateCLIAgent,
  fetchCLIAgents,
  installCLIAgent,
  probeCLIAgent,
  saveCLIAPIKey,
  statusDotClass,
  statusLabel,
  type CLIAgentStatus,
} from '../api/cliAgentsAPI';
import { isTauriRuntime } from '../utils/promptAttachments';

interface CLIAgentsManagerProps {
  serverAddr: string;
  /** When true, only show featured CLI types with no expand control (e.g. setup wizard). */
  featuredOnly?: boolean;
  /** When true, show featured agents first and let the user expand to see the rest. */
  expandable?: boolean;
  compact?: boolean;
  onAgentActivated?: () => void;
}

const FEATURED_TYPES = ['cursor', 'claude', 'gemini'];

function LoginTerminal({
  cliType,
  loginCommand,
  onDone,
}: {
  cliType: string;
  loginCommand: string;
  onDone: () => void;
}) {
  const sessionId = useMemo(() => `cli-login-${cliType}-${Date.now()}`, [cliType]);
  const [booted, setBooted] = useState(false);

  useEffect(() => {
    if (booted) return;
    const timer = window.setTimeout(async () => {
      try {
        await terminalAPI.writePtySession(sessionId, `${loginCommand}\n`);
        setBooted(true);
      } catch {
        // PTY may not be ready yet; user can type manually.
      }
    }, 900);
    return () => window.clearTimeout(timer);
  }, [booted, loginCommand, sessionId]);

  return (
    <div className="space-y-2">
      <div className="text-xs text-gray-400">
        Complete sign-in below, then click <strong className="text-gray-300">Check status</strong>.
      </div>
      <div className="h-40 rounded border border-gray-700 bg-black overflow-hidden">
        <XTerminal sessionId={sessionId} isActive />
      </div>
      <button
        type="button"
        onClick={() => void onDone()}
        className="px-3 py-1 text-xs bg-gray-700 text-gray-200 rounded hover:bg-gray-600"
      >
        Check status
      </button>
    </div>
  );
}

function CLIAgentCard({
  agent,
  serverAddr,
  compact,
  onRefresh,
  onAgentActivated,
}: {
  agent: CLIAgentStatus;
  serverAddr: string;
  compact?: boolean;
  onRefresh: () => void;
  onAgentActivated?: () => void;
}) {
  const [installing, setInstalling] = useState(false);
  const [installLog, setInstallLog] = useState('');
  const [busy, setBusy] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [showLogin, setShowLogin] = useState(false);
  const [apiKey, setApiKey] = useState('');
  const [showApiKey, setShowApiKey] = useState(false);
  const tauri = isTauriRuntime();

  async function handleInstall() {
    setInstalling(true);
    setInstallLog('');
    setError(null);
    try {
      await installCLIAgent(serverAddr, agent.type, (msg) => setInstallLog(msg));
      await onRefresh();
      const updated = await probeCLIAgent(serverAddr, agent.type);
      if (updated.auth_state === 'authed' || updated.auth_state === 'not_applicable') {
        await handleActivate();
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setInstalling(false);
    }
  }

  async function handleActivate() {
    setBusy('Activating…');
    setError(null);
    try {
      await activateCLIAgent(serverAddr, agent.type);
      onAgentActivated?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy('');
    }
  }

  async function handleProbe() {
    setBusy('Checking…');
    setError(null);
    try {
      const updated = await probeCLIAgent(serverAddr, agent.type);
      await onRefresh();
      if (
        updated.installed &&
        (updated.auth_state === 'authed' ||
          updated.auth_state === 'not_applicable' ||
          updated.auth_state === 'unknown')
      ) {
        setShowLogin(false);
        await handleActivate();
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy('');
    }
  }

  async function handleSaveAPIKey() {
    if (!apiKey.trim()) return;
    setBusy('Saving key…');
    setError(null);
    try {
      await saveCLIAPIKey(serverAddr, agent.type, apiKey.trim());
      setApiKey('');
      setShowApiKey(false);
      await onRefresh();
      await handleActivate();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy('');
    }
  }

  const canUseAPIKey = (agent.auth?.env_vars?.length ?? 0) > 0;

  return (
    <div className={`rounded-lg border border-gray-700 bg-gray-800/60 ${compact ? 'p-3' : 'p-4'} space-y-3`}>
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <span className={`w-2 h-2 rounded-full shrink-0 ${statusDotClass(agent)}`} />
            <h4 className="text-sm font-semibold text-white">{agent.name}</h4>
            <span className="text-xs text-gray-500">{statusLabel(agent)}</span>
          </div>
          {!compact && (
            <p className="mt-1 text-xs text-gray-500">
              {agent.installed
                ? `${agent.binary ?? 'binary'}${agent.version ? ` · ${agent.version}` : ''}`
                : agent.install_hint}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={() => void handleProbe()}
          disabled={!!busy || installing}
          className="px-2 py-1 text-xs text-gray-400 hover:text-gray-200"
        >
          Refresh
        </button>
      </div>

      {agent.missing_prereqs && agent.missing_prereqs.length > 0 && (
        <p className="text-xs text-amber-400">
          Missing prerequisites: {agent.missing_prereqs.join(', ')}
        </p>
      )}

      {!agent.installed && agent.can_install && (
        <div className="space-y-2">
          <button
            type="button"
            onClick={() => void handleInstall()}
            disabled={installing || (agent.missing_prereqs?.length ?? 0) > 0}
            className="px-3 py-1.5 text-xs bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-50"
          >
            {installing ? 'Installing…' : `Install ${agent.name}`}
          </button>
          {agent.install?.command && (
            <p className="text-xs text-gray-500 font-mono break-all">{agent.install.command}</p>
          )}
        </div>
      )}

      {!agent.installed && !agent.can_install && agent.install_hint && (
        <p className="text-xs text-gray-500">{agent.install_hint}</p>
      )}

      {installLog && (
        <pre className="text-xs text-gray-400 whitespace-pre-wrap max-h-24 overflow-y-auto">{installLog}</pre>
      )}

      {agent.installed && agent.auth_state === 'needs_auth' && (
        <div className="space-y-2">
          {agent.login_command && tauri && (
            <>
              {!showLogin ? (
                <button
                  type="button"
                  onClick={() => setShowLogin(true)}
                  className="px-3 py-1.5 text-xs bg-indigo-600 text-white rounded hover:bg-indigo-500"
                >
                  Sign in ({agent.login_command})
                </button>
              ) : (
                <LoginTerminal
                  cliType={agent.type}
                  loginCommand={agent.login_command}
                  onDone={() => void handleProbe()}
                />
              )}
            </>
          )}

          {agent.login_command && !tauri && (
            <div className="text-xs text-gray-400 space-y-1">
              <p>Run in your terminal:</p>
              <code className="block p-2 rounded bg-gray-900 text-gray-200">{agent.login_command}</code>
              <button
                type="button"
                onClick={() => void handleProbe()}
                className="px-3 py-1 text-xs bg-gray-700 text-gray-200 rounded hover:bg-gray-600"
              >
                I signed in — check status
              </button>
            </div>
          )}

          {canUseAPIKey && (
            <div className="space-y-2">
              {!showApiKey ? (
                <button
                  type="button"
                  onClick={() => setShowApiKey(true)}
                  className="px-3 py-1 text-xs text-gray-300 underline hover:text-white"
                >
                  Or paste API key ({agent.auth?.env_vars?.[0]})
                </button>
              ) : (
                <div className="flex flex-col sm:flex-row gap-2">
                  <input
                    type="password"
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    placeholder={agent.auth?.env_vars?.[0] ?? 'API key'}
                    className="flex-1 px-3 py-1.5 text-xs bg-gray-900 border border-gray-700 rounded text-white font-mono"
                  />
                  <button
                    type="button"
                    onClick={() => void handleSaveAPIKey()}
                    disabled={!apiKey.trim() || !!busy}
                    className="px-3 py-1.5 text-xs bg-green-700 text-white rounded hover:bg-green-600 disabled:opacity-50"
                  >
                    Save key
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {agent.installed &&
        (agent.auth_state === 'authed' ||
          agent.auth_state === 'not_applicable' ||
          agent.auth_state === 'unknown') && (
          <button
            type="button"
            onClick={() => void handleActivate()}
            disabled={!!busy}
            className="px-3 py-1.5 text-xs bg-emerald-700/80 text-emerald-100 rounded hover:bg-emerald-700 disabled:opacity-50"
          >
            {busy || 'Activate in #general'}
          </button>
        )}

      {busy && <p className="text-xs text-gray-400">{busy}</p>}
      {error && <p className="text-xs text-red-400">{error}</p>}
    </div>
  );
}

function AgentCardGrid({
  agents,
  serverAddr,
  compact,
  onRefresh,
  onAgentActivated,
}: {
  agents: CLIAgentStatus[];
  serverAddr: string;
  compact?: boolean;
  onRefresh: () => void;
  onAgentActivated?: () => void;
}) {
  if (agents.length === 0) return null;

  return (
    <div className={`grid gap-3 ${compact ? 'grid-cols-1' : 'grid-cols-1 xl:grid-cols-2'}`}>
      {agents.map((agent) => (
        <CLIAgentCard
          key={agent.type}
          agent={agent}
          serverAddr={serverAddr}
          compact={compact}
          onRefresh={onRefresh}
          onAgentActivated={onAgentActivated}
        />
      ))}
    </div>
  );
}

export function CLIAgentsManager({
  serverAddr,
  featuredOnly = false,
  expandable = true,
  compact = false,
  onAgentActivated,
}: CLIAgentsManagerProps) {
  const [agents, setAgents] = useState<CLIAgentStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAll, setShowAll] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const list = await fetchCLIAgents(serverAddr);
      setAgents(list);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [serverAddr]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const { featured, other } = useMemo(() => {
    const featuredList: CLIAgentStatus[] = [];
    const otherList: CLIAgentStatus[] = [];
    for (const agent of agents) {
      if (FEATURED_TYPES.includes(agent.type)) {
        featuredList.push(agent);
      } else {
        otherList.push(agent);
      }
    }
    return { featured: featuredList, other: otherList };
  }, [agents]);

  const showExpandControl = expandable && !featuredOnly && other.length > 0;
  const showOther = !featuredOnly && (!expandable || showAll);

  return (
    <div className="space-y-3">
      {!compact && (
        <>
          <h3 className="text-sm font-semibold text-gray-300">CLI agents</h3>
          <p className="text-xs text-gray-500 leading-relaxed">
            Install and sign in to CLI coding agents from here. Cursor, Claude Code, and Gemini are
            recommended; expand below for Copilot, Codex, Aider, and more. Ready agents join{' '}
            <code className="text-gray-400">#general</code> when activated.
          </p>
        </>
      )}

      {loading && <div className="text-sm text-gray-500">Loading CLI status…</div>}
      {error && <div className="text-sm text-red-400">{error}</div>}

      {featuredOnly ? (
        <AgentCardGrid
          agents={featured}
          serverAddr={serverAddr}
          compact={compact}
          onRefresh={refresh}
          onAgentActivated={onAgentActivated}
        />
      ) : (
        <div className="space-y-4">
          {featured.length > 0 && (
            <div className="space-y-2">
              {!compact && expandable && (
                <h4 className="text-xs font-medium uppercase tracking-wide text-gray-500">
                  Recommended
                </h4>
              )}
              <AgentCardGrid
                agents={featured}
                serverAddr={serverAddr}
                compact={compact}
                onRefresh={refresh}
                onAgentActivated={onAgentActivated}
              />
            </div>
          )}

          {showExpandControl && !showAll && (
            <button
              type="button"
              onClick={() => setShowAll(true)}
              className="w-full px-3 py-2 text-xs text-gray-300 border border-gray-700 rounded-lg hover:bg-gray-800/80 hover:text-white transition-colors"
            >
              Show {other.length} more CLI agent{other.length === 1 ? '' : 's'} (Copilot, Codex,
              Aider, OpenCode, …)
            </button>
          )}

          {showOther && other.length > 0 && (
            <div className="space-y-2">
              {!compact && expandable && (
                <div className="flex items-center justify-between gap-3">
                  <h4 className="text-xs font-medium uppercase tracking-wide text-gray-500">
                    More CLI agents
                  </h4>
                  {showExpandControl && (
                    <button
                      type="button"
                      onClick={() => setShowAll(false)}
                      className="text-xs text-gray-400 hover:text-gray-200"
                    >
                      Show fewer
                    </button>
                  )}
                </div>
              )}
              <AgentCardGrid
                agents={other}
                serverAddr={serverAddr}
                compact={compact}
                onRefresh={refresh}
                onAgentActivated={onAgentActivated}
              />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
