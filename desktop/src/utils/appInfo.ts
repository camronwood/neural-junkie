// App metadata and version information
import pkg from '../../package.json';

export const APP_INFO = {
  name: 'Neural Junkie',
  version: pkg.version,
  description: 'Multi-agent AI collaboration system',
  author: 'Camron Wood',
  license: 'MIT',
  repository: 'https://github.com/camronwood/neural-junkie',
  documentation: 'https://github.com/camronwood/neural-junkie/tree/main/docs',
} as const;

export const TECH_STACK = [
  'React 18',
  'TypeScript',
  'Tauri',
  'Go 1.21+',
  'Zustand',
  'Tailwind CSS',
  'WebSocket',
] as const;

// Prefer package.json semver for UI labels (bundle may embed WiX-safe 1.0.0.N).
export async function getAppVersion(): Promise<string> {
  return APP_INFO.version;
}
