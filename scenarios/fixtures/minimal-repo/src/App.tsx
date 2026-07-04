import React from 'react';

const App: React.FC = () => {
  const handleHelloWorld = async () => {
    try {
      // Call Go backend through bridge
      if (typeof window !== 'undefined') {
        console.log('Calling Go backend...');
      }
    } catch (err) {
      console.error('Error calling Go backend:', err);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-8">
      <header className="mb-8 text-center">
        <h1 className="text-3xl font-bold text-slate-800 mb-2">Hello World App</h1>
        <p className="text-slate-600">React + Tauri + Go Bridge</p>
      </header>

      <main className="max-w-4xl mx-auto space-y-4">
        <div className="bg-white rounded-xl shadow-lg p-6">
          <button
            onClick={handleHelloWorld}
            className="px-6 py-3 bg-indigo-600 text-white font-semibold rounded-lg hover:bg-indigo-700 transition-colors"
          >
            Say Hello to Go Backend
          </button>
        </div>

        <div className="bg-white rounded-xl shadow-lg p-6">
          <h2 className="text-xl font-semibold text-slate-800 mb-3">Go Bridge Status</h2>
          <p className="text-slate-600">Ready to communicate with core/sample/...</p>
        </div>

        <div className="bg-white rounded-xl shadow-lg p-6">
          <h2 className="text-xl font-semibold text-slate-800 mb-3">Current Version</h2>
          <code className="block bg-slate-100 px-4 py-2 rounded text-sm text-slate-700">v1.0.0</code>
        </div>
      </main>
    </div>
  );
};

export default App;