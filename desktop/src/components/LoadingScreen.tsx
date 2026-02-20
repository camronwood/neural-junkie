import { useState, useEffect } from 'react';
import { listen } from '@tauri-apps/api/event';

interface LoadingScreenProps {
  onReady: () => void;
  onError?: (error: string) => void;
}

export function LoadingScreen({ onReady, onError }: LoadingScreenProps) {
  const [status, setStatus] = useState('Starting server...');
  const [hasError, setHasError] = useState(false);

  useEffect(() => {
    const unlisten1 = listen<boolean>('server-ready', () => {
      setStatus('Server ready!');
      setTimeout(onReady, 300);
    });

    const unlisten2 = listen<string>('server-error', (event) => {
      setStatus(event.payload);
      setHasError(true);
      onError?.(event.payload);
    });

    // Fallback: poll the health endpoint directly
    const pollInterval = setInterval(async () => {
      try {
        const resp = await fetch('http://localhost:8080/api/health');
        if (resp.ok) {
          clearInterval(pollInterval);
          setStatus('Server ready!');
          setTimeout(onReady, 300);
        }
      } catch {
        // Server not ready yet
      }
    }, 1000);

    return () => {
      unlisten1.then(fn => fn());
      unlisten2.then(fn => fn());
      clearInterval(pollInterval);
    };
  }, [onReady, onError]);

  return (
    <div className="flex items-center justify-center w-full h-screen bg-gray-950">
      <div className="text-center space-y-6">
        <div className="text-4xl font-bold text-white tracking-tight">
          Neural Junkie
        </div>
        <div className="text-sm text-gray-400">{status}</div>
        {!hasError && (
          <div className="flex justify-center">
            <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
          </div>
        )}
        {hasError && (
          <div className="text-xs text-red-400 max-w-sm mx-auto">
            The backend server could not be reached. If running in development mode,
            start it with <code className="bg-gray-800 px-1 rounded">make server</code>.
          </div>
        )}
      </div>
    </div>
  );
}
