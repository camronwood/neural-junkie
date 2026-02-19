// App metadata and version information
export const APP_INFO = {
  name: 'Neural Junkie',
  version: '1.0.0',
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

// Get app version from Tauri (with fallback for dev mode)
export async function getAppVersion(): Promise<string> {
  try {
    // In Tauri environment, we can get version from the app
    if (typeof window !== 'undefined' && (window as any).__TAURI__) {
      const { getVersion } = await import('@tauri-apps/api/app');
      return await getVersion();
    }
  } catch (error) {
    console.warn('Failed to get Tauri version:', error);
  }
  
  // Fallback to static version
  return APP_INFO.version;
}
