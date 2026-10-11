import { downsampleMono, encodeWavMono } from './wavEncode';

export type AudioInputDevice = {
  deviceId: string;
  label: string;
};

export async function listAudioInputDevices(): Promise<AudioInputDevice[]> {
  if (!navigator.mediaDevices?.enumerateDevices) {
    return [];
  }
  // Prompt once so labels are populated.
  try {
    const probe = await navigator.mediaDevices.getUserMedia({ audio: true });
    probe.getTracks().forEach((t) => t.stop());
  } catch {
    /* permission may already be denied; still try enumerate */
  }
  const devices = await navigator.mediaDevices.enumerateDevices();
  return devices
    .filter((d) => d.kind === 'audioinput')
    .map((d, i) => ({
      deviceId: d.deviceId,
      label: d.label || `Microphone ${i + 1}`,
    }));
}

export type CaptureCallbacks = {
  onChunk: (wav: ArrayBuffer) => void | Promise<void>;
  onError: (err: Error) => void;
};

const TARGET_SAMPLE_RATE = 16000;
const CHUNK_MS = 4000;

/**
 * Starts mic (or virtual input) capture and emits ~4s WAV chunks at 16 kHz mono.
 * Returns a stop function that ends tracks and audio graph.
 */
export async function startAudioCapture(
  deviceId: string | undefined,
  callbacks: CaptureCallbacks
): Promise<() => void> {
  if (!navigator.mediaDevices?.getUserMedia) {
    throw new Error('Microphone capture is not available in this environment');
  }

  const constraints: MediaStreamConstraints = {
    audio: deviceId
      ? { deviceId: { exact: deviceId }, echoCancellation: true, noiseSuppression: true }
      : { echoCancellation: true, noiseSuppression: true },
  };

  let stream: MediaStream;
  try {
    stream = await navigator.mediaDevices.getUserMedia(constraints);
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    if (deviceId) {
      throw new Error(
        `Could not open selected audio input (device may be disconnected): ${msg}`
      );
    }
    throw new Error(`Microphone permission denied or unavailable: ${msg}`);
  }

  const AudioCtx =
    window.AudioContext ||
    (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
  const ctx = new AudioCtx();
  const source = ctx.createMediaStreamSource(stream);
  const processor = ctx.createScriptProcessor(4096, 1, 1);
  const silent = ctx.createGain();
  silent.gain.value = 0;

  let pending: Float32Array[] = [];
  let pendingSamples = 0;
  const samplesPerChunk = Math.floor((ctx.sampleRate * CHUNK_MS) / 1000);
  let stopped = false;

  const flush = async () => {
    if (pendingSamples === 0) return;
    const merged = new Float32Array(pendingSamples);
    let offset = 0;
    for (const part of pending) {
      merged.set(part, offset);
      offset += part.length;
    }
    pending = [];
    pendingSamples = 0;
    const down = downsampleMono(merged, ctx.sampleRate, TARGET_SAMPLE_RATE);
    const wav = encodeWavMono(down, TARGET_SAMPLE_RATE);
    try {
      await callbacks.onChunk(wav);
    } catch (e) {
      callbacks.onError(e instanceof Error ? e : new Error(String(e)));
    }
  };

  processor.onaudioprocess = (ev) => {
    if (stopped) return;
    const input = ev.inputBuffer.getChannelData(0);
    const copy = new Float32Array(input.length);
    copy.set(input);
    pending.push(copy);
    pendingSamples += copy.length;
    if (pendingSamples >= samplesPerChunk) {
      void flush();
    }
  };

  source.connect(processor);
  processor.connect(silent);
  silent.connect(ctx.destination);

  return () => {
    stopped = true;
    void flush();
    try {
      processor.disconnect();
      source.disconnect();
      silent.disconnect();
    } catch {
      /* already disconnected */
    }
    stream.getTracks().forEach((t) => t.stop());
    void ctx.close();
  };
}
