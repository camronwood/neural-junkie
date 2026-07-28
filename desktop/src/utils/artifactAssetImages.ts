import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';

const ARTIFACT_ASSET_SRC =
  /(?:^|\/)api\/artifacts\/([^/]+)\/assets\/([^"'?\s]+)/i;

/**
 * Rewrite <img src="/api/artifacts/{id}/assets/{name}"> to hub data URLs
 * so Neural Canvas embedded images load with session auth.
 */
export async function resolveArtifactAssetImagesInHtml(
  html: string,
  fallbackArtifactId?: string,
): Promise<string> {
  if (!html.includes('/api/artifacts/') && !html.includes('api/artifacts/')) {
    return html;
  }
  const api = new ChatAPI(getHubBaseURL());
  const re = /<img\b([^>]*?)\ssrc=(["'])(.*?)\2/gi;
  const matches = [...html.matchAll(re)];
  if (matches.length === 0) return html;

  let out = html;
  for (const match of matches) {
    const full = match[0];
    const attrs = match[1] ?? '';
    const quote = match[2] ?? '"';
    const src = (match[3] ?? '').trim();
    const parsed = parseArtifactAssetSrc(src, fallbackArtifactId);
    if (!parsed) continue;
    try {
      const dataUrl = await api.fetchArtifactAssetDataUrl(parsed.artifactId, parsed.name);
      if (!dataUrl) continue;
      out = out.replace(full, `<img${attrs} src=${quote}${dataUrl}${quote}`);
    } catch {
      // leave original src
    }
  }
  return out;
}

export function parseArtifactAssetSrc(
  src: string,
  fallbackArtifactId?: string,
): { artifactId: string; name: string } | null {
  const trimmed = src.trim();
  const m = trimmed.match(ARTIFACT_ASSET_SRC);
  if (m) {
    return { artifactId: decodeURIComponent(m[1]), name: decodeURIComponent(m[2]) };
  }
  if (fallbackArtifactId && /^[A-Za-z0-9][A-Za-z0-9._-]{0,255}$/.test(trimmed) && !trimmed.includes('/')) {
    return { artifactId: fallbackArtifactId, name: trimmed };
  }
  return null;
}
