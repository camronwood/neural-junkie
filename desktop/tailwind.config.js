/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Theme-aware palette (CSS vars in styles.css; data-theme on html)
        slack: {
          bg: 'var(--nj-bg)',
          bgHover: 'var(--nj-bg-hover)',
          sidebar: 'var(--nj-bg-hover)',
          text: 'var(--nj-text)',
          textMuted: 'var(--nj-text-muted)',
          accent: 'var(--nj-accent)',
          accentHover: 'var(--nj-accent-hover)',
          success: 'var(--nj-success)',
          border: 'var(--nj-border)',
        },
        nj: {
          surface: 'var(--nj-surface-elevated)',
          header: 'var(--nj-header-accent)',
        },
        agent: {
          frontend: '#52b6ef',
          backend: '#af77ca',
          devops: '#f09348',
          database: '#fbd837',
          security: '#f16a5a',
          default: '#a9b9ba',
        },
      },
      fontFamily: {
        sans: [
          '-apple-system',
          'BlinkMacSystemFont',
          '"Segoe UI"',
          'Roboto',
          '"Helvetica Neue"',
          'Arial',
          'sans-serif',
        ],
        mono: [
          '"SF Mono"',
          'Monaco',
          '"Cascadia Code"',
          '"Roboto Mono"',
          'Consolas',
          'monospace',
        ],
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0', transform: 'translateY(4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
      animation: {
        fadeIn: 'fadeIn 0.3s ease-out',
      },
    },
  },
  plugins: [],
}
