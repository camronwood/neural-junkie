import { useState, useEffect } from 'react';
import { listen } from '@tauri-apps/api/event';
import { getHubBaseURL } from '../config/hubUrl';

interface LoadingScreenProps {
  onReady: () => void;
  onError?: (error: string) => void;
}

export function LoadingScreen({ onReady, onError }: LoadingScreenProps) {
  const [status, setStatus] = useState('Connecting to hub…');
  const [hasError, setHasError] = useState(false);
  const [retryToken, setRetryToken] = useState(0);

  useEffect(() => {
    const unlisten1 = listen<boolean>('server-ready', () => {
      setStatus('Ready');
      setHasError(false);
      setTimeout(onReady, 280);
    });

    const unlisten2 = listen<string>('server-error', event => {
      setStatus(event.payload);
      setHasError(true);
      onError?.(event.payload);
    });

    const pollInterval = setInterval(async () => {
      try {
        const resp = await fetch(`${getHubBaseURL()}/api/health`);
        if (!resp.ok) return;
        const data = (await resp.json()) as { status?: string };
        if (data.status !== 'ok') return;
        clearInterval(pollInterval);
        setHasError(false);
        setStatus('Ready');
        setTimeout(onReady, 280);
      } catch {
        /* hub still starting */
      }
    }, 1000);

    return () => {
      unlisten1.then(fn => fn());
      unlisten2.then(fn => fn());
      clearInterval(pollInterval);
    };
  }, [onReady, onError, retryToken]);

  const handleRetry = () => {
    setHasError(false);
    setStatus('Retrying connection…');
    setRetryToken(t => t + 1);
  };

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-slack-bg px-6">
      <div
        className="w-full max-w-md rounded-2xl border border-slack-border bg-slack-bgHover/80 px-8 py-10 text-center shadow-2xl backdrop-blur-sm"
        role="status"
        aria-live="polite"
      >
        <div className="mx-auto mb-6 flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-slack-accent to-purple-700 shadow-lg shadow-slack-accent/20">
          <span className="text-2xl font-black tracking-tight text-white" aria-hidden>
            NJ
          </span>
        </div>
        <h1 className="text-xl font-semibold tracking-tight text-slack-text">Neural Junkie</h1>
        <p className="mt-3 text-sm leading-relaxed text-slack-textMuted">{status}</p>

        {!hasError && (
          <div className="mt-8 flex justify-center" aria-hidden>
            <div className="h-9 w-9 animate-spin rounded-full border-2 border-slack-accent border-t-transparent" />
          </div>
        )}

        {hasError && (
          <div className="mt-6 space-y-4">
            <div
              className="rounded-lg border border-red-500/40 bg-red-950/30 px-4 py-3 text-left text-xs leading-relaxed text-red-100/95"
              role="alert"
            >
              The hub did not become healthy in time. If you use dev mode, run{' '}
              <code className="rounded bg-slack-bg px-1.5 py-0.5 font-mono text-[11px] text-slack-text">
                make server
              </code>{' '}
              or align{' '}
              <code className="rounded bg-slack-bg px-1.5 py-0.5 font-mono text-[11px] text-slack-text">
                NEURAL_JUNKIE_HUB_URL
              </code>{' '}
              with your hub.
            </div>
            <button
              type="button"
              onClick={handleRetry}
              className="w-full rounded-md bg-slack-accent py-2.5 text-sm font-medium text-white shadow hover:bg-slack-accentHover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
            >
              Check again
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
