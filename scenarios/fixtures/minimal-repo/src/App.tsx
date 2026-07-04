import React, { useState, useEffect } from 'react';
import { invoke } from '@tauri-apps/api/core';

type VersionResponse = {
  version: string;
  platform?: string;
  arch?: string;
};

const App: React.FC = () => {
  const [version, setVersion] = useState<string>('Loading...');
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch version info via Tauri IPC instead of HTTP
  const getVersion = async () => {
    setLoading(true);
    setError(null);
    try {
      // Get the Go backend's PrintVersion output through Tauri invoke
      const data: VersionResponse | undefined = await invoke<VersionResponse>('get_version');
      if (data) {
        setVersion(`v${data.version} ${data.platform ? `(${data.platform})` : ''}`);
      } else {
        setVersion('Unknown');
      }
    } catch (err) {
      console.error('Failed to fetch version:', err);
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';
      setError(errorMessage.includes('unreachable') ? 'Tauri bridge not available' : errorMessage);
      setVersion('Offline');
    } finally {
      setLoading(false);
    }
  };

  // Call the Go backend's HelloWorld function through Tauri IPC
  const handleHelloWorld = async () => {
    try {
      await invoke<void>('hello_world');
      console.log('Hello World executed successfully via Go backend');
    } catch (err) {
      console.error('Failed to call hello_world:', err);
      setError(err instanceof Error ? err.message : 'Call failed');
    }
  };

  useEffect(() => {
    // Attempt to get version on mount if Tauri is available
    getVersion();
  }, []);

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <header className="mb-8 flex justify-between items-center">
        <h1 className="text-2xl font-bold text-gray-800">Minimal Repo Sample</h1>
        <span className={`px-4 py-2 rounded-full font-semibold ${
          version === 'Loading...' ? 'bg-yellow-500' :
          version === 'Offline' ? 'bg-red-500' :
          'bg-green-500'
        } text-white text-sm`}>
          {loading ? '⏳ Syncing...' : version}
        </span>
      </header>
      
      <main className="max-w-2xl mx-auto">
        <section className="bg-white rounded-lg shadow p-6 mb-4">
          <h2 className="text-lg font-semibold text-gray-700 mb-3">Interactive Demo</h2>
          <button 
            onClick={handleHelloWorld}
            disabled={loading}
            className={`px-6 py-3 rounded-lg font-medium transition ${
              loading ? 'bg-gray-400 cursor-not-allowed' : 'bg-blue-500 hover:bg-blue-600 text-white'
            }`}
          >
            {loading ? 'Processing...' : 'Say Hello'}
          </button>
          
          {error && (
            <p className="mt-4 text-red-600 text-sm bg-red-50 p-3 rounded">
              ⚠️ {error}
            </p>
          )}
          
          {!loading && !error && (
            <p className="mt-4 text-gray-600">
              This app bridges Go and React. Click the button to call the backend's 
              hello_world function via Tauri IPC.
            </p>
          )}
        </section>

        <section className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold text-gray-700 mb-3">About</h2>
          <p className="text-gray-600">
            This is a minimal repository demonstrating the integration between 
            Go backend services and React frontend with Tailwind CSS using 
            Tauri IPC for cross-process communication.
          </p>
        </section>
      </main>
    </div>
  );
};

export default App;