import { useState, useEffect } from 'react';
import { LoginScreen } from './components/LoginScreen';
import { ChatWindow } from './components/ChatWindow';
import { SettingsModal } from './components/SettingsModal';
import { MarkdownPreview } from './components/MarkdownPreview';
import { useSettingsStore } from './stores/settingsStore';
import { useTerminalStore } from './stores/terminalStore';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { useChatStore } from './stores/chatStore';
import { loadCredentials } from './utils/secureStorage';
import { ChatAPI } from './api/chatAPI';

type AppView = 'login' | 'chat';

function App() {
  const [currentView, setCurrentView] = useState<AppView>('login');
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [isPreviewMode, setIsPreviewMode] = useState(false);
  const [previewParams, setPreviewParams] = useState<{ workspaceId: string; filePath: string } | null>(null);
  const [testMode, setTestMode] = useState(false);
  const { settings, loadSettings } = useSettingsStore();
  const { togglePanel } = useTerminalStore();
  const { setUsername, setChannel, setServerAddr } = useChatStore();

  // Load settings on app start
  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  // Auto-login if credentials are saved
  useEffect(() => {
    const attemptAutoLogin = async () => {
      try {
        const savedCredentials = await loadCredentials();
        if (savedCredentials) {
          console.log('[App] Found saved credentials, attempting auto-login...');
          
          // Test server connection
          const api = new ChatAPI(savedCredentials.serverAddr);
          const connected = await api.testConnection();
          
          if (connected) {
            // Set chat store state
            setUsername(savedCredentials.username);
            setChannel(savedCredentials.channel);
            setServerAddr(savedCredentials.serverAddr);
            
            // Switch to chat view
            setCurrentView('chat');
            console.log('[App] Auto-login successful');
          } else {
            console.log('[App] Server connection failed, showing login screen');
          }
        }
      } catch (error) {
        console.error('[App] Auto-login failed:', error);
      }
    };

    attemptAutoLogin();
  }, [setUsername, setChannel, setServerAddr]);

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
    
    // Apply scope class to body
    document.body.className = `font-scope-${settings.fontSizeScope}`;
  }, [settings.fontSize, settings.fontSizeScope]);

  // Keyboard shortcuts
  useKeyboardShortcuts({
    onOpenSettings: () => setIsSettingsOpen(true),
    onToggleTerminal: togglePanel,
  });

  const handleConnect = () => {
    setCurrentView('chat');
  };

  const handleOpenSettings = () => {
    setIsSettingsOpen(true);
  };

  const handleCloseSettings = () => {
    setIsSettingsOpen(false);
  };

  const handleLogout = () => {
    setCurrentView('login');
  };

  // Render preview mode if active
  if (isPreviewMode && previewParams) {
    return (
      <MarkdownPreview 
        workspaceId={previewParams.workspaceId}
        filePath={previewParams.filePath}
      />
    );
  }

  return (
    <div className="w-full h-screen overflow-hidden">
      {currentView === 'login' ? (
        <LoginScreen onConnect={handleConnect} />
      ) : (
        <ChatWindow 
          onOpenSettings={handleOpenSettings} 
          onLogout={handleLogout}
          testMode={testMode}
          setTestMode={setTestMode}
        />
      )}
      
      {/* Settings Modal */}
      <SettingsModal 
        isOpen={isSettingsOpen} 
        onClose={handleCloseSettings}
        testMode={testMode}
        setTestMode={setTestMode}
      />
    </div>
  );
}

export default App;

