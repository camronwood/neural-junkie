import { useState, useEffect } from 'react';
import { LoginScreen } from './components/LoginScreen';
import { ChatWindow } from './components/ChatWindow';
import { SettingsModal } from './components/SettingsModal';
import { MarkdownPreview } from './components/MarkdownPreview';
import { LoadingScreen } from './components/LoadingScreen';
import { SetupWizard } from './components/SetupWizard';
import { UpdateBanner } from './components/UpdateBanner';
import { useSettingsStore } from './stores/settingsStore';
import { useTerminalStore } from './stores/terminalStore';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { useChatStore } from './stores/chatStore';
import { loadCredentials } from './utils/secureStorage';
import { ChatAPI } from './api/chatAPI';

type AppPhase = 'loading' | 'setup' | 'login' | 'chat';

function App() {
  const [phase, setPhase] = useState<AppPhase>('loading');
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isPreviewMode, setIsPreviewMode] = useState(false);
  const [previewParams, setPreviewParams] = useState<{ workspaceId: string; filePath: string } | null>(null);
  const { settings, loadSettings } = useSettingsStore();
  const { togglePanel } = useTerminalStore();
  const { setUsername, setChannel, setServerAddr } = useChatStore();

  const serverAddr = 'http://localhost:8080';

  // Load settings on app start
  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

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

  // Apply font size to document root
  useEffect(() => {
    const root = document.documentElement;
    root.style.setProperty('--app-font-size', `${settings.fontSize}px`);
    document.body.className = `font-scope-${settings.fontSizeScope}`;
  }, [settings.fontSize, settings.fontSizeScope]);

  // Keyboard shortcuts
  useKeyboardShortcuts({
    onOpenSettings: () => setIsSettingsOpen(true),
    onToggleTerminal: togglePanel,
  });

  async function onServerReady() {
    // Check if first-run setup is needed (no config.json yet)
    try {
      const resp = await fetch(`${serverAddr}/api/settings`);
      if (resp.ok) {
        const config = await resp.json();
        // If no providers are configured, show setup wizard
        if (!config.ai?.providers || config.ai.providers.length === 0) {
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
          setUsername(savedCredentials.username);
          setChannel(savedCredentials.channel);
          setServerAddr(savedCredentials.serverAddr);
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
  const handleOpenSettings = () => setIsSettingsOpen(true);
  const handleCloseSettings = () => setIsSettingsOpen(false);
  const handleLogout = () => setPhase('login');

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
    return <LoadingScreen onReady={onServerReady} />;
  }

  if (phase === 'setup') {
    return <SetupWizard onComplete={onSetupComplete} serverAddr={serverAddr} />;
  }

  return (
    <div className="w-full h-screen overflow-hidden flex flex-col">
      <UpdateBanner />
      <div className="flex-1 overflow-hidden">
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
      />
    </div>
  );
}

export default App;
