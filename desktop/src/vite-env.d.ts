/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_NJ_HUB_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

