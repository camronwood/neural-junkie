import { useState, useEffect, useRef } from 'react';
import { listen } from '@tauri-apps/api/event';
import { getHubBaseURL } from '../config/hubUrl';

interface LoadingScreenProps {
  onReady: () => void;
  onError?: (error: string) => void;
  /** When hub never responds, let user open sign-in to enter a different server URL. */
  onContinueWithoutHub?: () => void;
}

const HUB_WAIT_MS = 50_000;

export function LoadingScreen({ onReady, onError, onContinueWithoutHub }: LoadingScreenProps) {
  const [status, setStatus] = useState('Connecting to hub…');
  const [hasError, setHasError] = useState(false);
  const [retryToken, setRetryToken] = useState(0);
  const hubUrl = getHubBaseURL();
  const finishedRef = useRef(false);

  const markConnected = () => {
    finishedRef.current = true;
  };

  useEffect(() => {
    finishedRef.current = false;

    const unlisten1 = listen<boolean>('server-ready', () => {
      setStatus('Ready');
      setHasError(false);
      markConnected();
      setTimeout(onReady, 280);
    });

    const unlisten2 = listen<string>('server-error', event => {
      setStatus(event.payload);
      setHasError(true);
      onError?.(event.payload);
    });

    const pollInterval = setInterval(async () => {
      try {
        const resp = await fetch(`${hubUrl}/api/health`);
        if (!resp.ok) return;
        const data = (await resp.json()) as { status?: string };
        if (data.status !== 'ok') return;
        clearInterval(pollInterval);
        setHasError(false);
        setStatus('Ready');
        markConnected();
        setTimeout(onReady, 280);
      } catch {
        /* hub still starting or wrong host/port */
      }
    }, 1000);

    const timeoutId = window.setTimeout(() => {
      if (!finishedRef.current) {
        setHasError(true);
        setStatus('Still waiting for the hub');
        onError?.(
          `No answer from ${hubUrl} after ${HUB_WAIT_MS / 1000}s. Start the hub (e.g. make server) or rebuild with VITE_NJ_HUB_URL if the port differs.`
        );
      }
    }, HUB_WAIT_MS);

    return () => {
      clearTimeout(timeoutId);
      unlisten1.then(fn => fn());
      unlisten2.then(fn => fn());
      clearInterval(pollInterval);
    };
  }, [onReady, onError, retryToken, hubUrl]);

  const handleRetry = () => {
    finishedRef.current = false;
    setHasError(false);
    setStatus('Retrying connection…');
    setRetryToken(t => t + 1);
  };

  const showActions = hasError;

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
        <p className="mt-2 font-mono text-[11px] text-slack-textMuted/90 break-all">
          {hubUrl}
        </p>

        {!showActions && (
          <div className="mt-8 flex justify-center" aria-hidden>
            <div className="h-9 w-9 animate-spin rounded-full border-2 border-slack-accent border-t-transparent" />
          </div>
        )}

        {showActions && (
          <div className="mt-6 space-y-3">
            <div
              className="rounded-lg border border-red-500/40 bg-red-950/30 px-4 py-3 text-left text-xs leading-relaxed text-red-100/95"
              role="alert"
            >
              <p className="font-medium text-red-200">Hub not reachable</p>
              <p className="mt-2">
                Start the Go hub so <span className="font-mono text-red-100/90">{hubUrl}/api/health</span> returns{' '}
                <code className="rounded bg-slack-bg px-1 py-0.5 font-mono text-[11px]">{`{"status":"ok"}`}</code>
                . From the repo:{' '}
                <code className="rounded bg-slack-bg px-1.5 py-0.5 font-mono text-[11px] text-slack-text">
                  make server
                </code>{' '}
                or{' '}
                <code className="rounded bg-slack-bg px-1.5 py-0.5 font-mono text-[11px] text-slack-text">
                  make start-all
                </code>
                . If the hub uses another port, set <strong>VITE_NJ_HUB_URL</strong> when building the desktop app, or use{' '}
                <strong>Continue to sign-in</strong> and enter the URL there.
              </p>
            </div>
            <button
              type="button"
              onClick={handleRetry}
              className="w-full rounded-md bg-slack-accent py-2.5 text-sm font-medium text-white shadow hover:bg-slack-accentHover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
            >
              Check again
            </button>
            {onContinueWithoutHub && (
              <button
                type="button"
                onClick={onContinueWithoutHub}
                className="w-full rounded-md border border-slack-border bg-slack-bg py-2.5 text-sm font-medium text-slack-text hover:bg-slack-bgHover focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slack-accent"
              >
                Continue to sign-in
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
