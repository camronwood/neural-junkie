/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_NJ_HUB_URL?: string;
  /** Set to "1" for Playwright Vite e2e (bypasses DesktopOnlyGate). */
  readonly VITE_NJ_E2E?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

