import { useState, useEffect } from 'react';
import { useChatStore } from '../stores/chatStore';
import { ChatAPI } from '../api/chatAPI';
import { saveCredentials, loadCredentials } from '../utils/secureStorage';

interface LoginScreenProps {
  onConnect: () => void;
}

export function LoginScreen({ onConnect }: LoginScreenProps) {
  const { setUsername, setChannel, setServerAddr, serverAddr, channel } = useChatStore();
  
  const [nameInput, setNameInput] = useState('');
  const [channelInput, setChannelInput] = useState(channel);
  const [serverInput, setServerInput] = useState(serverAddr);
  const [rememberMe, setRememberMe] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoadingCredentials, setIsLoadingCredentials] = useState(true);

  // Load saved credentials on mount
  useEffect(() => {
    const loadSavedCredentials = async () => {
      try {
        const saved = await loadCredentials();
        if (saved) {
          setNameInput(saved.username);
          setChannelInput(saved.channel);
          setServerInput(saved.serverAddr);
          setRememberMe(true);
          console.log('[LoginScreen] Loaded saved credentials');
        }
      } catch (err) {
        console.error('[LoginScreen] Failed to load saved credentials:', err);
      } finally {
        setIsLoadingCredentials(false);
      }
    };

    loadSavedCredentials();
  }, []);

  const handleConnect = async () => {
    // Validate inputs
    const name = nameInput.trim() || 'Anonymous';
    const chan = channelInput.trim() || 'general';
    const server = serverInput.trim() || 'localhost:8080';

    setError(null);
    setIsConnecting(true);

    try {
      // Test server connection
      const api = new ChatAPI(server);
      const connected = await api.testConnection();

      if (!connected) {
        throw new Error('Unable to connect to server');
      }

      // Update store
      setUsername(name);
      setChannel(chan);
      setServerAddr(server);

      // Save credentials if Remember Me is checked
      try {
        await saveCredentials(name, chan, server, rememberMe);
      } catch (err) {
        console.error('[LoginScreen] Failed to save credentials:', err);
        // Non-fatal error, continue with connection
      }

      // Notify parent
      onConnect();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Connection failed');
      setIsConnecting(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !isConnecting) {
      handleConnect();
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-slack-bg p-8">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-slack-text mb-2">
            Welcome to Neural Junkie
          </h1>
          <p className="text-slack-textMuted">
            Connect with AI agents to get help with your projects
          </p>
        </div>

        {/* Form */}
        <div className="bg-slack-bgHover rounded-lg p-6 shadow-xl border border-slack-border">
          <div className="space-y-4">
            {/* Name Input */}
            <div>
              <label htmlFor="name" className="block text-sm font-medium text-slack-text mb-2">
                Your Name
              </label>
              <input
                id="name"
                type="text"
                value={nameInput}
                onChange={(e) => setNameInput(e.target.value)}
                onKeyPress={handleKeyPress}
                placeholder="Enter your name"
                disabled={isConnecting}
                className="w-full px-4 py-2 bg-slack-bg text-slack-text placeholder-slack-textMuted rounded border border-slack-border focus:outline-none focus:ring-2 focus:ring-slack-accent disabled:opacity-50"
              />
            </div>

            {/* Channel Input */}
            <div>
              <label htmlFor="channel" className="block text-sm font-medium text-slack-text mb-2">
                Channel
              </label>
              <input
                id="channel"
                type="text"
                value={channelInput}
                onChange={(e) => setChannelInput(e.target.value)}
                onKeyPress={handleKeyPress}
                placeholder="general"
                disabled={isConnecting}
                className="w-full px-4 py-2 bg-slack-bg text-slack-text placeholder-slack-textMuted rounded border border-slack-border focus:outline-none focus:ring-2 focus:ring-slack-accent disabled:opacity-50"
              />
            </div>

            {/* Server Input */}
            <div>
              <label htmlFor="server" className="block text-sm font-medium text-slack-text mb-2">
                Server Address
              </label>
              <input
                id="server"
                type="text"
                value={serverInput}
                onChange={(e) => setServerInput(e.target.value)}
                onKeyPress={handleKeyPress}
                placeholder="localhost:8080"
                disabled={isConnecting}
                className="w-full px-4 py-2 bg-slack-bg text-slack-text placeholder-slack-textMuted rounded border border-slack-border focus:outline-none focus:ring-2 focus:ring-slack-accent disabled:opacity-50"
              />
            </div>

            {/* Remember Me Checkbox */}
            <div className="flex items-center">
              <input
                id="rememberMe"
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                disabled={isConnecting}
                className="w-4 h-4 bg-slack-bg border-slack-border rounded text-slack-accent focus:ring-2 focus:ring-slack-accent disabled:opacity-50"
              />
              <label htmlFor="rememberMe" className="ml-2 text-sm text-slack-text cursor-pointer">
                Remember me
              </label>
            </div>

            {/* Error Message */}
            {error && (
              <div className="p-3 bg-red-500/10 border border-red-500/50 rounded text-red-500 text-sm">
                {error}
              </div>
            )}

            {/* Connect Button */}
            <button
              onClick={handleConnect}
              disabled={isConnecting || isLoadingCredentials}
              className="w-full px-4 py-3 bg-slack-accent hover:bg-slack-accentHover text-white font-medium rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoadingCredentials ? 'Loading...' : isConnecting ? 'Connecting...' : 'Connect'}
            </button>
          </div>
        </div>

        {/* Footer */}
        <div className="text-center mt-6 text-sm text-slack-textMuted">
          <p>Press Enter to connect • Agents will respond based on their expertise</p>
        </div>
      </div>
    </div>
  );
}

