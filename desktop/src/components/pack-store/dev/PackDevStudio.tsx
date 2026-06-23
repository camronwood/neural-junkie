import { useCallback, useState } from 'react';
import { isTauriRuntime } from '../../../utils/promptAttachments';
import { PackScaffoldWizard } from './PackScaffoldWizard';
import { PackManifestEditor } from './PackManifestEditor';
import { PackDevLink } from './PackDevLink';
import { PackTestPanel } from './PackTestPanel';
import { CustomPackInstall } from '../CustomPackInstall';

type DevTab = 'create' | 'edit' | 'link' | 'test';

export function PackDevStudio() {
  const [tab, setTab] = useState<DevTab>('create');
  const [packDir, setPackDir] = useState<string | null>(null);
  const [editorYaml, setEditorYaml] = useState<string | undefined>();

  const onScaffolded = useCallback((dir: string, yaml: string) => {
    setPackDir(dir);
    setEditorYaml(yaml);
    setTab('edit');
  }, []);

  if (!isTauriRuntime()) {
    return (
      <div className="space-y-4">
        <CustomPackInstall />
        <div className="border border-slack-border rounded-lg p-4 bg-slack-bgHover/20 text-sm text-slack-textMuted">
          Pack dev studio requires the <strong className="text-slack-text">desktop app</strong> for folder access,
          scaffold creation, and zip builds.
        </div>
      </div>
    );
  }

  const tabs: Array<{ id: DevTab; label: string }> = [
    { id: 'create', label: 'Create' },
    { id: 'edit', label: 'Edit' },
    { id: 'link', label: 'Dev link' },
    { id: 'test', label: 'Test' },
  ];

  return (
    <div className="border border-teal-800/30 rounded-xl p-5 bg-teal-950/10 space-y-5">
      <div>
        <h3 className="text-lg font-semibold text-teal-100">Pack dev studio</h3>
        <p className="text-sm text-slack-textMuted mt-1">
          Scaffold, edit, validate, and test customer packs before shipping a zip to your organization.
        </p>
      </div>

      <CustomPackInstall />

      <div className="flex flex-wrap gap-2 border-b border-slack-border pb-2">
        {tabs.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => setTab(t.id)}
            className={`px-3 py-1.5 text-xs font-medium rounded-t transition-colors ${
              tab === t.id
                ? 'text-teal-200 border-b-2 border-teal-500 -mb-[2px]'
                : 'text-slack-textMuted hover:text-slack-text'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'create' && <PackScaffoldWizard onScaffolded={onScaffolded} />}
      {tab === 'edit' && (
        <PackManifestEditor
          packDir={packDir}
          initialYaml={editorYaml}
          onPackDirChange={setPackDir}
        />
      )}
      {tab === 'link' && (
        <PackDevLink
          packDir={packDir}
          onPackDirChange={setPackDir}
          onValidated={() => setTab('test')}
        />
      )}
      {tab === 'test' && <PackTestPanel />}
    </div>
  );
}
