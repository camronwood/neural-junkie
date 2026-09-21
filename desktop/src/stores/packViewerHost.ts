import type { ResolvedCapability } from '../api/chatAPI';
import { matchFileViewer, NJ_VIEWER } from './packCapabilityRegistry';
import { useEditorStore } from './editorStore';

/** Trusted host viewers — packs declare viewer ids; React stays in core. */
const TRUSTED_VIEWERS = new Set<string>([
  NJ_VIEWER.STRUCTURE,
  NJ_VIEWER.CAD,
  NJ_VIEWER.MUSIC,
  NJ_VIEWER.ARENA,
  NJ_VIEWER.HTML,
  NJ_VIEWER.SCAN_SUMMARY,
  NJ_VIEWER.SCAN_ANALYSIS,
]);

export type OpenPackViewerResult = 'opened' | 'untrusted' | 'none';

/**
 * Open a pack-declared file viewer by registry match (glob → viewer id).
 * Returns whether a trusted viewer handled the path.
 */
export async function openPackViewer(
  registry: ResolvedCapability[],
  workspaceId: string,
  path: string,
  fetchContent: (workspaceId: string, path: string) => Promise<string>,
): Promise<OpenPackViewerResult> {
  const matched = matchFileViewer(registry, path);
  const viewerId = matched?.viewer?.trim();
  if (!viewerId) return 'none';
  if (!TRUSTED_VIEWERS.has(viewerId)) return 'untrusted';

  const store = useEditorStore.getState();
  switch (viewerId) {
    case NJ_VIEWER.STRUCTURE: {
      const content = await fetchContent(workspaceId, path);
      store.openStructureWorkbench(workspaceId, path, content);
      return 'opened';
    }
    case NJ_VIEWER.CAD: {
      const content = await fetchContent(workspaceId, path);
      store.openCadWorkbench(workspaceId, path, content);
      return 'opened';
    }
    case NJ_VIEWER.MUSIC: {
      const content = /\.json$/i.test(path) ? await fetchContent(workspaceId, path) : '';
      store.openMusicWorkbench(workspaceId, path, content);
      return 'opened';
    }
    case NJ_VIEWER.ARENA: {
      store.openArenaWorkbench(workspaceId, path);
      return 'opened';
    }
    case NJ_VIEWER.HTML: {
      const content = await fetchContent(workspaceId, path);
      store.openHtmlBrowser(workspaceId, path, content);
      return 'opened';
    }
    default:
      return 'none';
  }
}
