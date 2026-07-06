import { useCallback, useEffect, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
import { ChatAPI } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { useToastStore } from '../stores/toastStore';
import { useEditorStore } from '../stores/editorStore';

const api = new ChatAPI(getHubBaseURL());

interface StructureAtom {
  x: number;
  y: number;
  z: number;
  element: string;
}

interface StructureWorkbenchProps {
  workspaceId: string;
  structurePath: string;
  initialContent?: string;
  tabId: string;
}

function parsePdbAtoms(pdbText: string): StructureAtom[] {
  const atoms: StructureAtom[] = [];
  for (const line of pdbText.split('\n')) {
    if (!line.startsWith('ATOM') && !line.startsWith('HETATM')) continue;
    const x = parseFloat(line.substring(30, 38));
    const y = parseFloat(line.substring(38, 46));
    const z = parseFloat(line.substring(46, 54));
    if (Number.isNaN(x) || Number.isNaN(y) || Number.isNaN(z)) continue;
    const element =
      line.substring(76, 78).trim() ||
      line.substring(12, 16).trim().replace(/[0-9]/g, '').charAt(0) ||
      'C';
    atoms.push({ x, y, z, element });
  }
  return atoms;
}

function elementColor(element: string): number {
  switch (element.toUpperCase()) {
    case 'N':
      return 0x3050f8;
    case 'O':
      return 0xff0d0d;
    case 'S':
      return 0xffff30;
    default:
      return 0x909090;
  }
}

export function StructureWorkbench({
  workspaceId,
  structurePath,
  initialContent = '',
  tabId,
}: StructureWorkbenchProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const sceneRef = useRef<{
    renderer: THREE.WebGLRenderer;
    scene: THREE.Scene;
    camera: THREE.PerspectiveCamera;
    controls: OrbitControls;
    group: THREE.Group;
  } | null>(null);

  const [content, setContent] = useState('');
  const [renderStyle, setRenderStyle] = useState<'cartoon' | 'stick' | 'surface'>('stick');
  const [loadedPath, setLoadedPath] = useState<string | null>(null);
  const { addToast } = useToastStore();

  const normalizedPath = structurePath.replace(/\\/g, '/').replace(/^\/+/, '');

  useEffect(() => {
    let cancelled = false;
    setLoadedPath(null);
    void (async () => {
      try {
        const diskContent = await api.fetchFileContent(workspaceId, normalizedPath);
        if (cancelled) return;
        setContent(diskContent);
        setLoadedPath(normalizedPath);
        useEditorStore.getState().updateTabContent(tabId, diskContent);
      } catch (err) {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : 'Failed to load structure file';
        addToast({ type: 'error', title: 'Structure', message });
        setContent(initialContent);
        setLoadedPath(normalizedPath);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [tabId, normalizedPath, workspaceId, addToast, initialContent]);

  const renderStructure = useCallback(
    (pdbText: string, style: 'cartoon' | 'stick' | 'surface') => {
      const ctx = sceneRef.current;
      if (!ctx) return;
      const atoms = parsePdbAtoms(pdbText);
      while (ctx.group.children.length > 0) {
        const child = ctx.group.children[0];
        ctx.group.remove(child);
        if (child instanceof THREE.Mesh) {
          child.geometry.dispose();
          (child.material as THREE.Material).dispose();
        } else if (child instanceof THREE.Line) {
          child.geometry.dispose();
          (child.material as THREE.Material).dispose();
        }
      }
      if (atoms.length === 0) return;

      const center = new THREE.Vector3();
      for (const a of atoms) center.add(new THREE.Vector3(a.x, a.y, a.z));
      center.divideScalar(atoms.length);
      ctx.group.position.set(-center.x, -center.y, -center.z);

      if (style === 'cartoon') {
        const caAtoms = atoms.filter((_, i) => i % 4 === 1 || atoms.length < 20);
        const points = (caAtoms.length > 2 ? caAtoms : atoms).map(
          (a) => new THREE.Vector3(a.x, a.y, a.z),
        );
        if (points.length >= 2) {
          const geometry = new THREE.BufferGeometry().setFromPoints(points);
          const material = new THREE.LineBasicMaterial({ color: 0x4ade80, linewidth: 2 });
          ctx.group.add(new THREE.Line(geometry, material));
        }
      }

      const radius = style === 'surface' ? 1.2 : 0.35;
      for (const a of atoms) {
        const geometry = new THREE.SphereGeometry(radius, 12, 12);
        const material = new THREE.MeshStandardMaterial({
          color: elementColor(a.element),
          metalness: 0.1,
          roughness: 0.7,
        });
        const mesh = new THREE.Mesh(geometry, material);
        mesh.position.set(a.x, a.y, a.z);
        ctx.group.add(mesh);
      }

      const box = new THREE.Box3().setFromObject(ctx.group);
      const size = box.getSize(new THREE.Vector3()).length() || 10;
      ctx.camera.position.set(size, size * 0.8, size);
      ctx.controls.target.set(0, 0, 0);
      ctx.controls.update();
    },
    [],
  );

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const renderer = new THREE.WebGLRenderer({ canvas, antialias: true });
    renderer.setPixelRatio(window.devicePixelRatio);
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x0f1419);
    const camera = new THREE.PerspectiveCamera(50, 1, 0.1, 10000);
    const controls = new OrbitControls(camera, canvas);
    controls.enableDamping = true;
    scene.add(new THREE.AmbientLight(0xffffff, 0.6));
    const dir = new THREE.DirectionalLight(0xffffff, 0.8);
    dir.position.set(5, 10, 7);
    scene.add(dir);
    const group = new THREE.Group();
    scene.add(group);
    sceneRef.current = { renderer, scene, camera, controls, group };

    const resize = () => {
      const parent = canvas.parentElement;
      if (!parent) return;
      const w = parent.clientWidth;
      const h = parent.clientHeight;
      renderer.setSize(w, h, false);
      camera.aspect = w / Math.max(h, 1);
      camera.updateProjectionMatrix();
    };
    resize();
    window.addEventListener('resize', resize);
    let frame = 0;
    const animate = () => {
      frame = requestAnimationFrame(animate);
      controls.update();
      renderer.render(scene, camera);
    };
    animate();

    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener('resize', resize);
      renderer.dispose();
      sceneRef.current = null;
    };
  }, []);

  useEffect(() => {
    if (loadedPath !== normalizedPath || !content) return;
    renderStructure(content, renderStyle);
  }, [content, loadedPath, normalizedPath, renderStyle, renderStructure]);

  const handleEditorChange = useCallback(
    (value: string | undefined) => {
      const next = value ?? '';
      setContent(next);
      useEditorStore.getState().updateTabContent(tabId, next);
      useEditorStore.getState().markTabDirty(tabId, true);
      renderStructure(next, renderStyle);
    },
    [tabId, renderStyle, renderStructure],
  );

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-2 border-b border-[var(--border)] px-3 py-2 text-sm">
        <span className="text-[var(--text-muted)]">Structure</span>
        <span className="truncate font-mono text-xs">{normalizedPath}</span>
        <div className="ml-auto flex gap-1">
          {(['stick', 'cartoon', 'surface'] as const).map((style) => (
            <button
              key={style}
              type="button"
              className={`rounded px-2 py-0.5 text-xs ${
                renderStyle === style
                  ? 'bg-[var(--accent)] text-white'
                  : 'bg-[var(--bg-secondary)] text-[var(--text-muted)]'
              }`}
              onClick={() => setRenderStyle(style)}
            >
              {style}
            </button>
          ))}
        </div>
      </div>
      <div className="grid min-h-0 flex-1 grid-cols-2">
        <div className="min-h-0 border-r border-[var(--border)]">
          <Editor
            height="100%"
            defaultLanguage="plaintext"
            value={content}
            onChange={handleEditorChange}
            options={{ minimap: { enabled: false }, wordWrap: 'on', fontSize: 12 }}
          />
        </div>
        <div className="relative min-h-0 bg-[#0f1419]">
          <canvas ref={canvasRef} className="h-full w-full" />
        </div>
      </div>
    </div>
  );
}
