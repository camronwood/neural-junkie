import { useCallback, useEffect, useRef, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import type { ChatToolbarActionsLayout } from './ChatToolbarActions';
import {
  listAudioInputDevices,
  startAudioCapture,
  type AudioInputDevice,
} from '../listen/audioCapture';
import { loadListenDeviceId, saveListenDeviceId } from '../listen/devicePreference';

const iconBtn =
  'w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2';

type Props = {
  layout: ChatToolbarActionsLayout;
  serverAddr: string;
};

export function DesktopListenChip({ layout, serverAddr }: Props) {
  const [devices, setDevices] = useState<AudioInputDevice[]>([]);
  const [deviceId, setDeviceId] = useState(() => loadListenDeviceId());
  const [listening, setListening] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [partial, setPartial] = useState('');
  const [sttReady, setSttReady] = useState<boolean | null>(null);
  const sessionIdRef = useRef<string | null>(null);
  const stopCaptureRef = useRef<(() => Promise<void>) | null>(null);
  const uploadChainRef = useRef<Promise<void>>(Promise.resolve());
  const apiRef = useRef<ChatAPI | null>(null);

  useEffect(() => {
    apiRef.current = new ChatAPI(serverAddr);
  }, [serverAddr]);

  const refreshDevices = useCallback(async () => {
    try {
      const list = await listAudioInputDevices();
      setDevices(list);
      if (deviceId && list.length > 0 && !list.some((d) => d.deviceId === deviceId)) {
        setError('Selected audio input is no longer available. Choose another device.');
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to list audio devices');
    }
  }, [deviceId]);

  useEffect(() => {
    void refreshDevices();
    void apiRef.current
      ?.getListenStatus()
      .then((s) => setSttReady(s.stt_ready))
      .catch(() => setSttReady(null));
  }, [refreshDevices]);

  const onDeviceChange = (id: string) => {
    setDeviceId(id);
    saveListenDeviceId(id);
    setError(null);
  };

  const stopAll = useCallback(async (mode: 'stop' | 'cancel') => {
    const stopCap = stopCaptureRef.current;
    stopCaptureRef.current = null;
    if (stopCap) {
      await stopCap();
    }
    // Drain any in-flight chunk uploads before hub stop/cancel.
    await uploadChainRef.current.catch(() => {});
    const sid = sessionIdRef.current;
    sessionIdRef.current = null;
    setListening(false);
    if (!sid || !apiRef.current) return;
    if (mode === 'cancel') {
      await apiRef.current.cancelListen(sid);
      return;
    }
    await apiRef.current.stopListen(sid);
  }, []);

  const handleStart = async () => {
    setBusy(true);
    setError(null);
    setPartial('');
    try {
      const list = await listAudioInputDevices();
      setDevices(list);
      if (deviceId && list.length > 0 && !list.some((d) => d.deviceId === deviceId)) {
        throw new Error('Selected audio input is no longer available. Choose another device.');
      }
      uploadChainRef.current = Promise.resolve();
      // Capture first — do not open hub session if mic fails.
      const stopCap = await startAudioCapture(deviceId || undefined, {
        onChunk: async (wav) => {
          const run = async () => {
            const sid = sessionIdRef.current;
            if (!sid || !apiRef.current) return;
            try {
              const res = await apiRef.current.sendListenChunk(sid, wav);
              setPartial(res.partial_text || '');
            } catch (e) {
              setError(e instanceof Error ? e.message : 'Chunk upload failed');
              throw e;
            }
          };
          const next = uploadChainRef.current.then(run, run);
          uploadChainRef.current = next.then(
            () => undefined,
            () => undefined
          );
          await next;
        },
        onError: (err) => setError(err.message),
      });
      stopCaptureRef.current = stopCap;

      try {
        const started = await apiRef.current!.startListen();
        sessionIdRef.current = started.session_id;
        setListening(true);
        setSttReady(true);
      } catch (hubErr) {
        await stopCap();
        stopCaptureRef.current = null;
        throw hubErr;
      }
    } catch (e) {
      const stopCap = stopCaptureRef.current;
      stopCaptureRef.current = null;
      if (stopCap) {
        await stopCap().catch(() => {});
      }
      if (sessionIdRef.current && apiRef.current) {
        try {
          await apiRef.current.cancelListen(sessionIdRef.current);
        } catch {
          /* ignore */
        }
        sessionIdRef.current = null;
      }
      setListening(false);
      setError(e instanceof Error ? e.message : 'Failed to start listening');
    } finally {
      setBusy(false);
    }
  };

  const handleStop = async () => {
    setBusy(true);
    setError(null);
    try {
      await stopAll('stop');
      setPartial('');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to stop listening');
    } finally {
      setBusy(false);
    }
  };

  const handleCancel = async () => {
    setBusy(true);
    try {
      await stopAll('cancel');
      setPartial('');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to cancel listening');
    } finally {
      setBusy(false);
    }
  };

  const isVertical = layout === 'vertical';

  return (
    <div
      className={
        isVertical
          ? 'flex flex-col items-center gap-1 w-full'
          : 'flex items-center gap-1 shrink-0'
      }
      aria-label="Desktop listen"
    >
      <select
        className="max-w-[9rem] h-7 text-[10px] rounded border border-slack-border bg-slack-bgElevated text-slack-text px-1"
        value={deviceId}
        onChange={(e) => onDeviceChange(e.target.value)}
        disabled={listening || busy}
        title="Audio input (mic or virtual device for Zoom)"
        aria-label="Listen audio input device"
      >
        <option value="">Default microphone</option>
        {devices.map((d) => (
          <option key={d.deviceId} value={d.deviceId}>
            {d.label}
          </option>
        ))}
      </select>
      {!listening ? (
        <button
          type="button"
          disabled={busy || sttReady === false}
          onClick={() => void handleStart()}
          className={`${iconBtn} ${
            sttReady === false
              ? 'bg-slack-bgHover text-slack-textMuted cursor-not-allowed'
              : 'bg-rose-600 hover:bg-rose-700 text-white'
          }`}
          title={
            sttReady === false
              ? 'Local STT unavailable — install whisper.cpp / set NJ_WHISPER_CMD'
              : 'Start Desktop Listen (local STT meeting notes)'
          }
          aria-label="Start desktop listen"
        >
          <MicIcon recording={false} />
        </button>
      ) : (
        <>
          <button
            type="button"
            disabled={busy}
            onClick={() => void handleStop()}
            className={`${iconBtn} bg-rose-700 text-white ring-2 ring-rose-300 animate-pulse`}
            title="Stop listening and save meeting note"
            aria-label="Stop desktop listen"
          >
            <MicIcon recording />
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={() => void handleCancel()}
            className={`${iconBtn} bg-slack-bgHover hover:bg-slack-border text-slack-textMuted text-[10px]`}
            title="Cancel listen without saving"
            aria-label="Cancel desktop listen"
          >
            ✕
          </button>
        </>
      )}
      {error && (
        <span className="text-[10px] text-rose-400 max-w-[12rem] truncate" title={error}>
          {error}
        </span>
      )}
      {listening && partial && (
        <span className="text-[10px] text-slack-textMuted max-w-[10rem] truncate" title={partial}>
          …
        </span>
      )}
    </div>
  );
}

function MicIcon({ recording }: { recording: boolean }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      className="w-3.5 h-3.5"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden
    >
      {recording ? (
        <rect x="6" y="6" width="8" height="8" rx="1" />
      ) : (
        <path d="M10 2a2 2 0 00-2 2v6a2 2 0 104 0V4a2 2 0 00-2-2zM5 9a1 1 0 012 0 3 3 0 006 0 1 1 0 112 0 5 5 0 01-4 4.9V16h2a1 1 0 110 2H7a1 1 0 110-2h2v-2.1A5 5 0 015 9z" />
      )}
    </svg>
  );
}
