import { useCallback, useEffect, useMemo, useState } from 'react';
import Editor from '@monaco-editor/react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useToastStore } from '../stores/toastStore';
import { useEditorStore } from '../stores/editorStore';

const api = new ChatAPI(getHubBaseURL());

interface HtmlBrowserWorkbenchProps {
  workspaceId: string;
  htmlPath: string;
  initialContent?: string;
  initialUrl?: string;
  tabId: string;
}

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
  const { addToast } = useToastStore();

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

  const editorReady = loadedPath === normalizedPath;

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
        <div className="min-h-0 rounded border border-slack-border overflow-hidden bg-white">
          <iframe
            key={`${iframeSrc}-${reloadKey}`}
            title="HTML preview"
            src={iframeSrc}
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
            className="h-full w-full border-0 bg-white"
          />
        </div>
      </div>
      <p className="text-[11px] text-slack-textMuted">
        Use <strong>Dev server URL</strong> for Vite/Next local servers. Save the HTML file and click Reload to refresh
        workspace file preview.
      </p>
    </div>
  );
}
