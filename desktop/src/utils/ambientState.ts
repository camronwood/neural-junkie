import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useDiagnosticsStore, type EditorDiagnostic } from '../stores/diagnosticsStore';
import { useEditorStore } from '../stores/editorStore';
import { useFileExplorerStore } from '../stores/fileExplorerStore';
import { useTerminalStore } from '../stores/terminalStore';

export const AMBIENT_STATE_METADATA_KEY = 'ambient_state';
export const CLIENT_AMBIENT_STATE_TARGET_BYTES = 8 * 1024;

export interface AmbientGitState {
  branch?: string;
  staged?: string[];
  unstaged?: string[];
  untracked?: string[];
}

export interface AmbientState {
  active_editor?: {
    path: string;
    cursor?: { line: number; column: number };
    selection?: { start_line: number; end_line: number; text?: string };
  };
  diagnostics?: EditorDiagnostic[];
  terminal?: { cwd?: string; failed_tail: string };
  git?: AmbientGitState;
  recent_edits?: Array<{ path: string; edited_at: number }>;
  truncated?: boolean;
}

const SENSITIVE_PATH_RE =
  /(?:^|[/\\])(?:\.env(?:\.|$)|id_(?:rsa|dsa|ecdsa|ed25519)(?:\.|$)|credentials?(?:\.|$)|secrets?(?:\.|$)|.*\.(?:pem|key|p12|pfx))|(?:^|[/\\])(?:\.aws|\.ssh)(?:[/\\]|$)/i;
const PRIVATE_KEY_RE =
  /-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z0-9 ]*PRIVATE KEY-----/g;
