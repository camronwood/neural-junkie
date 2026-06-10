import { useState, useEffect } from 'react';
import { APP_INFO, TECH_STACK, getAppVersion } from '../../utils/appInfo';
import {
  checkForAppUpdate,
  getUpdateChannelLabel,
  installAppUpdate,
} from '../../utils/appUpdater';
import { getHubBaseURL, getHubWebSocketURL } from '../../config/hubUrl';
import { openExternalLink, type SettingsTabProps } from './settingsShared';

export function AboutSettingsTab({ isActive }: SettingsTabProps) {
  const [appVersion, setAppVersion] = useState<string>('1.0.0');
  const [updateCheckStatus, setUpdateCheckStatus] = useState<string | null>(null);
  const [updateChecking, setUpdateChecking] = useState(false);
  const [updateInstalling, setUpdateInstalling] = useState(false);
  const [updateProgress, setUpdateProgress] = useState(0);
  const [pendingUpdateVersion, setPendingUpdateVersion] = useState<string | null>(null);

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
          <span className="ml-2 text-slack-text">{getUpdateChannelLabel(appVersion)}</span>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <button
          type="button"
          disabled={updateChecking || updateInstalling}
          onClick={async () => {
            setUpdateChecking(true);
            setUpdateCheckStatus(null);
            setPendingUpdateVersion(null);
            try {
              const result = await checkForAppUpdate();
              if (result.status === 'available') {
                setPendingUpdateVersion(result.update.version ?? 'new version');
                setUpdateCheckStatus(`Update available: ${result.update.version ?? 'new version'}`);
              } else if (result.status === 'current') {
                setUpdateCheckStatus('You are on the latest version.');
              } else {
                setUpdateCheckStatus(result.reason);
              }
            } catch (e) {
              setUpdateCheckStatus(e instanceof Error ? e.message : 'Update check failed');
            } finally {
              setUpdateChecking(false);
            }
          }}
          className="px-3 py-1.5 text-sm bg-slack-accent text-white rounded hover:bg-slack-accentHover transition-colors disabled:opacity-50"
        >
          {updateChecking ? 'Checking…' : 'Check for updates'}
        </button>
        {pendingUpdateVersion && (
          <button
            type="button"
            disabled={updateInstalling}
            onClick={async () => {
              setUpdateInstalling(true);
              setUpdateCheckStatus(null);
              try {
                await installAppUpdate(setUpdateProgress);
              } catch (e) {
                setUpdateCheckStatus(e instanceof Error ? e.message : 'Update install failed');
                setUpdateInstalling(false);
              }
            }}
            className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            {updateInstalling ? `Installing… ${updateProgress}%` : `Install ${pendingUpdateVersion}`}
          </button>
        )}
        {updateCheckStatus && (
          <span className="text-sm text-slack-textMuted">{updateCheckStatus}</span>
        )}
      </div>
    </div>

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
