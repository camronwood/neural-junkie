import { useEffect, useState } from 'react';
import { invoke, isTauri } from '@tauri-apps/api/core';
import { listen } from '@tauri-apps/api/event';
import { getHubBaseURL } from '../config/hubUrl';

/**
 * True once the packaged sidecar has emitted server-ready (or health confirms ready).
 * Dev / browser builds assume the hub is managed externally (make server).
 */
export function useHubSidecarReady(): boolean {
  const [ready, setReady] = useState(() => !import.meta.env.PROD || !isTauri());

  useEffect(() => {
    if (!import.meta.env.PROD || !isTauri()) {
      return;
    }

    let cancelled = false;

    const markReady = () => {
      if (!cancelled) {
        setReady(true);
      }
    };

    void invoke<boolean>('get_server_status')
      .then((ok) => {
        if (ok) markReady();
      })
      .catch(() => {
        /* fall through to health poll */
      });

    const unlisten = listen<boolean>('server-ready', () => {
      markReady();
    });

    // LoadingScreen may finish before this hook mounts — poll health as a fallback.
    const pollHealth = async () => {
      try {
        const resp = await fetch(`${getHubBaseURL()}/api/health`);
        if (!resp.ok) return;
        const data = (await resp.json()) as { status?: string };
        if (data.status === 'ok') {
          markReady();
        }
      } catch {
        /* hub still starting */
      }
    };

    void pollHealth();
    const pollId = window.setInterval(() => void pollHealth(), 1500);

    return () => {
      cancelled = true;
      window.clearInterval(pollId);
      void unlisten.then((fn) => fn());
    };
  }, []);

  return ready;
}
