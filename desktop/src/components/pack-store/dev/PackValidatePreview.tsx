import type { PackValidationReport } from '../../../api/chatAPI';
import { RichMarkdownView } from '../../RichMarkdownView';

interface PackValidatePreviewProps {
  report: PackValidationReport | null;
  loading?: boolean;
  error?: string | null;
}

export function PackValidatePreview({ report, loading, error }: PackValidatePreviewProps) {
  if (loading) {
    return <p className="text-xs text-slack-textMuted">Validating…</p>;
  }
  if (error) {
    return <p className="text-xs text-red-400">{error}</p>;
  }
  if (!report) {
    return <p className="text-xs text-slack-textMuted">Run validate to preview manifest, assets, and overlays.</p>;
  }

  return (
    <div className="space-y-4 text-xs">
      <div
        className={`rounded-lg border px-3 py-2 ${
          report.valid
            ? 'border-emerald-700/50 bg-emerald-950/30 text-emerald-200'
            : 'border-red-700/50 bg-red-950/30 text-red-200'
        }`}
      >
        {report.valid ? 'Validation passed' : 'Validation failed'}
        {report.errors && report.errors.length > 0 && (
          <ul className="mt-2 list-disc pl-4 text-red-300">
            {report.errors.map((e) => (
              <li key={e}>{e}</li>
            ))}
          </ul>
        )}
      </div>

      {report.warnings && report.warnings.length > 0 && (
        <div className="rounded-lg border border-amber-700/40 bg-amber-950/20 px-3 py-2 text-amber-200">
          <p className="font-medium">Warnings</p>
          <ul className="mt-1 list-disc pl-4">
            {report.warnings.map((w) => (
              <li key={w}>{w}</li>
            ))}
          </ul>
        </div>
      )}

      {report.manifest && (
        <div>
          <p className="font-medium text-slack-text mb-1">Manifest</p>
          <p className="text-slack-textMuted">
            <span className="text-slack-text">{report.manifest.title}</span> ({report.manifest.id}
            {report.manifest.version ? ` v${report.manifest.version}` : ''})
          </p>
          {report.manifest.capabilities && report.manifest.capabilities.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {report.manifest.capabilities.map((cap) => (
                <span key={cap} className="px-2 py-0.5 rounded bg-slack-bgHover text-slack-textMuted border border-slack-border">
                  {cap}
                </span>
              ))}
            </div>
          )}
        </div>
      )}

      {report.requires_packs && report.requires_packs.length > 0 && (
        <div>
          <p className="font-medium text-slack-text mb-1">Requires packs</p>
          <ul className="space-y-1">
            {report.requires_packs.map((req) => (
              <li key={req.id} className="text-slack-textMuted">
                <span className="text-slack-text">{req.id}</span>
                {' — '}
                {req.enabled ? 'enabled' : req.installed ? 'installed (not enabled)' : 'not installed'}
              </li>
            ))}
          </ul>
        </div>
      )}

      {report.resolved_overlay && Object.keys(report.resolved_overlay).length > 0 && (
        <div>
          <p className="font-medium text-slack-text mb-1">Resolved overlay</p>
          <ul className="space-y-1 font-mono text-[11px] text-slack-textMuted break-all">
            {Object.entries(report.resolved_overlay).map(([k, v]) => (
              <li key={k}>
                <span className="text-teal-300">{k}</span>: {v}
              </li>
            ))}
          </ul>
        </div>
      )}

      {report.preview?.effective_capabilities && report.preview.effective_capabilities.length > 0 && (
        <div>
          <p className="font-medium text-slack-text mb-1">Effective capabilities if enabled</p>
          <div className="flex flex-wrap gap-1">
            {report.preview.effective_capabilities.map((cap) => (
              <span key={cap} className="px-2 py-0.5 rounded bg-teal-950/40 text-teal-200 border border-teal-800/40">
                {cap}
              </span>
            ))}
          </div>
        </div>
      )}

      {report.preview?.agents && report.preview.agents.length > 0 && (
        <div>
          <p className="font-medium text-slack-text mb-1">Agents</p>
          <ul className="text-slack-textMuted">
            {report.preview.agents.map((a) => (
              <li key={a.type}>
                {a.name ? `${a.name} (${a.type})` : a.type}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div>
        <p className="font-medium text-slack-text mb-1">Assets</p>
        <p className="text-slack-textMuted">
          Workspace guide: {report.assets.workspace_guide_found ? 'found' : 'missing'}
          {report.assets.workspace_guide_path ? ` (${report.assets.workspace_guide_path})` : ''}
        </p>
        <p className="text-slack-textMuted">Runbooks: {report.assets.runbooks_count} matched</p>
      </div>

      {report.assets.workspace_guide_preview && (
        <div className="border border-slack-border rounded-lg p-3 max-h-48 overflow-y-auto bg-slack-bg">
          <p className="font-medium text-slack-text mb-2">Workspace guide preview</p>
          <RichMarkdownView content={report.assets.workspace_guide_preview} />
        </div>
      )}
    </div>
  );
}
