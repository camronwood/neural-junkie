import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import * as THREE from 'three';
import { STLLoader } from 'three/examples/jsm/loaders/STLLoader.js';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
import { ChatAPI, type CadParam } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useToastStore } from '../stores/toastStore';
import { useEditorStore } from '../stores/editorStore';

const api = new ChatAPI(getHubBaseURL());

interface CadWorkbenchProps {
  workspaceId: string;
  scadPath: string;
  initialContent?: string;
  projectId?: string;
  tabId: string;
}

export function CadWorkbench({
  workspaceId,
  scadPath,
  initialContent = '',
  projectId,
  tabId,
}: CadWorkbenchProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const sceneRef = useRef<{
    renderer: THREE.WebGLRenderer;
    scene: THREE.Scene;
    camera: THREE.PerspectiveCamera;
    controls: OrbitControls;
    mesh: THREE.Mesh | null;
  } | null>(null);
  const renderTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [scadContent, setScadContent] = useState('');
  const [params, setParams] = useState<CadParam[]>([]);
  const [paramOverrides, setParamOverrides] = useState<Record<string, string>>({});
  const [rendering, setRendering] = useState(false);
  const [versions, setVersions] = useState<Array<{ id: string; label: string; created_at: string }>>([]);
  const [versionLabel, setVersionLabel] = useState('');
  const [drawerOpen, setDrawerOpen] = useState(true);
  const [loadedPath, setLoadedPath] = useState<string | null>(null);
  const [contentRevision, setContentRevision] = useState(0);
  const { addToast } = useToastStore();

  const normalizedScadPath = scadPath.replace(/\\/g, '/').replace(/^\/+/, '');

  useEffect(() => {
    let cancelled = false;
    setLoadedPath(null);
    setParamOverrides({});
    setParams([]);

    void (async () => {
      try {
        const diskContent = await api.fetchFileContent(workspaceId, normalizedScadPath);
        if (cancelled) return;
        setScadContent(diskContent);
        setLoadedPath(normalizedScadPath);
        useEditorStore.getState().updateTabContent(tabId, diskContent);
      } catch (err) {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : 'Failed to load SCAD file';
        addToast({ type: 'error', title: 'CAD', message });
        setScadContent(initialContent);
        setLoadedPath(normalizedScadPath);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [tabId, normalizedScadPath, workspaceId, addToast]);

  const handleEditorChange = useCallback((value: string | undefined) => {
    const next = value ?? '';
    setScadContent(next);
    useEditorStore.getState().updateTabContent(tabId, next);
    useEditorStore.getState().markTabDirty(tabId, true);
  }, [tabId]);

  const editorReady = loadedPath === normalizedScadPath;

  const effectiveParams = useMemo(() => {
    const base: Record<string, string> = {};
    for (const p of params) {
      base[p.name] = paramOverrides[p.name] ?? p.value.replace(/^"|"$/g, '');
    }
    return base;
  }, [params, paramOverrides]);

  const loadMeshFromBase64 = useCallback((b64: string) => {
    const sceneCtx = sceneRef.current;
    if (!sceneCtx || !b64) return;
    const loader = new STLLoader();
    const binary = atob(b64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i);
    const geometry = loader.parse(bytes.buffer);
    geometry.computeBoundingBox();
    geometry.center();
    if (sceneCtx.mesh) {
      sceneCtx.scene.remove(sceneCtx.mesh);
      sceneCtx.mesh.geometry.dispose();
      (sceneCtx.mesh.material as THREE.Material).dispose();
    }
    const material = new THREE.MeshStandardMaterial({ color: 0x5b8def, metalness: 0.1, roughness: 0.65 });
    const mesh = new THREE.Mesh(geometry, material);
    sceneCtx.mesh = mesh;
    sceneCtx.scene.add(mesh);
    const box = geometry.boundingBox!;
    const size = box.getSize(new THREE.Vector3()).length();
    sceneCtx.camera.position.set(size, size, size);
    sceneCtx.controls.target.set(0, 0, 0);
    sceneCtx.controls.update();
  }, []);

  const runRender = useCallback(async () => {
    setRendering(true);
    try {
      await api.saveFileContent(workspaceId, scadPath, scadContent);
      const result = await api.renderCAD({
        workspace: workspaceId,
        path: scadPath,
        project_id: projectId,
        params: effectiveParams,
      });
      if (result.params) {
        setParams(result.params as CadParam[]);
      }
      loadMeshFromBase64(result.content_base64);
      useEditorStore.getState().updateTabContent(tabId, scadContent);
      useEditorStore.getState().markTabDirty(tabId, false);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Render failed';
      addToast({ type: 'error', title: 'CAD render', message });
    } finally {
      setRendering(false);
    }
  }, [
    workspaceId,
    scadPath,
    scadContent,
    projectId,
    effectiveParams,
    loadMeshFromBase64,
    addToast,
    tabId,
  ]);

  const refreshParams = useCallback(async () => {
    try {
      const data = await api.fetchCADParams(workspaceId, scadPath, projectId);
      setParams(data.params ?? []);
    } catch {
      /* optional */
    }
  }, [workspaceId, scadPath, projectId]);

  const refreshVersions = useCallback(async () => {
    try {
      const data = await api.fetchCADVersions(projectId ?? 'default');
      setVersions(data.versions ?? []);
    } catch {
      /* optional */
    }
  }, [projectId]);

  useEffect(() => {
    if (!canvasRef.current) return;
    const renderer = new THREE.WebGLRenderer({ canvas: canvasRef.current, antialias: true });
    renderer.setPixelRatio(window.devicePixelRatio);
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x1a1d21);
    const camera = new THREE.PerspectiveCamera(45, 1, 0.1, 10000);
    camera.position.set(120, 120, 120);
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    const ambient = new THREE.AmbientLight(0xffffff, 0.55);
    const dir = new THREE.DirectionalLight(0xffffff, 0.85);
    dir.position.set(200, 300, 100);
    scene.add(ambient, dir);
    sceneRef.current = { renderer, scene, camera, controls, mesh: null };

    const resize = () => {
      const parent = canvasRef.current?.parentElement;
      if (!parent) return;
      const w = parent.clientWidth;
      const h = parent.clientHeight;
      renderer.setSize(w, h, false);
      camera.aspect = w / Math.max(h, 1);
      camera.updateProjectionMatrix();
    };
    resize();
    const resizeObserver = new ResizeObserver(resize);
    if (canvasRef.current?.parentElement) {
      resizeObserver.observe(canvasRef.current.parentElement);
    }
    window.addEventListener('resize', resize);
    let frame = 0;
    const animate = () => {
      frame = requestAnimationFrame(animate);
      controls.update();
      renderer.render(scene, camera);
    };
    animate();

    void refreshParams();
    void refreshVersions();
    void (async () => {
      try {
        const mesh = await api.fetchCADMesh(workspaceId, scadPath, projectId);
        if (mesh.content_base64) loadMeshFromBase64(mesh.content_base64);
      } catch {
        /* no preview yet */
      }
    })();

    return () => {
      resizeObserver.disconnect();
      window.removeEventListener('resize', resize);
      cancelAnimationFrame(frame);
      controls.dispose();
      renderer.dispose();
      sceneRef.current = null;
    };
  }, [workspaceId, scadPath, projectId, loadMeshFromBase64, refreshParams, refreshVersions]);

  useEffect(() => {
    if (renderTimer.current) clearTimeout(renderTimer.current);
    renderTimer.current = setTimeout(() => {
      if (Object.keys(paramOverrides).length > 0) void runRender();
    }, 450);
    return () => {
      if (renderTimer.current) clearTimeout(renderTimer.current);
    };
  }, [paramOverrides, runRender]);

  const saveVersion = async () => {
    try {
      await api.saveFileContent(workspaceId, scadPath, scadContent);
      await api.saveCADVersion({
        workspace: workspaceId,
        path: scadPath,
        project_id: projectId ?? 'default',
        label: versionLabel || `Version ${new Date().toLocaleString()}`,
        params: effectiveParams,
      });
      useEditorStore.getState().updateTabContent(tabId, scadContent);
      useEditorStore.getState().markTabDirty(tabId, false);
      setVersionLabel('');
      await refreshVersions();
      addToast({ type: 'success', title: 'CAD', message: 'Version saved' });
    } catch (err) {
      addToast({ type: 'error', title: 'CAD', message: err instanceof Error ? err.message : 'Save failed' });
    }
  };

  const restoreVersion = async (versionId: string) => {
    try {
      const data = await api.restoreCADVersion(projectId ?? 'default', versionId);
      if (data.content) {
        setScadContent(data.content);
        setContentRevision((n) => n + 1);
        useEditorStore.getState().updateTabContent(tabId, data.content);
        setParamOverrides({});
        await runRender();
      }
      addToast({ type: 'success', title: 'CAD', message: 'Version restored' });
    } catch (err) {
      addToast({ type: 'error', title: 'CAD', message: err instanceof Error ? err.message : 'Restore failed' });
    }
  };

  return (
    <div className="flex flex-col h-full min-h-0 bg-slack-bg">
      <div className="flex flex-1 min-h-0">
        <div className="flex flex-col w-[32%] min-w-[220px] max-w-[420px] border-r border-slack-border">
          <div className="px-3 py-2 border-b border-slack-border flex items-center justify-between gap-2">
            <span className="text-xs font-medium text-slack-text truncate">{scadPath}</span>
            <button
              type="button"
              onClick={() => void runRender()}
              disabled={rendering}
              className="text-xs px-2 py-1 rounded bg-slack-accent text-white disabled:opacity-50 shrink-0"
            >
              {rendering ? 'Rendering…' : 'Render'}
            </button>
          </div>
          <div className="flex-1 min-h-0">
            {editorReady ? (
              <Editor
                key={`${workspaceId}:${normalizedScadPath}:${contentRevision}`}
                path={`cad://${workspaceId}/${normalizedScadPath}`}
                height="100%"
                language="plaintext"
                theme="vs-dark"
                defaultValue={scadContent}
                onChange={handleEditorChange}
                options={{ minimap: { enabled: false }, fontSize: 13, wordWrap: 'on' }}
              />
            ) : (
              <div className="flex h-full items-center justify-center text-xs text-slack-textMuted">
                Loading {normalizedScadPath}…
              </div>
            )}
          </div>
        </div>
        <div className="flex flex-col flex-1 min-w-0 min-h-0">
          <canvas ref={canvasRef} className="flex-1 w-full min-h-0 block" />
        </div>
      </div>

      <div className="border-t border-slack-border bg-slack-bgHover shrink-0">
        <button
          type="button"
          onClick={() => setDrawerOpen((open) => !open)}
          className="w-full px-3 py-1.5 flex items-center justify-between gap-2 text-xs font-medium text-slack-text hover:bg-slack-bg transition-colors"
          aria-expanded={drawerOpen}
        >
          <span>Parameters &amp; versions</span>
          <span className="text-slack-textMuted">{drawerOpen ? '▼' : '▲'}</span>
        </button>
        {drawerOpen && (
          <div className="px-3 pb-3 pt-1 max-h-[220px] overflow-y-auto">
            <div className="flex flex-wrap gap-x-8 gap-y-4 items-start">
              <section className="min-w-[200px] flex-1">
                <h3 className="font-semibold text-slack-text mb-2 text-sm">Parameters</h3>
                {params.length === 0 ? (
                  <p className="text-slack-textMuted text-xs">No top-level variables found.</p>
                ) : (
                  <div className="flex flex-wrap gap-x-4 gap-y-3">
                    {params.map((p) => {
                      const num = parseFloat(paramOverrides[p.name] ?? p.value);
                      const useSlider = p.min != null && p.max != null && Number.isFinite(num);
                      return (
                        <label key={p.name} className="block text-xs min-w-[140px] max-w-[220px]">
                          <span className="text-slack-textMuted">{p.name}</span>
                          {useSlider ? (
                            <input
                              type="range"
                              min={p.min}
                              max={p.max}
                              step={p.step ?? 1}
                              value={paramOverrides[p.name] ?? p.value}
                              onChange={(e) =>
                                setParamOverrides((prev) => ({ ...prev, [p.name]: e.target.value }))
                              }
                              className="w-full mt-1"
                            />
                          ) : (
                            <input
                              type="text"
                              value={paramOverrides[p.name] ?? p.value}
                              onChange={(e) =>
                                setParamOverrides((prev) => ({ ...prev, [p.name]: e.target.value }))
                              }
                              className="mt-1 w-full px-2 py-1 border border-slack-border rounded bg-slack-bg text-slack-text font-mono"
                            />
                          )}
                        </label>
                      );
                    })}
                  </div>
                )}
              </section>

              <section className="min-w-[180px] max-w-[260px]">
                <h3 className="font-semibold text-slack-text mb-2 text-sm">Versions</h3>
                <div className="flex gap-1 mb-2">
                  <input
                    type="text"
                    value={versionLabel}
                    onChange={(e) => setVersionLabel(e.target.value)}
                    placeholder="Label"
                    className="flex-1 px-2 py-1 text-xs border border-slack-border rounded bg-slack-bg"
                  />
                  <button
                    type="button"
                    onClick={() => void saveVersion()}
                    className="text-xs px-2 py-1 rounded bg-slack-bgHover shrink-0"
                  >
                    Save
                  </button>
                </div>
                <ul className="space-y-1 text-xs max-h-[120px] overflow-y-auto">
                  {versions.map((v) => (
                    <li key={v.id}>
                      <button
                        type="button"
                        className="text-left text-slack-accent hover:underline w-full truncate"
                        onClick={() => void restoreVersion(v.id)}
                      >
                        {v.label || v.id}
                      </button>
                    </li>
                  ))}
                </ul>
              </section>

              <section className="min-w-[160px] max-w-[240px]">
                <h3 className="font-semibold text-slack-text mb-1 text-sm">Export</h3>
                <p className="text-xs text-slack-textMuted">
                  STL via Render. STEP requires FreeCAD (optional in Settings). Save SCAD from the editor and use Render
                  for STL.
                </p>
              </section>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
