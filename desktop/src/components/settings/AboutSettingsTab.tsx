import { useState, useEffect } from 'react';
import { APP_INFO, TECH_STACK, getAppVersion } from '../../utils/appInfo';
import {
  getUpdateChannelLabel,
} from '../../utils/appUpdater';
import { useAppUpdaterStore } from '../../stores/appUpdaterStore';
import { getHubBaseURL, getHubWebSocketURL } from '../../config/hubUrl';
import { openExternalLink, type SettingsTabProps } from './settingsShared';

export function AboutSettingsTab({
  isActive,
  onRerunSetup,
}: SettingsTabProps & { onRerunSetup?: () => void }) {
  const [appVersion, setAppVersion] = useState<string>('1.0.0');
  const updaterStatus = useAppUpdaterStore((state) => state.status);
  const update = useAppUpdaterStore((state) => state.update);
  const updateProgress = useAppUpdaterStore((state) => state.progress);
  const updateError = useAppUpdaterStore((state) => state.error);
  const checkForUpdates = useAppUpdaterStore((state) => state.check);
  const restartToUpdate = useAppUpdaterStore((state) => state.restartToUpdate);

  useEffect(() => {
    if (!isActive) return;
    void getAppVersion().then(setAppVersion);
  }, [isActive]);


  if (!isActive) return null;
  return (
  <div className="space-y-6">
    {/* App Info */}
    <div>
      <h3 className="text-lg font-semibold text-slack-text mb-2">{APP_INFO.name}</h3>
      <p className="text-slack-textMuted mb-4">{APP_INFO.description}</p>
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div>
          <span className="text-slack-textMuted">Version:</span>
          <span className="ml-2 text-slack-text">{appVersion}</span>
        </div>
        <div>
          <span className="text-slack-textMuted">License:</span>
          <span className="ml-2 text-slack-text">{APP_INFO.license}</span>
        </div>
        <div className="col-span-2">
          <span className="text-slack-textMuted">Update channel:</span>
          <span className="ml-2 text-slack-text">
            {getUpdateChannelLabel(appVersion, update?.policy)}
          </span>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <button
          type="button"
          disabled={updaterStatus === 'checking' || updaterStatus === 'downloading' || updaterStatus === 'installing'}
          onClick={() => void checkForUpdates(true)}
          className="px-3 py-1.5 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors disabled:opacity-50"
        >
          {updaterStatus === 'checking' ? 'Checking…' : 'Check for updates'}
        </button>
        {updaterStatus === 'ready' && update && (
          <button
            type="button"
            onClick={() => void restartToUpdate()}
            className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            Restart to install {update.version}
          </button>
        )}
        {updaterStatus === 'current' && (
          <span className="text-sm text-slack-textMuted">You are on the latest eligible version.</span>
        )}
        {updaterStatus === 'downloading' && (
          <span className="text-sm text-slack-textMuted">
            Downloading {update?.version}… {updateProgress === null ? '' : `${updateProgress}%`}
          </span>
        )}
        {updaterStatus === 'unsupported' && (
          <span className="text-sm text-slack-textMuted">Linux packages currently update manually.</span>
        )}
        {updateError && <span className="text-sm text-red-400">{updateError}</span>}
      </div>
    </div>

    {onRerunSetup && (
      <div>
        <h3 className="text-lg font-semibold text-slack-text mb-2">Setup</h3>
        <p className="text-sm text-slack-textMuted mb-4">
          Re-open the first-run wizard to change focus track, AI backend, agents, and domain packs.
          Existing API keys and unrelated providers are preserved when possible.
        </p>
        <button
          type="button"
          onClick={onRerunSetup}
          className="px-3 py-1.5 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors"
        >
          Run setup again
        </button>
      </div>
    )}

    <div>
      <h3 className="text-lg font-semibold text-slack-text mb-2">Hub connection</h3>
      <div className="space-y-2 text-sm">
        <div className="p-3 bg-slack-bgHover rounded">
          <span className="text-slack-textMuted">HTTP:</span>
          <span className="ml-2 text-slack-text font-mono break-all">{getHubBaseURL()}</span>
        </div>
        <div className="p-3 bg-slack-bgHover rounded">
          <span className="text-slack-textMuted">WebSocket:</span>
          <span className="ml-2 text-slack-text font-mono break-all">{getHubWebSocketURL()}</span>
        </div>
      </div>
    </div>

    {/* Technology Stack */}
    <div>
      <h4 className="text-md font-semibold text-slack-text mb-3">Technology Stack</h4>
      <div className="flex flex-wrap gap-2">
        {TECH_STACK.map((tech) => (
          <span
            key={tech}
            className="px-3 py-1 bg-slack-bgHover text-slack-text text-sm rounded-full border border-slack-border"
          >
            {tech}
          </span>
        ))}
      </div>
    </div>

    {/* Links */}
    <div>
      <h4 className="text-md font-semibold text-slack-text mb-3">Links</h4>
      <div className="space-y-2">
        <button
          onClick={() => openExternalLink(APP_INFO.repository)}
          className="block text-left text-slack-accent hover:text-slack-accentHover transition-colors"
        >
          📁 GitHub Repository
        </button>
        <button
          onClick={() => openExternalLink(APP_INFO.documentation)}
          className="block text-left text-slack-accent hover:text-slack-accentHover transition-colors"
        >
          📚 Documentation
        </button>
      </div>
    </div>

    {/* Copyright */}
    <div className="pt-4 border-t border-slack-border">
      <p className="text-xs text-slack-textMuted">
        © 2025 {APP_INFO.author}. Licensed under {APP_INFO.license}.
      </p>
    </div>
  </div>
  );
}
