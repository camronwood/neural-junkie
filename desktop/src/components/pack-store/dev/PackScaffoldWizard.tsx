import { useCallback, useMemo, useState } from 'react';
import { invoke } from '@tauri-apps/api/tauri';
import { usePacksStore } from '../../../stores/packsStore';
import { isTauriRuntime } from '../../../utils/promptAttachments';
import { ipcWorkspaceRoots } from '../../../utils/ipcWorkspaceRoots';
import { GENERIC_OVERLAY_FIELD_DOCS } from './packDevConstants';

function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === 'string') return e;
  return 'Scaffold failed';
}

interface PackScaffoldWizardProps {
  onScaffolded: (outputDir: string, yaml: string) => void;
}

export function PackScaffoldWizard({ onScaffolded }: PackScaffoldWizardProps) {
  const catalog = usePacksStore((s) => s.catalog);
  const [step, setStep] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [outputDir, setOutputDir] = useState<string | null>(null);

  const [id, setId] = useState('');
  const [version, setVersion] = useState('1.0.0');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [publisher, setPublisher] = useState('');
  const [requiresPacks, setRequiresPacks] = useState<string[]>(['life-sciences']);
  const [overlay, setOverlay] = useState<Record<string, string>>({});
  const [workspaceGuide, setWorkspaceGuide] = useState('assets/WORKSPACE.md');
  const [runbooksGlob, setRunbooksGlob] = useState('assets/runbooks/*.md');
  const [toolbarChipLabel, setToolbarChipLabel] = useState('');
  const [toolbarChipIcon, setToolbarChipIcon] = useState('');

  const catalogChoices = useMemo(
    () => catalog.filter((c) => c.builtin).map((c) => ({ id: c.id, title: c.title })),
    [catalog],
  );

  const toggleRequires = (packId: string) => {
    setRequiresPacks((prev) =>
      prev.includes(packId) ? prev.filter((p) => p !== packId) : [...prev, packId],
    );
  };

  const pickOutput = useCallback(async () => {
    if (!isTauriRuntime()) {
      setError('Scaffold requires the desktop app.');
      return;
    }
    setError(null);
    try {
      const selected = await invoke<string | null>('pick_pack_directory', {
        title: 'Choose parent folder for new pack',
      });
      if (selected && id.trim()) {
        setOutputDir(`${selected}/${id.trim()}`);
      } else if (selected) {
        setOutputDir(selected);
      }
    } catch (e) {
      setError(errorMessage(e));
    }
  }, [id]);

  const handleCreate = useCallback(async () => {
    if (!isTauriRuntime()) {
      setError('Scaffold requires the desktop app.');
      return;
    }
    if (!outputDir || !id.trim() || !title.trim()) {
      setError('Output folder, id, and title are required.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const yaml = await invoke<string>('write_pack_scaffold', {
        req: {
          output_dir: outputDir,
          id: id.trim(),
          version: version.trim() || '1.0.0',
          title: title.trim(),
          description: description.trim() || null,
          publisher: publisher.trim() || null,
          requires_packs: requiresPacks,
          capabilities: ['customer-pack'],
          settings_overlay: overlay,
          workspace_guide: workspaceGuide.trim() || null,
          runbooks_glob: runbooksGlob.trim() || null,
          toolbar_chip_label: toolbarChipLabel.trim() || null,
          toolbar_chip_icon: toolbarChipIcon.trim() || null,
        },
        ...ipcWorkspaceRoots(),
      });
      onScaffolded(outputDir, yaml);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }, [
    outputDir,
    id,
    version,
    title,
    description,
    publisher,
    requiresPacks,
    overlay,
    workspaceGuide,
    runbooksGlob,
    toolbarChipLabel,
    toolbarChipIcon,
    onScaffolded,
  ]);

  const steps = ['Identity', 'Dependencies', 'Overlay', 'Output'];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        {steps.map((label, i) => (
          <button
            key={label}
            type="button"
            onClick={() => setStep(i)}
            className={`px-2 py-1 text-[11px] rounded border ${
              step === i
                ? 'border-teal-600/60 text-teal-200 bg-teal-950/30'
                : 'border-slack-border text-slack-textMuted'
            }`}
          >
            {i + 1}. {label}
          </button>
        ))}
      </div>

      {error && <p className="text-xs text-red-400">{error}</p>}

      {step === 0 && (
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-xs text-slack-textMuted">
            Pack id
            <input
              value={id}
              onChange={(e) => setId(e.target.value)}
              placeholder="acme-lab"
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text"
            />
          </label>
          <label className="text-xs text-slack-textMuted">
            Version
            <input
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text"
            />
          </label>
          <label className="text-xs text-slack-textMuted sm:col-span-2">
            Title
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text"
            />
          </label>
          <label className="text-xs text-slack-textMuted sm:col-span-2">
            Description
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text"
            />
          </label>
          <label className="text-xs text-slack-textMuted sm:col-span-2">
            Publisher
            <input
              value={publisher}
              onChange={(e) => setPublisher(e.target.value)}
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text"
            />
          </label>
          <label className="text-xs text-slack-textMuted">
            Sidebar chip label (optional, max 3 letters)
            <input
              value={toolbarChipLabel}
              onChange={(e) => setToolbarChipLabel(e.target.value.slice(0, 3))}
              placeholder="LAB"
              maxLength={3}
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text font-mono uppercase"
            />
          </label>
          <label className="text-xs text-slack-textMuted">
            Sidebar chip icon (optional, pack-relative path)
            <input
              value={toolbarChipIcon}
              onChange={(e) => setToolbarChipIcon(e.target.value)}
              placeholder="assets/icons/chip.png"
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text font-mono"
            />
          </label>
        </div>
      )}

      {step === 1 && (
        <div className="space-y-2">
          <p className="text-xs text-slack-textMuted">Required domain packs (must be enabled before yours):</p>
          {catalogChoices.map((c) => (
            <label key={c.id} className="flex items-center gap-2 text-sm text-slack-text">
              <input
                type="checkbox"
                checked={requiresPacks.includes(c.id)}
                onChange={() => toggleRequires(c.id)}
              />
              {c.title} ({c.id})
            </label>
          ))}
        </div>
      )}

      {step === 2 && (
        <div className="space-y-3">
          <p className="text-xs text-slack-textMuted">
            Generic custom packs use workspace guides and runbooks below. Biology tool overlays (
            <code className="font-mono text-teal-300/90">secondary_analysis_tools_path</code>,{' '}
            <code className="font-mono text-teal-300/90">cumulative_qc_dir</code>, etc.) belong in your
            org-specific pack repo (e.g. brightest-bio-lab) with{' '}
            <code className="font-mono text-teal-300/90">secondary-analysis-api</code> — add them in the YAML
            editor after scaffold.
          </p>
          {GENERIC_OVERLAY_FIELD_DOCS.map((f) => (
            <label key={f.key} className="block text-xs text-slack-textMuted">
              {f.key}
              <span className="block text-[11px] text-slack-textMuted/80">{f.hint}</span>
              <input
                value={overlay[f.key] ?? ''}
                onChange={(e) =>
                  setOverlay((prev) => {
                    const next = { ...prev };
                    if (e.target.value.trim()) next[f.key] = e.target.value;
                    else delete next[f.key];
                    return next;
                  })
                }
                className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text font-mono"
              />
            </label>
          ))}
          <label className="block text-xs text-slack-textMuted">
            assets.workspace_guide
            <input
              value={workspaceGuide}
              onChange={(e) => setWorkspaceGuide(e.target.value)}
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text font-mono"
            />
          </label>
          <label className="block text-xs text-slack-textMuted">
            assets.runbooks_glob
            <input
              value={runbooksGlob}
              onChange={(e) => setRunbooksGlob(e.target.value)}
              className="mt-1 w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text font-mono"
            />
          </label>
        </div>
      )}

      {step === 3 && (
        <div className="space-y-3">
          <p className="text-xs text-slack-textMuted">
            Output folder (will contain pack.yaml and assets/). Typically{' '}
            <code className="text-teal-300">…/{'{id}'}/</code>.
          </p>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => void pickOutput()}
              className="px-3 py-1.5 text-xs rounded border border-slack-border text-slack-text hover:bg-slack-bgHover"
            >
              Choose parent folder…
            </button>
          </div>
          <input
            value={outputDir ?? ''}
            onChange={(e) => setOutputDir(e.target.value || null)}
            placeholder="/path/to/my-pack"
            className="w-full rounded border border-slack-border bg-slack-bg px-2 py-1.5 text-sm text-slack-text font-mono"
          />
          <button
            type="button"
            disabled={busy}
            onClick={() => void handleCreate()}
            className="px-3 py-1.5 text-xs font-medium rounded-lg border border-teal-600/50 text-teal-200 hover:bg-teal-900/40 disabled:opacity-40"
          >
            {busy ? 'Creating…' : 'Create scaffold & open editor'}
          </button>
        </div>
      )}

      <div className="flex justify-between">
        <button
          type="button"
          disabled={step === 0}
          onClick={() => setStep((s) => Math.max(0, s - 1))}
          className="text-xs text-slack-textMuted hover:text-slack-text disabled:opacity-30"
        >
          Back
        </button>
        <button
          type="button"
          disabled={step >= steps.length - 1}
          onClick={() => setStep((s) => Math.min(steps.length - 1, s + 1))}
          className="text-xs text-teal-300 hover:text-teal-200 disabled:opacity-30"
        >
          Next
        </button>
      </div>
    </div>
  );
}