const SECRET_ASSIGNMENT_RE =
  /(\b(?:api[_-]?key|access[_-]?token|auth[_-]?token|client[_-]?secret|password|passwd|secret)\b\s*[:=]\s*)(["']?)[^\s"',;]+/gi;
const ANSI_RE = /\x1b(?:[@-_][0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\))/g;
const CONTROL_RE = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/g;

export function isSensitiveAmbientPath(path: string): boolean {
  return SENSITIVE_PATH_RE.test(path.replace(/\\/g, '/'));
}

export function sanitizeAmbientText(value: string): string {
  return value
    .replace(ANSI_RE, '')
    .replace(CONTROL_RE, '')
    .replace(PRIVATE_KEY_RE, '[REDACTED PRIVATE KEY]')
    .replace(SECRET_ASSIGNMENT_RE, '$1[REDACTED]');
}

export function ambientStateIsRelevant(message: string, ideCoding = false): boolean {
  const text = message.trim();
  if (!text) return false;
  return (
    ideCoding ||
    /\b(code|file|editor|selection|line|implement|fix|debug|error|warning|diagnostic|test|build|terminal|command|shell|git|commit|branch|diff|staged|workspace|repo)\b/i.test(
      text,
    ) ||
    /(?:^|[\s"'`(])(?:[./]?(?:[\w.-]+\/)+[\w.-]+\.[a-z0-9]+|[\w.-]+\.(?:go|rs|py|js|jsx|ts|tsx|json|ya?ml|toml|md))\b/i.test(
      text,
    )
  );
}

function jsonBytes(value: unknown): number {
  return new TextEncoder().encode(JSON.stringify(value)).length;
}

function boundedAmbientState(state: AmbientState): AmbientState | undefined {
  if (jsonBytes(state) <= CLIENT_AMBIENT_STATE_TARGET_BYTES) return state;
  const out: AmbientState = { ...state, truncated: true };
  if (out.diagnostics) out.diagnostics = out.diagnostics.slice(0, 12);
  if (out.recent_edits) out.recent_edits = out.recent_edits.slice(0, 8);
  if (out.terminal?.failed_tail) {
    out.terminal = { ...out.terminal, failed_tail: out.terminal.failed_tail.slice(-2048) };
  }
  if (out.active_editor?.selection?.text) {
    out.active_editor = {
      ...out.active_editor,
      selection: { ...out.active_editor.selection, text: out.active_editor.selection.text.slice(0, 1024) },
    };
  }
  for (const key of ['untracked', 'unstaged', 'staged'] as const) {
    if (out.git?.[key]) out.git[key] = out.git[key]!.slice(0, 20);
  }
  while (jsonBytes(out) > CLIENT_AMBIENT_STATE_TARGET_BYTES && (out.diagnostics?.length ?? 0) > 0) {
    out.diagnostics!.pop();
  }
  while (jsonBytes(out) > CLIENT_AMBIENT_STATE_TARGET_BYTES && (out.recent_edits?.length ?? 0) > 0) {
    out.recent_edits!.pop();
  }
  if (jsonBytes(out) > CLIENT_AMBIENT_STATE_TARGET_BYTES) delete out.terminal;
  if (jsonBytes(out) > CLIENT_AMBIENT_STATE_TARGET_BYTES) delete out.git;
  if (jsonBytes(out) > CLIENT_AMBIENT_STATE_TARGET_BYTES) delete out.active_editor?.selection?.text;
  return Object.keys(out).length > 1 || out.truncated !== true ? out : undefined;
}

export function buildAmbientState(
  message: string,
  options: { ideCoding?: boolean; git?: AmbientGitState } = {},
): AmbientState | undefined {
  if (!ambientStateIsRelevant(message, options.ideCoding)) return undefined;

  const editor = useEditorStore.getState();
  const activeTab = editor.tabs.find((tab) => tab.id === editor.activeTabId);
  const selection =
    activeTab && editor.activeSelection?.tabId === activeTab.id
      ? editor.activeSelection
      : null;
  const activeEditor = activeTab
    ? {
        path: activeTab.path,
        cursor: activeTab.cursorPosition,
        selection: selection
          ? {
              start_line: selection.startLine,
              end_line: selection.endLine,
              ...(!isSensitiveAmbientPath(activeTab.path)
                ? { text: sanitizeAmbientText(selection.text).slice(0, 2048) }
                : {}),
            }
          : undefined,
      }
    : undefined;

  const workspace = useFileExplorerStore.getState().workspaces.find(
    (item) => item.id === activeTab?.workspaceId,
  );
  const diagnostics = useDiagnosticsStore
    .getState()
    .allForWorkspace(workspace?.path)
    .slice(0, 40)
    .map((item) => ({
      ...item,
      message: sanitizeAmbientText(item.message).slice(0, 512),
    }));
  const terminalState = useTerminalStore.getState();
  const terminalTab = terminalState.tabs.find((tab) => tab.id === terminalState.activeTabId);
  const failedTail = terminalState.recentFailedTails[terminalState.activeTabId];

  const state: AmbientState = {
    active_editor: activeEditor,
    diagnostics: diagnostics.length > 0 ? diagnostics : undefined,
    terminal: failedTail
      ? {
          cwd: terminalTab?.cwd,
          failed_tail: sanitizeAmbientText(failedTail).slice(-4096),
        }
      : undefined,
    git: options.git,
    recent_edits:
      editor.recentEdits.length > 0
        ? editor.recentEdits.slice(0, 12).map((edit) => ({ path: edit.path, edited_at: edit.editedAt }))
        : undefined,
  };
  return boundedAmbientState(state);
}

export async function attachAmbientStateMetadata(
  metadata: Record<string, unknown> | undefined,
  message: string,
  ideCoding = false,
): Promise<Record<string, unknown> | undefined> {
  if (!ambientStateIsRelevant(message, ideCoding)) return metadata;
  let git: AmbientGitState | undefined;
  if (/\b(git|commit|branch|diff|staged|unstaged|untracked)\b/i.test(message)) {
    const explorer = useFileExplorerStore.getState();
    const workspace =
      explorer.workspaces.find((item) => item.id === explorer.activeWorkspaceId) ??
      explorer.workspaces[0];
    if (workspace?.id && workspace.is_git_repo) {
      try {
        const raw = (await new ChatAPI(getHubBaseURL()).getGitStatus(workspace.id)) as AmbientGitState;
        git = {
          branch: raw.branch,
          staged: raw.staged,
          unstaged: raw.unstaged,
          untracked: raw.untracked,
        };
      } catch {
        // Ambient context must never block a send.
      }
    }
  }
  const ambient = buildAmbientState(message, { ideCoding, git });
  if (!ambient) return metadata;
  return { ...(metadata ?? {}), [AMBIENT_STATE_METADATA_KEY]: ambient };
}
