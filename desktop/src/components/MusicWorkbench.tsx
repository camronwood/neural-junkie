import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useEditorStore } from '../stores/editorStore';
import { useToastStore } from '../stores/toastStore';
import { MusicCreationToolsPanel } from './MusicCreationToolsPanel';
import {
  musicExtract,
  musicGenerate,
  musicRenderArrangement,
  musicRepaint,
  musicWaveform,
  type MusicGenerationRecord,
  type WaveformPeaks,
} from './music/musicSidecarApi';

const api = new ChatAPI(getHubBaseURL());

export type MusicProject = {
  version: number;
  title: string;
  bpm: number;
  key?: string;
  style_tags: string;
  sections: Array<{
    id: string;
    label: string;
    start_sec: number;
    duration_sec: number;
    lyrics: string;
    style_tags: string;
    path?: string;
  }>;
  generations: MusicGenerationRecord[];
  accepted_mix: string | null;
};

interface MusicWorkbenchProps {
  workspaceId: string;
  audioPath?: string;
  projectPath?: string;
  tabId: string;
}

function isProjectPath(path: string) {
  return path.endsWith('.nj-music.json') || path.endsWith('project.nj-music.json');
}

export function MusicWorkbench({ workspaceId, audioPath, projectPath, tabId }: MusicWorkbenchProps) {
  const { addToast } = useToastStore();
  const activePath = projectPath || audioPath || '';
  const normalizedPath = activePath.replace(/\\/g, '/').replace(/^\/+/, '');

  const [styleTags, setStyleTags] = useState('lo-fi, 90 bpm');
  const [lyrics, setLyrics] = useState('[Instrumental]');
  const [project, setProject] = useState<MusicProject | null>(null);
  const [audioFilePath, setAudioFilePath] = useState<string | null>(audioPath ?? null);
  const [peaks, setPeaks] = useState<WaveformPeaks | null>(null);
  const [loopA, setLoopA] = useState(0);
  const [loopB, setLoopB] = useState<number | null>(null);
  const [playing, setPlaying] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [drawerTab, setDrawerTab] = useState<'generations' | 'stems' | 'settings'>('generations');
  const [compareA, setCompareA] = useState<MusicGenerationRecord | null>(null);
  const [compareB, setCompareB] = useState<MusicGenerationRecord | null>(null);
  const [selectedSectionId, setSelectedSectionId] = useState<string | null>(null);
  const [generations, setGenerations] = useState<MusicGenerationRecord[]>([]);

  const audioRef = useRef<HTMLAudioElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  const loadProject = useCallback(async () => {
    if (!isProjectPath(normalizedPath)) return;
    try {
      const raw = await api.fetchFileContent(workspaceId, normalizedPath);
      const parsed = JSON.parse(raw) as MusicProject;
      setProject(parsed);
      setStyleTags(parsed.style_tags || styleTags);
      if (parsed.sections?.[0]) {
        setSelectedSectionId(parsed.sections[0].id);
        setLyrics(parsed.sections[0].lyrics || lyrics);
        setStyleTags(parsed.sections[0].style_tags || parsed.style_tags || styleTags);
      }
      setGenerations(parsed.generations || []);
    } catch (err) {
      addToast({ type: 'error', title: 'Music', message: err instanceof Error ? err.message : String(err) });
    }
  }, [workspaceId, normalizedPath, addToast]);

  useEffect(() => {
    void loadProject();
  }, [loadProject]);

  const refreshWaveform = useCallback(async (path: string) => {
    try {
      const wf = await musicWaveform(path);
      setPeaks(wf);
      setLoopB(wf.duration_sec);
    } catch {
      setPeaks(null);
    }
  }, []);

  useEffect(() => {
    if (audioFilePath) void refreshWaveform(audioFilePath);
  }, [audioFilePath, refreshWaveform]);

  const drawWaveform = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas || !peaks) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const { width, height } = canvas;
    ctx.clearRect(0, 0, width, height);
    ctx.fillStyle = '#1a1d21';
    ctx.fillRect(0, 0, width, height);
    const barW = width / peaks.peaks.length;
    peaks.peaks.forEach((p, i) => {
      const h = p * height * 0.9;
      ctx.fillStyle = '#4a9eff';
      ctx.fillRect(i * barW, (height - h) / 2, Math.max(1, barW - 1), h);
    });
    if (loopB != null && peaks.duration_sec > 0) {
      const ax = (loopA / peaks.duration_sec) * width;
      const bx = (loopB / peaks.duration_sec) * width;
      ctx.fillStyle = 'rgba(255, 200, 80, 0.15)';
      ctx.fillRect(ax, 0, bx - ax, height);
      ctx.strokeStyle = '#ffc850';
      ctx.beginPath();
      ctx.moveTo(ax, 0);
      ctx.lineTo(ax, height);
      ctx.moveTo(bx, 0);
      ctx.lineTo(bx, height);
      ctx.stroke();
    }
  }, [peaks, loopA, loopB]);

  useEffect(() => {
    drawWaveform();
  }, [drawWaveform]);

  const persistProject = useCallback(
    async (next: MusicProject) => {
      if (!isProjectPath(normalizedPath)) return;
      const content = JSON.stringify(next, null, 2);
      await api.saveFileContent(workspaceId, normalizedPath, content);
      useEditorStore.getState().updateTabContent(tabId, content);
      setProject(next);
    },
    [workspaceId, normalizedPath, tabId],
  );

  const runGenerate = async () => {
    setGenerating(true);
    try {
      const section = project?.sections.find((s) => s.id === selectedSectionId);
      const res = await musicGenerate({
        style_tags: section?.style_tags || styleTags,
        lyrics: section?.lyrics || lyrics,
        duration_sec: section?.duration_sec || 30,
        instrumental: (section?.lyrics || lyrics).includes('[Instrumental]'),
      });
      const record: MusicGenerationRecord = {
        id: res.generation_id || `gen-${Date.now()}`,
        path: res.path,
        style_tags: section?.style_tags || styleTags,
        lyrics: section?.lyrics || lyrics,
        created_at: new Date().toISOString(),
      };
      setGenerations((g) => [record, ...g]);
      setAudioFilePath(res.path);
      if (project && isProjectPath(normalizedPath)) {
        const next = { ...project, generations: [record, ...(project.generations || [])] };
        if (section) {
          next.sections = project.sections.map((s) =>
            s.id === section.id ? { ...s, path: res.path } : s,
          );
        }
        await persistProject(next);
      }
      addToast({ type: 'success', title: 'Music', message: 'Generation complete' });
    } catch (err) {
      addToast({ type: 'error', title: 'Music', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setGenerating(false);
    }
  };

  const runRepaint = async () => {
    if (!audioFilePath || loopB == null) return;
    setGenerating(true);
    try {
      const res = await musicRepaint({
        audio_path: audioFilePath,
        start_sec: loopA,
        end_sec: loopB,
        style_tags: styleTags,
        lyrics,
      });
      setAudioFilePath(res.path);
      addToast({ type: 'success', title: 'Music', message: 'Loop repainted' });
    } catch (err) {
      addToast({ type: 'error', title: 'Music', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setGenerating(false);
    }
  };

  const runExtractStems = async () => {
    if (!audioFilePath) return;
    try {
      await musicExtract(audioFilePath, ['vocals', 'drums', 'bass']);
      addToast({ type: 'success', title: 'Music', message: 'Stems extracted — check chat or output folder' });
    } catch (err) {
      addToast({ type: 'error', title: 'Music', message: err instanceof Error ? err.message : String(err) });
    }
  };

  const runRenderArrangement = async () => {
    if (!project?.sections?.length) return;
    const withAudio = project.sections.filter((s) => s.path);
    if (!withAudio.length) {
      addToast({ type: 'error', title: 'Music', message: 'Generate audio for sections first' });
      return;
    }
    setGenerating(true);
    try {
      const res = await musicRenderArrangement(withAudio.map((s) => ({ id: s.id, path: s.path! })));
      setAudioFilePath(res.path);
      const next = { ...project, accepted_mix: res.path };
      await persistProject(next);
      addToast({ type: 'success', title: 'Music', message: 'Arrangement rendered' });
    } catch (err) {
      addToast({ type: 'error', title: 'Music', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setGenerating(false);
    }
  };

  const audioSrc = useMemo(() => {
    if (!audioFilePath) return '';
    return `${getHubBaseURL()}/api/workspace-preview?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(audioFilePath.replace(/^\/+/, ''))}`;
  }, [audioFilePath, workspaceId]);

  const compareSrcA = compareA?.path
    ? `${getHubBaseURL()}/api/workspace-preview?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(compareA.path.replace(/^\/+/, ''))}`
    : '';
  const compareSrcB = compareB?.path
    ? `${getHubBaseURL()}/api/workspace-preview?workspace=${encodeURIComponent(workspaceId)}&path=${encodeURIComponent(compareB.path.replace(/^\/+/, ''))}`
    : '';

  return (
    <div className="flex flex-col h-full min-h-0 bg-slack-bg">
      <div className="flex items-center gap-2 px-3 py-2 border-b border-slack-border bg-slack-bgHover text-sm flex-wrap">
        <button type="button" className="px-2 py-1 rounded bg-slack-accent text-white" onClick={() => audioRef.current?.play()} disabled={!audioSrc}>
          ▶ Play
        </button>
        <button type="button" className="px-2 py-1 rounded border border-slack-border" onClick={() => audioRef.current?.pause()}>
          ⏸ Pause
        </button>
        <span className="text-slack-textMuted">Loop A: {loopA.toFixed(1)}s</span>
        <span className="text-slack-textMuted">Loop B: {(loopB ?? 0).toFixed(1)}s</span>
        <button type="button" className="px-2 py-1 rounded border border-slack-border" onClick={runRepaint} disabled={generating || !audioFilePath}>
          Repaint loop
        </button>
        <button type="button" className="px-2 py-1 rounded border border-slack-border" onClick={runGenerate} disabled={generating}>
          {generating ? 'Generating…' : 'Generate'}
        </button>
        {project && (
          <button type="button" className="px-2 py-1 rounded border border-slack-border" onClick={runRenderArrangement} disabled={generating}>
            Render arrangement
          </button>
        )}
      </div>

      <div className="flex flex-1 min-h-0">
        <div className="w-[38%] min-w-[240px] border-r border-slack-border flex flex-col">
          <div className="p-2 border-b border-slack-border text-xs font-medium text-slack-textMuted">Style & lyrics</div>
          <textarea
            className="flex-1 min-h-[120px] p-3 bg-slack-bg text-slack-text text-sm resize-none border-0 outline-none"
            value={styleTags}
            onChange={(e) => setStyleTags(e.target.value)}
            placeholder="Style tags (ACE-Step caption)"
          />
          <textarea
            className="flex-1 min-h-[160px] p-3 bg-slack-bg text-slack-text text-sm resize-none border-t border-slack-border outline-none font-mono"
            value={lyrics}
            onChange={(e) => setLyrics(e.target.value)}
            placeholder="Lyrics with [Verse] / [Chorus] markers"
          />
          {project?.sections?.length ? (
            <div className="border-t border-slack-border p-2">
              <div className="text-xs text-slack-textMuted mb-1">Timeline sections</div>
              <div className="flex flex-wrap gap-1">
                {project.sections.map((sec) => (
                  <button
                    key={sec.id}
                    type="button"
                    className={`text-xs px-2 py-1 rounded border ${selectedSectionId === sec.id ? 'border-slack-accent bg-slack-accent/20' : 'border-slack-border'}`}
                    onClick={() => {
                      setSelectedSectionId(sec.id);
                      setLyrics(sec.lyrics);
                      setStyleTags(sec.style_tags);
                      if (sec.path) setAudioFilePath(sec.path);
                    }}
                  >
                    {sec.label} ({sec.duration_sec}s)
                  </button>
                ))}
              </div>
            </div>
          ) : null}
        </div>

        <div className="flex-1 flex flex-col min-w-0">
          <canvas ref={canvasRef} className="w-full h-40 bg-slack-bgHover" width={800} height={160} />
          {audioSrc ? (
            <audio
              ref={audioRef}
              src={audioSrc}
              controls
              className="w-full px-2"
              onPlay={() => setPlaying(true)}
              onPause={() => setPlaying(false)}
            />
          ) : (
            <div className="p-4 text-sm text-slack-textMuted">Open or generate audio to preview</div>
          )}

          {(compareA || compareB) && (
            <div className="border-t border-slack-border p-2 grid grid-cols-2 gap-2">
              <div>
                <div className="text-xs mb-1">Gen A {compareA?.created_at?.slice(0, 19)}</div>
                {compareSrcA ? <audio controls src={compareSrcA} className="w-full" /> : null}
                <button type="button" className="text-xs mt-1 text-slack-accent" onClick={() => compareA && setAudioFilePath(compareA.path)}>
                  Accept A
                </button>
              </div>
              <div>
                <div className="text-xs mb-1">Gen B {compareB?.created_at?.slice(0, 19)}</div>
                {compareSrcB ? <audio controls src={compareSrcB} className="w-full" /> : null}
                <button type="button" className="text-xs mt-1 text-slack-accent" onClick={() => compareB && setAudioFilePath(compareB.path)}>
                  Accept B
                </button>
              </div>
            </div>
          )}
        </div>

        <div className="w-64 border-l border-slack-border flex flex-col text-sm">
          <div className="flex border-b border-slack-border">
            {(['generations', 'stems', 'settings'] as const).map((t) => (
              <button
                key={t}
                type="button"
                className={`flex-1 py-2 text-xs capitalize ${drawerTab === t ? 'border-b-2 border-slack-accent' : 'text-slack-textMuted'}`}
                onClick={() => setDrawerTab(t)}
              >
                {t}
              </button>
            ))}
          </div>
          <div className="flex-1 overflow-auto p-2">
            {drawerTab === 'generations' && (
              <ul className="space-y-2">
                {generations.map((g) => (
                  <li key={g.id} className="border border-slack-border rounded p-2">
                    <div className="text-xs text-slack-textMuted truncate">{g.style_tags}</div>
                    <div className="flex gap-1 mt-1">
                      <button type="button" className="text-xs text-slack-accent" onClick={() => setAudioFilePath(g.path)}>
                        Load
                      </button>
                      <button type="button" className="text-xs text-slack-accent" onClick={() => setCompareA(g)}>
                        A
                      </button>
                      <button type="button" className="text-xs text-slack-accent" onClick={() => setCompareB(g)}>
                        B
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
            {drawerTab === 'stems' && (
              <div className="space-y-2">
                <p className="text-xs text-slack-textMuted">Requires SFT variant for real stems.</p>
                <button type="button" className="w-full py-2 rounded bg-slack-accent text-white text-xs" onClick={runExtractStems} disabled={!audioFilePath}>
                  Extract vocals + drums + bass
                </button>
              </div>
            )}
            {drawerTab === 'settings' && (
              <MusicCreationToolsPanel hubHttp={getHubBaseURL()} isActive packEnabled />
            )}
          </div>
        </div>
      </div>
      {playing ? null : null}
    </div>
  );
}
