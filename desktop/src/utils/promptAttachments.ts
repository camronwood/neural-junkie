import type { PromptAttachmentPayload } from '../constants/promptMetadata';
import type { WorkspaceFileDragPayload } from './workspaceFileDrag';
import { isImagePreviewPath } from './editorFileKind';

export const MAX_ATTACH_BYTES = 80_000;
export const MAX_ATTACH_COUNT = 12;
export const MAX_ATTACH_TOTAL = 350_000;

const VALID_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/jpg', 'image/webp', 'image/gif'];

const BINARY_EXT = new Set([
  'png', 'jpg', 'jpeg', 'gif', 'webp', 'ico', 'svg', 'bmp', 'zip', 'tar', 'gz', 'pdf', 'mp4', 'mp3', 'wav',
  'exe', 'dll', 'so', 'dylib', 'woff', 'woff2', 'ttf', 'eot', 'gguf', 'bin',
]);

const LANG_BY_EXT: Record<string, string> = {
  go: 'go',
  rs: 'rust',
  py: 'python',
  ts: 'typescript',
  tsx: 'tsx',
  js: 'javascript',
  jsx: 'jsx',
  md: 'markdown',
  json: 'json',
  yaml: 'yaml',
  yml: 'yaml',
  toml: 'toml',
  sql: 'sql',
  sh: 'bash',
  tf: 'hcl',
  hcl: 'hcl',
  html: 'html',
  css: 'css',
  scss: 'scss',
  vue: 'vue',
  rb: 'ruby',
  java: 'java',
  kt: 'kotlin',
  swift: 'swift',
  c: 'c',
  cpp: 'cpp',
  h: 'c',
  cs: 'csharp',
};

export function isTauriRuntime(): boolean {
  return typeof window !== 'undefined' && '__TAURI__' in window;
}

export function isImageMime(mime: string): boolean {
  return VALID_IMAGE_TYPES.includes(mime);
}

export function isImageFile(file: File): boolean {
  return isImageMime(file.type);
}

export function inferLanguageFromPath(path: string): string {
  const base = path.split(/[/\\]/).pop() ?? path;
  const ext = base.includes('.') ? base.slice(base.lastIndexOf('.') + 1).toLowerCase() : '';
  return LANG_BY_EXT[ext] ?? 'text';
}

export function isBinaryPath(path: string): boolean {
  const base = path.split(/[/\\]/).pop() ?? path;
  const ext = base.includes('.') ? base.slice(base.lastIndexOf('.') + 1).toLowerCase() : '';
  return BINARY_EXT.has(ext);
}

function truncateContent(text: string): string {
  if (text.length <= MAX_ATTACH_BYTES) {
    return text;
  }
  return text.slice(0, MAX_ATTACH_BYTES) + '\n[truncated client-side]';
}

async function readFileAsText(file: File): Promise<string> {
  if (typeof file.text === 'function') {
    return file.text();
  }
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ''));
    reader.onerror = () => reject(reader.error ?? new Error('read failed'));
    reader.readAsText(file);
  });
}

/** Merge new attachments into existing, respecting count and total size caps. */
export function mergePromptAttachments(
  existing: PromptAttachmentPayload[],
  added: PromptAttachmentPayload[]
): PromptAttachmentPayload[] {
  const next = [...existing];
  let total = next.reduce((s, x) => s + x.content.length, 0);
  for (const item of added) {
    if (next.length >= MAX_ATTACH_COUNT) break;
    if (next.some((x) => x.path === item.path)) continue;
    if (total + item.content.length > MAX_ATTACH_TOTAL) break;
    next.push(item);
    total += item.content.length;
  }
  return next;
}

/** Read text attachments from browser File objects (Vite dev / browser). */
export async function attachmentsFromFileList(
  files: FileList | File[],
  existing: PromptAttachmentPayload[]
): Promise<PromptAttachmentPayload[]> {
  const list = Array.from(files);
  const added: PromptAttachmentPayload[] = [];
  for (const file of list) {
    if (isImageFile(file)) continue;
    if (isBinaryPath(file.name)) continue;
    try {
      const text = await readFileAsText(file);
      added.push({
        path: file.name,
        language: inferLanguageFromPath(file.name),
        content: truncateContent(text),
      });
    } catch {
      /* skip unreadable */
    }
  }
  return mergePromptAttachments(existing, added);
}

/** Load text file content from workspace refs (file explorer → chat drag). */
export async function attachmentsFromWorkspaceRefs(
  refs: WorkspaceFileDragPayload[],
  existing: PromptAttachmentPayload[]
): Promise<PromptAttachmentPayload[]> {
  if (!refs.length) return existing;
  const { ChatAPI } = await import('../api/chatAPI');
  const { getHubBaseURL } = await import('../config/hubUrl');
  const api = new ChatAPI(getHubBaseURL());
  const added: PromptAttachmentPayload[] = [];
  for (const ref of refs) {
    if (isImagePreviewPath(ref.path) || isBinaryPath(ref.path)) continue;
    try {
      const content = await api.fetchFileContent(ref.workspaceId, ref.path);
      added.push({
        path: ref.path,
        language: inferLanguageFromPath(ref.path),
        content: truncateContent(content),
      });
    } catch (e) {
      console.error('[attachmentsFromWorkspaceRefs]', ref.path, e);
    }
  }
  return mergePromptAttachments(existing, added);
}

/** Read attachments from absolute paths via Tauri (Finder / desktop drops). */
export async function attachmentsFromAbsolutePaths(
  paths: string[],
  existing: PromptAttachmentPayload[]
): Promise<PromptAttachmentPayload[]> {
  if (!paths.length || !isTauriRuntime()) {
    return existing;
  }
  try {
    const { invoke } = await import('@tauri-apps/api/tauri');
    const read = await invoke<PromptAttachmentPayload[]>('read_prompt_attachment_paths', { paths });
    return mergePromptAttachments(existing, read);
  } catch (e) {
    console.error('[attachmentsFromAbsolutePaths]', e);
    return existing;
  }
}
