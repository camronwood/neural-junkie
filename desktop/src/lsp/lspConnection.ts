import { getHubBaseURL } from '../config/hubUrl';

export type LspLang = 'go' | 'rust' | 'python';

export interface LspPosition {
  line: number;
  character: number;
}

export interface PublishDiagnosticsParams {
  uri: string;
  diagnostics: Array<{
    range: {
      start: { line: number; character: number };
      end: { line: number; character: number };
    };
    message: string;
    severity?: number;
  }>;
}

type PendingRequest = {
  resolve: (value: unknown) => void;
  reject: (err: Error) => void;
};

let nextId = 1;

function wsBase(): string {
  return getHubBaseURL().replace(/^http/, 'ws');
}

export class LspConnection {
  private ws: WebSocket | null = null;
  private pending = new Map<number, PendingRequest>();
  private notifyHandlers = new Map<string, Array<(params: unknown) => void>>();
  private openDocs = new Map<string, { version: number; languageId: string }>();
  private changeTimers = new Map<string, ReturnType<typeof setTimeout>>();

  constructor(
    private workspaceId: string,
    private lang: LspLang
  ) {}

  async connect(): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) return;
    const url = `${wsBase()}/api/lsp/ws?workspace=${encodeURIComponent(this.workspaceId)}&lang=${encodeURIComponent(this.lang)}`;
    await new Promise<void>((resolve, reject) => {
      const ws = new WebSocket(url);
      ws.onopen = () => {
        this.ws = ws;
        resolve();
      };
      ws.onerror = () => reject(new Error('LSP websocket failed'));
      ws.onmessage = (ev) => this.onMessage(String(ev.data));
      ws.onclose = () => {
        this.ws = null;
      };
    });
  }

  disconnect(): void {
    for (const t of this.changeTimers.values()) clearTimeout(t);
    this.changeTimers.clear();
    this.ws?.close();
    this.ws = null;
    this.pending.clear();
  }

  onNotification(method: string, handler: (params: unknown) => void): () => void {
    const list = this.notifyHandlers.get(method) ?? [];
    list.push(handler);
    this.notifyHandlers.set(method, list);
    return () => {
      const next = (this.notifyHandlers.get(method) ?? []).filter((h) => h !== handler);
      this.notifyHandlers.set(method, next);
    };
  }

  private onMessage(raw: string) {
    let msg: { id?: number; method?: string; params?: unknown; result?: unknown; error?: { message?: string } };
    try {
      msg = JSON.parse(raw);
    } catch {
      return;
    }
    if (msg.id != null && msg.id !== 0) {
      const p = this.pending.get(msg.id);
      if (!p) return;
      this.pending.delete(msg.id);
      if (msg.error) p.reject(new Error(msg.error.message ?? 'LSP error'));
      else p.resolve(msg.result);
      return;
    }
    if (msg.method) {
      for (const h of this.notifyHandlers.get(msg.method) ?? []) {
        h(msg.params);
      }
    }
  }

  private send(payload: Record<string, unknown>) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    this.ws.send(JSON.stringify(payload));
  }

  notify(method: string, params: unknown) {
    this.send({ jsonrpc: '2.0', method, params });
  }

  request<T>(method: string, params: unknown): Promise<T> {
    const id = nextId++;
    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, {
        resolve: (v) => resolve(v as T),
        reject,
      });
      this.send({ jsonrpc: '2.0', id, method, params });
    });
  }

  didOpen(uri: string, languageId: string, text: string) {
    const version = 1;
    this.openDocs.set(uri, { version, languageId });
    this.notify('textDocument/didOpen', {
      textDocument: { uri, languageId, version, text },
    });
  }

  didChange(uri: string, text: string) {
    const doc = this.openDocs.get(uri);
    if (!doc) return;
    const prev = this.changeTimers.get(uri);
    if (prev) clearTimeout(prev);
    this.changeTimers.set(
      uri,
      setTimeout(() => {
        doc.version += 1;
        this.notify('textDocument/didChange', {
          textDocument: { uri, version: doc.version },
          contentChanges: [{ text }],
        });
      }, 200)
    );
  }

  didClose(uri: string) {
    const t = this.changeTimers.get(uri);
    if (t) clearTimeout(t);
    this.changeTimers.delete(uri);
    this.openDocs.delete(uri);
    this.notify('textDocument/didClose', {
      textDocument: { uri },
    });
  }
}

const pool = new Map<string, LspConnection>();

export function getLspConnection(workspaceId: string, lang: LspLang): LspConnection {
  const key = `${workspaceId}:${lang}`;
  let conn = pool.get(key);
  if (!conn) {
    conn = new LspConnection(workspaceId, lang);
    pool.set(key, conn);
  }
  return conn;
}

export async function lspRequest<T>(
  workspaceId: string,
  lang: LspLang,
  method: string,
  params: unknown,
  opts?: { uri?: string; text?: string; languageId?: string }
): Promise<T> {
  const res = await fetch(`${getHubBaseURL()}/api/lsp/request`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      workspace_id: workspaceId,
      lang,
      method,
      params,
      uri: opts?.uri,
      text: opts?.text,
    }),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json() as Promise<T>;
}
