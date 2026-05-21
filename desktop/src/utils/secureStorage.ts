import { Store } from '@tauri-apps/plugin-store';
import { invoke } from '@tauri-apps/api/tauri';
import { normalizeHubBaseURL } from '../config/hubUrl';

export interface SavedCredentials {
  username: string;
  channel: string;
  serverAddr: string;
  savedAt: string; // ISO timestamp
  sessionToken?: string;
}

const STORE_FILENAME = 'credentials.dat';
const CREDENTIALS_KEY = 'user_credentials';
const ENCRYPTED_KEY = 'user_credentials_enc';

function isTauriShell(): boolean {
  return (
    typeof window !== 'undefined' &&
    Object.prototype.hasOwnProperty.call(window, '__TAURI__')
  );
}

// Initialize the store
let store: Store | null = null;

async function getStore(): Promise<Store> {
  if (!store) {
    store = new Store(STORE_FILENAME);
  }
  return store;
}

async function encryptPayload(json: string): Promise<string> {
  if (!isTauriShell()) {
    return json;
  }
  return invoke<string>('encrypt_credential_blob', { plaintext: json });
}

async function decryptPayload(blob: string): Promise<string> {
  if (!isTauriShell() || !blob || blob.startsWith('{')) {
    return blob;
  }
  return invoke<string>('decrypt_credential_blob', { blob });
}

/**
 * Save user credentials to secure storage
 */
export async function saveCredentials(
  username: string,
  channel: string,
  serverAddr: string,
  rememberMe: boolean,
  sessionToken?: string
): Promise<void> {
  try {
    if (!rememberMe) {
      await clearCredentials();
      return;
    }

    const credentials: SavedCredentials = {
      username,
      channel,
      serverAddr,
      savedAt: new Date().toISOString(),
      sessionToken,
    };

    const storeInstance = await getStore();
    const json = JSON.stringify(credentials);
    if (isTauriShell()) {
      const enc = await encryptPayload(json);
      await storeInstance.set(ENCRYPTED_KEY, enc);
      await storeInstance.delete(CREDENTIALS_KEY);
    } else {
      await storeInstance.set(CREDENTIALS_KEY, credentials);
      await storeInstance.delete(ENCRYPTED_KEY);
    }
    await storeInstance.save();
  } catch (error) {
    console.error('[SecureStorage] Failed to save credentials:', error);
    throw error;
  }
}

/**
 * Load saved credentials from secure storage
 */
export async function loadCredentials(): Promise<SavedCredentials | null> {
  try {
    const storeInstance = await getStore();
    let credentials: SavedCredentials | null = null;

    const enc = await storeInstance.get<string>(ENCRYPTED_KEY);
    if (typeof enc === 'string' && enc.length > 0) {
      const json = await decryptPayload(enc);
      credentials = JSON.parse(json) as SavedCredentials;
    } else {
      credentials = (await storeInstance.get<SavedCredentials>(CREDENTIALS_KEY)) ?? null;
    }

    if (credentials) {
      const normalized = normalizeHubBaseURL(credentials.serverAddr);
      if (normalized !== credentials.serverAddr.trim()) {
        const updated: SavedCredentials = {
          ...credentials,
          serverAddr: normalized,
        };
        await saveCredentials(
          updated.username,
          updated.channel,
          updated.serverAddr,
          true,
          updated.sessionToken
        );
        return updated;
      }
      return credentials;
    }

    return null;
  } catch (error) {
    console.error('[SecureStorage] Failed to load credentials:', error);
    return null;
  }
}

/**
 * Clear saved credentials from secure storage
 */
export async function clearCredentials(): Promise<void> {
  try {
    const storeInstance = await getStore();
    await storeInstance.delete(CREDENTIALS_KEY);
    await storeInstance.delete(ENCRYPTED_KEY);
    await storeInstance.save();
  } catch (error) {
    console.error('[SecureStorage] Failed to clear credentials:', error);
    throw error;
  }
}
