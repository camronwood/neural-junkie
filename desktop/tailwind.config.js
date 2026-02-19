/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Slack-inspired color palette
        slack: {
          bg: '#1a1d21',           // Main background
          bgHover: '#222529',      // Hover background
          sidebar: '#3f0e40',      // Sidebar (optional)
          text: '#d1d2d3',         // Primary text
          textMuted: '#9ca3af',    // Muted text
          accent: '#1164a3',       // Primary blue
          accentHover: '#0e4c7a',  // Primary blue hover
          success: '#148567',      // Green accent
          border: '#545454',       // Subtle borders
        },
        agent: {
          // Agent type colors (matching Fyne colors)
          frontend: '#52b6ef',     // Blue
          backend: '#af77ca',      // Purple
          devops: '#f09348',       // Orange
          database: '#fbd837',     // Yellow
          security: '#f16a5a',     // Red
          default: '#a9b9ba',      // Gray
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

