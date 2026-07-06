import { getHubBaseURL } from '../../config/hubUrl';

export type BrowserViewport = { width: number; height: number };

async function browserPost<T>(path: string, body: Record<string, unknown>): Promise<T> {
  const res = await fetch(`${getHubBaseURL()}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const data = (await res.json()) as T & { error?: string };
  if (!res.ok) {
    throw new Error(data.error || res.statusText);
  }
  if (typeof data === 'object' && data !== null && 'error' in data && data.error) {
    throw new Error(data.error);
  }
  return data;
}

export async function browserA11yAudit(url: string) {
  return browserPost<{
    violations: Array<{
      id: string;
      impact: string;
      description: string;
      help: string;
      help_url: string;
      node_count: number;
    }>;
    violation_count: number;
    url: string;
  }>('/api/browser/a11y-audit', { url });
}

export async function browserMetrics(url: string) {
  return browserPost<{
    metrics: Record<string, number>;
    url: string;
  }>('/api/browser/metrics', { url });
}

export async function browserVisualDiff(
  url: string,
  baselinePath: string,
  workspaceId: string,
  viewport?: BrowserViewport | null,
) {
  const body: Record<string, unknown> = {
    url,
    baseline_path: baselinePath,
    workspace_id: workspaceId,
  };
  if (viewport) body.viewport = viewport;
  return browserPost<{
    match_pct: number;
    baseline_exists: boolean;
    baseline_path: string;
    diff_png_b64?: string;
  }>('/api/browser/visual-diff', body);
}

export async function browserScreenshot(url: string, viewport?: BrowserViewport | null, fullPage = false) {
  const body: Record<string, unknown> = { url, full_page: fullPage };
  if (viewport) body.viewport = viewport;
  return browserPost<{ png_b64: string; width: number; height: number; url: string }>(
    '/api/browser/screenshot',
    body,
  );
}

export async function browserAcceptBaseline(
  url: string,
  baselinePath: string,
  workspaceId: string,
  viewport?: BrowserViewport | null,
) {
  const body: Record<string, unknown> = {
    url,
    baseline_path: baselinePath,
    workspace_id: workspaceId,
    full_page: true,
  };
  if (viewport) body.viewport = viewport;
  return browserPost<{ ok: boolean; baseline_path: string; bytes: number }>(
    '/api/browser/accept-baseline',
    body,
  );
}

export async function browserPickElement(url: string, x: number, y: number, viewport?: BrowserViewport | null) {
  const body: Record<string, unknown> = { url, x, y };
  if (viewport) body.viewport = viewport;
  return browserPost<{
    selector: string;
    outer_html: string;
    tag: string;
    computed_styles: Record<string, string>;
  }>('/api/browser/pick-element', body);
}
