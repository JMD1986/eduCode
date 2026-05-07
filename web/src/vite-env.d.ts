/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_DEV_USER_SUBJECT?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
