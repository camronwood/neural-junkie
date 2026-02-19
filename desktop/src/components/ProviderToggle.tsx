import { useState } from 'react';
import { useChatStore } from '../stores/chatStore';

interface ProviderToggleProps {
  className?: string;
}

export function ProviderToggle({ className = '' }: ProviderToggleProps) {
  const { switchAllAgentProviders } = useChatStore();
  const [isSwitching, setIsSwitching] = useState(false);
  const [currentProvider, setCurrentProvider] = useState<'claude' | 'ollama'>('claude');

  const handleProviderSwitch = async (provider: 'claude' | 'ollama', model: string) => {
    setIsSwitching(true);
    try {
      await switchAllAgentProviders(provider, model);
      setCurrentProvider(provider);
    } catch (error) {
      console.error('Failed to switch providers:', error);
      // You could add a toast notification here
    } finally {
      setIsSwitching(false);
    }
  };

  const getProviderIcon = (provider: 'claude' | 'ollama') => {
    switch (provider) {
      case 'ollama':
        return '🤖';
      case 'claude':
        return '🧠';
    }
  };


  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <span className="text-sm text-slack-textMuted">AI Provider:</span>
      
      <div className="flex gap-1">
        <button
          onClick={() => handleProviderSwitch('claude', 'claude-sonnet')}
          disabled={isSwitching}
          className={`px-3 py-1 text-xs text-white rounded transition-colors ${
            currentProvider === 'claude' 
              ? 'bg-purple-600' 
              : 'bg-purple-500 hover:bg-purple-600'
          } ${isSwitching ? 'opacity-50 cursor-not-allowed' : ''}`}
          title="Switch all agents to Claude"
        >
          {getProviderIcon('claude')} Claude
        </button>
        
        <button
          onClick={() => handleProviderSwitch('ollama', 'llama3.1')}
          disabled={isSwitching}
          className={`px-3 py-1 text-xs text-white rounded transition-colors ${
            currentProvider === 'ollama' 
              ? 'bg-blue-600' 
              : 'bg-blue-500 hover:bg-blue-600'
          } ${isSwitching ? 'opacity-50 cursor-not-allowed' : ''}`}
          title="Switch all agents to Ollama"
        >
          {getProviderIcon('ollama')} Ollama
        </button>
      </div>
      
      {isSwitching && (
        <div className="flex items-center gap-1 text-xs text-slack-textMuted">
          <div className="w-3 h-3 border-2 border-slack-textMuted border-t-transparent rounded-full animate-spin" />
          Switching...
        </div>
      )}
    </div>
  );
}
