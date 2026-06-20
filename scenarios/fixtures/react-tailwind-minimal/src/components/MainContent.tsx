import { FC } from "react";

export default function MainContent() {
  return (
    <main className="flex-1 overflow-y-auto bg-slate-950 dark:bg-slate-50 rounded-lg p-6">
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-white dark:text-slate-900 mb-2">
          Welcome to Your Dashboard
        </h1>
        <p className="text-slate-400 dark:text-slate-600 max-w-xl">
          This is a minimal React application built with Tailwind CSS. 
          Start building your UI components here.
        </p>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {/* Welcome Card */}
        <div className="bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl p-6 text-white shadow-lg transform hover:-translate-y-1 transition-all duration-300">
          <div className="mb-4">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-10 w-10 text-white/80"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
              />
            </svg>
          </div>
          <h3 className="text-xl font-semibold mb-2">Get Started</h3>
          <p className="text-white/80 text-sm leading-relaxed">
            Replace this content with your dashboard widgets, charts, or other components. 
            Tailwind makes styling a breeze.
          </p>
        </div>

        {/* Features Card */}
        <div className="bg-slate-800 dark:bg-slate-100 rounded-xl p-6 text-white dark:text-slate-900 shadow-lg transform hover:-translate-y-1 transition-all duration-300">
          <div className="mb-4">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-10 w-10 text-indigo-500 dark:text-indigo-600"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19.428 15.428a2 2 0 00-1.022-.547l-2.384-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"
              />
            </svg>
          </div>
          <h3 className="text-xl font-semibold mb-2">Features</h3>
          <ul className="space-y-2 text-sm leading-relaxed">
            <li className="flex items-start gap-2">
              <span className="text-green-500">✓</span>
              <span>React 18 with TypeScript</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-green-500">✓</span>
              <span>Tailwind CSS for styling</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-green-500">✓</span>
              <span>Vite for development</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="text-green-500">✓</span>
              <span>Dark/Light mode support</span>
            </li>
          </ul>
        </div>

        {/* Actions Card */}
        <div className="bg-slate-800 dark:bg-slate-100 rounded-xl p-6 text-white dark:text-slate-900 shadow-lg transform hover:-translate-y-1 transition-all duration-300">
          <div className="mb-4">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-10 w-10 text-pink-500 dark:text-pink-600"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
              />
            </svg>
          </div>
          <h3 className="text-xl font-semibold mb-2">Quick Actions</h3>
          <p className="text-sm leading-relaxed mb-4">
            Build your UI components, add functionality, and create a beautiful 
            user interface for your application.
          </p>
          
          <div className="flex flex-wrap gap-2">
            <button className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-medium transition-all duration-200 shadow-sm">
              Create Component
            </button>
            <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 dark:hover:bg-slate-200 text-slate-300 dark:text-slate-900 rounded-lg font-medium transition-all duration-200">
              View Documentation
            </button>
          </div>
        </div>
      </div>

      {/* Additional Info */}
      <div className="mt-8 p-4 bg-slate-800 dark:bg-slate-100 rounded-lg text-center text-sm text-slate-500 dark:text-slate-600">
        Built with React, Tailwind CSS and Vite
      </div>
    </main>
  );
}