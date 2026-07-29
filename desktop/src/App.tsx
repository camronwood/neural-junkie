import { useState, useEffect } from 'react';
import { LoginScreen } from './components/LoginScreen';
import { ChatWindow } from './components/ChatWindow';
import { SettingsModal, type SettingsTab } from './components/SettingsModal';
import { MarkdownPreview } from './components/MarkdownPreview';
import { LoadingScreen } from './components/LoadingScreen';
import { SetupWizard } from './components/SetupWizard';
import { UpdateBanner } from './components/UpdateBanner';
import { useSettingsStore } from './stores/settingsStore';
import { shallow } from 'zustand/shallow';
import { useChatStore } from './stores/chatStore';
import { loadCredentials } from './utils/secureStorage';
import { setHubSessionToken } from './config/hubUrl';
import { ChatAPI } from './api/chatAPI';
import { getHubBaseURL } from './config/hubUrl';
import { loadConnectionSettings } from './stores/connectionStore';
import { applyMermaidTheme } from './utils/mermaidConfig';
import { isTauriRuntime } from './utils/promptAttachments';
import { DesktopOnlyGate } from './components/DesktopOnlyGate';
import { installExternalLinkClickInterceptor } from './utils/openExternalLink';

type AppPhase = 'loading' | 'setup' | 'login' | 'chat';

function isMarkdownPreviewFromUrl(): boolean {
  const params = new URLSearchParams(window.location.search);
  return (
    params.get('preview') === 'true' &&
    Boolean(params.get('workspace')?.trim()) &&
    Boolean(params.get('path')?.trim())
  );
}

function App() {
  const [phase, setPhase] = useState<AppPhase>('loading');
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [settingsInitialTab, setSettingsInitialTab] = useState<SettingsTab | undefined>();
  const [isPreviewMode, setIsPreviewMode] = useState(false);
  const [previewParams, setPreviewParams] = useState<{ workspaceId: string; filePath: string } | null>(null);
  const { settings, loadSettings, syncUserRulesFromHub } = useSettingsStore();
  const { setUsername, setChannel, setServerAddr } = useChatStore(
    (s) => ({
      setUsername: s.setUsername,
      setChannel: s.setChannel,
      setServerAddr: s.setServerAddr,
    }),
    shallow
  );

  const serverAddr = getHubBaseURL();

  // Load settings and connection on app start
  useEffect(() => {
    loadSettings();
    void loadConnectionSettings().then((conn) => {
      setServerAddr(conn.hubUrl);
    });
  }, [loadSettings, setServerAddr]);

  // Keep http(s) clicks in the OS browser — never navigate the Tauri webview.
  useEffect(() => installExternalLinkClickInterceptor(), []);

  // Return to login when hub rejects the session (401).
  useEffect(() => {
    const onUnauthorized = () => {
      setHubSessionToken(null);
      setPhase('login');
    };
    window.addEventListener('nj-hub-unauthorized', onUnauthorized);
    return () => window.removeEventListener('nj-hub-unauthorized', onUnauthorized);
  }, []);

  // Check for preview mode on mount
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const isPreview = urlParams.get('preview') === 'true';
    const workspaceId = urlParams.get('workspace');
    const filePath = urlParams.get('path');

    if (isPreview && workspaceId && filePath) {
      setIsPreviewMode(true);
      setPreviewParams({ workspaceId, filePath });
    }
  }, []);

  // Apply font size and color theme to document root
  useEffect(() => {
    const theme = settings.colorTheme ?? 'slack';
    const root = document.documentElement;
    root.style.setProperty('--app-font-size', `${settings.fontSize}px`);
    root.dataset.theme = theme;
    document.body.className = `font-scope-${settings.fontSizeScope}`;
    applyMermaidTheme(theme);
  }, [settings.fontSize, settings.fontSizeScope, settings.colorTheme]);

  async function onServerReady() {
    // First-run setup when the wizard has not completed (defaults seed a provider, so
    // empty-providers is not a reliable gate).
    try {
      const resp = await fetch(`${serverAddr}/api/settings`);
      if (resp.ok) {
        const config = await resp.json();
        if (config.setup_needed === true || config.setup_completed === false) {
          setPhase('setup');
          return;
        }
      }
    } catch {
      // Config check failed, proceed to login
    }

    // Try auto-login
    await attemptAutoLogin();
  }

  async function attemptAutoLogin() {
    try {
      const savedCredentials = await loadCredentials();
      if (savedCredentials) {
        const api = new ChatAPI(savedCredentials.serverAddr);
        const connected = await api.testConnection();
        if (connected) {
          try {
            const session = await api.createSession(savedCredentials.username);
            setHubSessionToken(session.token);
          } catch {
            if (savedCredentials.sessionToken) {
              setHubSessionToken(savedCredentials.sessionToken);
            }
          }
          setUsername(savedCredentials.username);
          setChannel(savedCredentials.channel);
          setServerAddr(savedCredentials.serverAddr);
          await syncUserRulesFromHub();
          setPhase('chat');
          return;
        }
      }
    } catch (error) {
      console.error('[App] Auto-login failed:', error);
    }
    setPhase('login');
  }

  function onSetupComplete() {
    attemptAutoLogin();
  }

  const handleConnect = () => setPhase('chat');
  const handleOpenSettings = (tab?: SettingsTab) => {
    setSettingsInitialTab(tab);
    setIsSettingsOpen(true);
  };
  const handleCloseSettings = () => {
    setIsSettingsOpen(false);
    setSettingsInitialTab(undefined);
  };
  const handleLogout = () => setPhase('login');

  if (!isTauriRuntime() && !isMarkdownPreviewFromUrl() && !isPreviewMode) {
    return <DesktopOnlyGate />;
  }

  // Render preview mode if active
  if (isPreviewMode && previewParams) {
    return (
      <MarkdownPreview 
        workspaceId={previewParams.workspaceId}
        filePath={previewParams.filePath}
      />
    );
  }

  if (phase === 'loading') {
    return (
      <LoadingScreen
        onReady={onServerReady}
        onContinueWithoutHub={() => setPhase('login')}
      />
    );
  }

  if (phase === 'setup') {
    return <SetupWizard onComplete={onSetupComplete} serverAddr={serverAddr} />;
  }

  return (
    <div className="w-full h-screen overflow-hidden flex flex-col">
      <UpdateBanner />
      <div className="flex-1 min-h-0 overflow-hidden">
        {phase === 'login' ? (
          <LoginScreen onConnect={handleConnect} />
        ) : (
          <ChatWindow 
            onOpenSettings={handleOpenSettings} 
            onLogout={handleLogout}
          />
        )}
      </div>
      
      <SettingsModal
        isOpen={isSettingsOpen}
        onClose={handleCloseSettings}
        initialTab={settingsInitialTab}
      />
    </div>
  );
}

export default App;
