import { SCAN_SUMMARY_METADATA_FILE } from './scanSummary';

/** Common scan export subfolder names (relative to analysis root). */
export const SCAN_LINK_CANDIDATE_DIRS = [
  'scan-export',
  'scan_export',
  'scan',
  'summary',
] as const;

/** Normalize user-entered scan link path for storage and API requests. */
export function normalizeScanLinkInput(input: string): string {
  let s = input.trim().replace(/\\/g, '/');
  // User pasted full metadata path or path ending with /
  s = s.replace(/\/imageMetadata\.json\/?$/i, '');
  s = s.replace(/\/+$/, '');
  // Collapse accidental double metadata segment
  if (s.endsWith('/imageMetadata.json')) {
    s = s.slice(0, -'/imageMetadata.json'.length);
  }
  return s.replace(/\/+$/, '');
}

export function scanMetadataRelativePath(linkedScanDir: string): string {
  const dir = normalizeScanLinkInput(linkedScanDir);
  return dir ? `${dir}/${SCAN_SUMMARY_METADATA_FILE}` : SCAN_SUMMARY_METADATA_FILE;
}

/** Candidate relative directories to probe for imageMetadata.json. */
export function scanLinkCandidateDirs(analysisDir: string): string[] {
  const base = analysisDir.replace(/[/\\]+$/, '');
  const out: string[] = [];
  const add = (p: string) => {
    const n = normalizeScanLinkInput(p);
    if (n !== undefined && !out.includes(n)) out.push(n);
  };
  add(base);
  for (const name of SCAN_LINK_CANDIDATE_DIRS) {
    add(base ? `${base}/${name}` : name);
  }
  return out;
}

export type ScanLinkProbeResult =
  | { ok: true; linkedScanDir: string; metadataPath: string }
  | { ok: false; metadataPath: string; reason: string };

/** Probe which candidate dir contains imageMetadata.json (via fetch). */
export async function probeLinkedScanDir(
  workspaceId: string,
  analysisDir: string,
  fetchContent: (workspaceId: string, path: string) => Promise<string>
): Promise<string> {
  for (const cand of scanLinkCandidateDirs(analysisDir)) {
    const metaPath = scanMetadataRelativePath(cand);
    try {
      const raw = await fetchContent(workspaceId, metaPath);
      if (raw && typeof raw === 'string' && raw.trim().startsWith('{')) {
        return normalizeScanLinkInput(cand);
      }
    } catch {
      /* try next */
    }
  }
  return '';
}

/** Validate a user-provided scan link before saving on the tab. */
export async function validateScanLink(
  workspaceId: string,
  linkedScanDir: string,
  fetchContent: (workspaceId: string, path: string) => Promise<string>
): Promise<ScanLinkProbeResult> {
  const normalized = normalizeScanLinkInput(linkedScanDir);
  const metadataPath = scanMetadataRelativePath(normalized);
  try {
    const raw = await fetchContent(workspaceId, metadataPath);
    if (!raw || typeof raw !== 'string' || !raw.trim().startsWith('{')) {
      return {
        ok: false,
        metadataPath,
        reason: `Expected JSON at ${metadataPath}, but the file was empty or not JSON.`,
      };
    }
    return { ok: true, linkedScanDir: normalized, metadataPath };
  } catch (err) {
    const detail = err instanceof Error ? err.message : String(err);
    return {
      ok: false,
      metadataPath,
      reason: detail.includes('Forbidden')
        ? `Path is outside the workspace. Use a folder relative to the workspace root (e.g. scan-export), not ../ or an absolute path.`
        : detail.includes('Not Found') || detail.includes('no such file')
          ? `Scan metadata not found at ${metadataPath}. Link the folder that contains imageMetadata.json (e.g. scan-export), not the file itself.`
          : `Could not read ${metadataPath}: ${detail}`,
    };
  }
}
