import { getHubBaseURL } from '../../config/hubUrl';

export type WaveformPeaks = {
  duration_sec: number;
  sample_rate: number;
  peaks: number[];
};

export type MusicGenerationRecord = {
  id: string;
  path: string;
  style_tags: string;
  lyrics: string;
  seed?: number;
  variant?: string;
  created_at: string;
};

async function musicPost<T>(path: string, body: Record<string, unknown>): Promise<T> {
  const res = await fetch(`${getHubBaseURL()}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const data = (await res.json()) as T & { error?: string };
  if (!res.ok) {
    throw new Error(data.error || res.statusText);
  }
  if (typeof data === 'object' && data !== null && 'error' in data && data.error) {
    throw new Error(data.error);
  }
  return data;
}

async function musicGet<T>(path: string): Promise<T> {
  const res = await fetch(`${getHubBaseURL()}${path}`);
  const data = (await res.json()) as T & { error?: string };
  if (!res.ok) {
    throw new Error(data.error || res.statusText);
  }
  return data;
}

export async function musicGenerate(body: {
  style_tags: string;
  lyrics?: string;
  duration_sec?: number;
  instrumental?: boolean;
  seed?: number;
}) {
  return musicPost<{ mime: string; data: string; path: string; generation_id?: string }>(
    '/api/music/generate',
    body,
  );
}

export async function musicWaveform(audioPath: string, buckets = 512) {
  return musicPost<WaveformPeaks>('/api/music/waveform', { audio_path: audioPath, buckets });
}

export async function musicExtract(audioPath: string, tracks: string[]) {
  return musicPost<{ stems?: Array<{ track: string; path: string; data: string; mime: string }> }>(
    '/api/music/extract',
    { audio_path: audioPath, tracks },
  );
}

export async function musicRepaint(body: {
  audio_path: string;
  start_sec: number;
  end_sec: number;
  style_tags?: string;
  lyrics?: string;
}) {
  return musicPost<{ path: string; data: string; mime: string }>('/api/music/repaint', body);
}

export async function musicRenderArrangement(sections: Array<{ id: string; path: string }>, crossfadeSec = 0.05) {
  return musicPost<{ path: string; data: string; mime: string }>('/api/music/render-arrangement', {
    sections,
    crossfade_sec: crossfadeSec,
  });
}

export async function musicStatus() {
  return musicGet<{
    ready: boolean;
    demo_mode: boolean;
    variant: string;
    install_progress?: { phase: string; detail: string };
  }>('/api/music/status');
}

export async function musicInstallProgress() {
  return musicGet<{ phase: string; detail: string }>('/api/music/install/progress');
}

export function b64ToBlobUrl(mime: string, b64: string): string {
  const binary = atob(b64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i);
  const blob = new Blob([bytes], { type: mime });
  return URL.createObjectURL(blob);
}
