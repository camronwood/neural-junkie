import { useState, useEffect, useRef } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { extractTitle, getContentHash } from '../utils/markdownRenderer';
import { RichMarkdownView } from './RichMarkdownView';

interface MarkdownPreviewProps {
  workspaceId: string;
  filePath: string;
}

export function MarkdownPreview({ workspaceId, filePath }: MarkdownPreviewProps) {
  const [content, setContent] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [title, setTitle] = useState<string>('Markdown Preview');
  const [isRendering] = useState<boolean>(false);

  const contentHashRef = useRef<string>('');
  const apiRef = useRef<ChatAPI>(new ChatAPI(getHubBaseURL()));
  const intervalRef = useRef<number | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  const filename = filePath.split('/').pop() || 'Unknown';

  const fetchContent = async () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    abortControllerRef.current = new AbortController();

    try {
      setError(null);
      const fileContent = await apiRef.current.fetchFileContent(workspaceId, filePath);
      const newHash = getContentHash(fileContent);

      if (newHash !== contentHashRef.current) {
        setContent(fileContent);
        setTitle(extractTitle(fileContent));
        setLastUpdated(new Date());
        contentHashRef.current = newHash;
      }
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        return;
      }
      const errorMessage = err instanceof Error ? err.message : 'Failed to load file';
      setError(errorMessage);
      console.error('Failed to fetch file content:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchContent();
  }, [workspaceId, filePath]);

  useEffect(() => {
    intervalRef.current = window.setInterval(() => {
      if (!loading && !isRendering) {
        fetchContent();
      }
    }, 8000);

    return () => {
      if (intervalRef.current) {
        window.clearInterval(intervalRef.current);
      }
    };
  }, [loading, isRendering, workspaceId, filePath]);

  useEffect(() => {
    document.title = `${title} - Markdown Preview`;
  }, [title]);

  const handleRefresh = () => {
    setLoading(true);
    fetchContent();
  };

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  if (loading && !content) {
    return (
      <div className="w-full h-screen bg-slack-bg flex items-center justify-center">
        <div className="flex items-center gap-3 text-slack-text">
          <div className="w-6 h-6 border-2 border-slack-accent border-t-transparent rounded-full animate-spin"></div>
          <span>Loading markdown preview...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="w-full h-screen bg-slack-bg flex items-center justify-center">
        <div className="text-center">
          <div className="text-6xl mb-4">⚠️</div>
          <h2 className="text-xl font-bold text-slack-text mb-2">Failed to load file</h2>
          <p className="text-slack-textMuted mb-4">{error}</p>
          <button
            onClick={handleRefresh}
            className="px-4 py-2 bg-slack-accent hover:bg-slack-accentHover text-white rounded transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full h-screen bg-slack-bg flex flex-col">
      {/* Header */}
      <div className="bg-slack-bgHover border-b border-slack-border px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="text-2xl">📝</div>
          <div>
            <h1 className="font-bold text-slack-text">{filename}</h1>
            <p className="text-sm text-slack-textMuted">{filePath}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {lastUpdated && (
            <span className="text-sm text-slack-textMuted">
              Updated at {formatTime(lastUpdated)}
            </span>
          )}
          <button
            onClick={handleRefresh}
            className="px-3 py-1 bg-slack-bg hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
            title="Refresh content"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        <div className="max-w-6xl mx-auto p-6">
          <RichMarkdownView content={content} />
        </div>
      </div>

      {/* Loading indicator for auto-refresh */}
      {loading && content && (
        <div className="absolute top-4 right-4 bg-slack-bgHover border border-slack-border rounded px-3 py-2 flex items-center gap-2">
          <div className="w-3 h-3 border border-slack-accent border-t-transparent rounded-full animate-spin"></div>
          <span className="text-sm text-slack-text">Updating...</span>
        </div>
      )}
    </div>
  );
}
