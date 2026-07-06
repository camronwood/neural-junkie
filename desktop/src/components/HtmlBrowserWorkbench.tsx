import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useToastStore } from '../stores/toastStore';
import { useEditorStore } from '../stores/editorStore';
import { usePacksStore } from '../stores/packsStore';
import { BrowserA11yPanel } from './browser/BrowserA11yPanel';
import { BrowserPerfPanel } from './browser/BrowserPerfPanel';
import { BrowserResponsiveToolbar, viewportForPreset, type ViewportPreset } from './browser/BrowserResponsiveToolbar';
import { BrowserVisualDiffPanel } from './browser/BrowserVisualDiffPanel';
import { browserPickElement } from './browser/browserSidecarApi';

const api = new ChatAPI(getHubBaseURL());

interface HtmlBrowserWorkbenchProps {
  workspaceId: string;
  htmlPath: string;
  initialContent?: string;
  initialUrl?: string;
  tabId: string;
}

type QAPanel = 'a11y' | 'perf' | 'visual' | null;

export function HtmlBrowserWorkbench({
  workspaceId,
  htmlPath,
  initialContent = '',
  initialUrl = '',
  tabId,
}: HtmlBrowserWorkbenchProps) {
  const hubBase = getHubBaseURL();
  const normalizedPath = htmlPath.replace(/\\/g, '/').replace(/^\/+/, '');
  const [htmlContent, setHtmlContent] = useState(initialContent);
  const [urlInput, setUrlInput] = useState(initialUrl);
  const [previewMode, setPreviewMode] = useState<'file' | 'url'>(initialUrl.trim() ? 'url' : 'file');
  const [reloadKey, setReloadKey] = useState(0);
  const [loadedPath, setLoadedPath] = useState<string | null>(null);
  const [viewportPreset, setViewportPreset] = useState<ViewportPreset>('desktop');
  const [qaPanel, setQAPanel] = useState<QAPanel>(null);
  const [pickMode, setPickMode] = useState(false);
  const [pickedElement, setPickedElement] = useState<string | null>(null);
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const { addToast } = useToastStore();
  const hasBrowserSidecar = usePacksStore((s) => s.hasCapability('browser-sidecar'));

  useEffect(() => {
    let cancelled = false;
    setLoadedPath(null);
    void (async () => {
      try {
        const diskContent = await api.fetchFileContent(workspaceId, normalizedPath);
        if (cancelled) return;
        setHtmlContent(diskContent);
        setLoadedPath(normalizedPath);
        useEditorStore.getState().updateTabContent(tabId, diskContent);
      } catch (err) {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : 'Failed to load HTML file';
        addToast({ type: 'error', title: 'HTML preview', message });
        setHtmlContent(initialContent);
        setLoadedPath(normalizedPath);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [tabId, normalizedPath, workspaceId, initialContent, addToast]);

  const handleEditorChange = useCallback(
    (value: string | undefined) => {
      const next = value ?? '';
      setHtmlContent(next);
      useEditorStore.getState().updateTabContent(tabId, next);
      useEditorStore.getState().markTabDirty(tabId, true);
    },
    [tabId],
  );

  const filePreviewSrc = useMemo(() => {
    const params = new URLSearchParams({ workspace: workspaceId, path: normalizedPath });
    return `${hubBase}/api/workspace-preview?${params.toString()}`;
  }, [hubBase, workspaceId, normalizedPath, reloadKey]);

  const iframeSrc = previewMode === 'url' && urlInput.trim() ? urlInput.trim() : filePreviewSrc;
  const previewUrlForSidecar = iframeSrc;
  const viewport = viewportForPreset(viewportPreset);
  const isSameOriginPreview = previewMode === 'file';

  const editorReady = loadedPath === normalizedPath;

  const handleIframeClick = useCallback(
    async (event: React.MouseEvent<HTMLIFrameElement>) => {
      if (!pickMode || !hasBrowserSidecar) return;
      const iframe = iframeRef.current;
      if (!iframe) return;
      const rect = iframe.getBoundingClientRect();
      const scaleX = (viewport?.width ?? rect.width) / rect.width;
      const scaleY = (viewport?.height ?? rect.height) / rect.height;
      const x = (event.clientX - rect.left) * scaleX;
      const y = (event.clientY - rect.top) * scaleY;

      if (isSameOriginPreview) {
        try {
          const doc = iframe.contentDocument;
          const el = doc?.elementFromPoint(x / scaleX, y / scaleY);
          if (el) {
            setPickedElement(
              `Selected \`${el.tagName.toLowerCase()}\`:\n\`\`\`html\n${el.outerHTML.slice(0, 2000)}\n\`\`\``,
            );
            setPickMode(false);
            return;
          }
        } catch {
          // fall through to sidecar
        }
      }

      try {
        const result = await browserPickElement(previewUrlForSidecar, x, y, viewport);
        setPickedElement(
          `Selected \`${result.selector}\`:\n\`\`\`html\n${result.outer_html.slice(0, 2000)}\n\`\`\``,
        );
        setPickMode(false);
      } catch (err) {
        addToast({
          type: 'error',
          title: 'DOM picker',
          message: err instanceof Error ? err.message : 'Pick failed',
        });
      }
    },
    [pickMode, hasBrowserSidecar, viewport, isSameOriginPreview, previewUrlForSidecar, addToast],
  );

  const copyPickedToClipboard = async () => {
    if (!pickedElement) return;
    try {
      await navigator.clipboard.writeText(pickedElement);
      addToast({ type: 'success', title: 'Copied', message: 'Element context copied for chat' });
    } catch {
      addToast({ type: 'error', title: 'Copy failed', message: 'Could not copy to clipboard' });
    }
  };

  const previewFrameStyle =
    viewport && viewportPreset !== 'full'
      ? { width: viewport.width, height: viewport.height, maxWidth: '100%' }
      : { width: '100%', height: '100%' };

  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2 border-b border-slack-border pb-2">
        <div className="flex rounded-md border border-slack-border overflow-hidden text-xs">
          <button
            type="button"
            onClick={() => setPreviewMode('file')}
            className={`px-3 py-1.5 ${previewMode === 'file' ? 'bg-teal-600 text-white' : 'bg-slack-bgHover text-slack-textMuted'}`}
          >
            Workspace file
          </button>
          <button
            type="button"
            onClick={() => setPreviewMode('url')}
            className={`px-3 py-1.5 ${previewMode === 'url' ? 'bg-teal-600 text-white' : 'bg-slack-bgHover text-slack-textMuted'}`}
          >
            Dev server URL
          </button>
        </div>
        <BrowserResponsiveToolbar preset={viewportPreset} onPresetChange={setViewportPreset} />
        {hasBrowserSidecar && (
          <>
            <button
              type="button"
              onClick={() => setQAPanel((p) => (p === 'a11y' ? null : 'a11y'))}
              className={`rounded border border-slack-border px-2 py-1 text-xs ${qaPanel === 'a11y' ? 'bg-teal-600 text-white' : 'hover:bg-slack-bgHover'}`}
            >
              A11y
            </button>
            <button
              type="button"
              onClick={() => setQAPanel((p) => (p === 'perf' ? null : 'perf'))}
              className={`rounded border border-slack-border px-2 py-1 text-xs ${qaPanel === 'perf' ? 'bg-teal-600 text-white' : 'hover:bg-slack-bgHover'}`}
            >
              Perf
            </button>
            <button
              type="button"
              onClick={() => setQAPanel((p) => (p === 'visual' ? null : 'visual'))}
              className={`rounded border border-slack-border px-2 py-1 text-xs ${qaPanel === 'visual' ? 'bg-teal-600 text-white' : 'hover:bg-slack-bgHover'}`}
            >
              Visual
            </button>
            <button
              type="button"
              onClick={() => setPickMode((v) => !v)}
              className={`rounded border border-slack-border px-2 py-1 text-xs ${pickMode ? 'bg-amber-600 text-white' : 'hover:bg-slack-bgHover'}`}
            >
              {pickMode ? 'Picking…' : 'Pick element'}
            </button>
          </>
        )}
        {previewMode === 'url' ? (
          <input
            type="url"
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
            placeholder="http://localhost:3000"
            className="min-w-[16rem] flex-1 rounded border border-slack-border bg-slack-bg px-3 py-1.5 font-mono text-sm text-slack-text"
          />
        ) : (
          <span className="truncate font-mono text-xs text-slack-textMuted">{normalizedPath}</span>
        )}
        <button
          type="button"
          onClick={() => setReloadKey((k) => k + 1)}
          className="rounded border border-slack-border px-3 py-1.5 text-xs text-slack-text hover:bg-slack-bgHover"
        >
          Reload preview
        </button>
      </div>
      <div className="grid min-h-0 flex-1 gap-2 lg:grid-cols-2">
        <div className="min-h-0 rounded border border-slack-border overflow-hidden">
          {editorReady ? (
            <Editor
              height="100%"
              defaultLanguage="html"
              value={htmlContent}
              onChange={handleEditorChange}
              theme="vs-dark"
              options={{ minimap: { enabled: false }, wordWrap: 'on', fontSize: 13 }}
            />
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-slack-textMuted">Loading…</div>
          )}
        </div>
        <div className="flex min-h-0 flex-col gap-2">
          <div className="min-h-0 flex-1 rounded border border-slack-border overflow-auto bg-slate-100 flex items-start justify-center p-2">
            <iframe
              ref={iframeRef}
              key={`${iframeSrc}-${reloadKey}-${viewportPreset}`}
              title="HTML preview"
              src={iframeSrc}
              sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
              className="border-0 bg-white shadow"
              style={previewFrameStyle}
              onClick={handleIframeClick}
            />
          </div>
          {hasBrowserSidecar && qaPanel === 'a11y' && (
            <div className="max-h-48 overflow-hidden rounded border border-slack-border bg-slack-bg">
              <BrowserA11yPanel previewUrl={previewUrlForSidecar} />
            </div>
          )}
          {hasBrowserSidecar && qaPanel === 'perf' && (
            <div className="max-h-40 overflow-hidden rounded border border-slack-border bg-slack-bg">
              <BrowserPerfPanel previewUrl={previewUrlForSidecar} />
            </div>
          )}
          {hasBrowserSidecar && qaPanel === 'visual' && (
            <div className="max-h-52 overflow-hidden rounded border border-slack-border bg-slack-bg">
              <BrowserVisualDiffPanel
                previewUrl={previewUrlForSidecar}
                workspaceId={workspaceId}
                htmlPath={normalizedPath}
                viewport={viewport}
              />
            </div>
          )}
          {pickedElement && (
            <div className="rounded border border-slack-border bg-slack-bg p-2 text-xs">
              <div className="mb-1 flex items-center justify-between gap-2">
                <span className="font-medium text-slack-text">Picked element</span>
                <button type="button" className="text-teal-400 hover:underline" onClick={() => void copyPickedToClipboard()}>
                  Copy for chat
                </button>
              </div>
              <pre className="max-h-24 overflow-auto whitespace-pre-wrap text-slack-textMuted">{pickedElement}</pre>
            </div>
          )}
        </div>
      </div>
      <p className="text-[11px] text-slack-textMuted">
        Use <strong>Dev server URL</strong> for Vite/Next local servers. Save the HTML file and click Reload to refresh
        workspace file preview.
        {hasBrowserSidecar && ' QA panels require the Playwright sidecar (run setup-playwright.sh once).'}
      </p>
    </div>
  );
}
