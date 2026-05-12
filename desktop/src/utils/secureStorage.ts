import { Store } from '@tauri-apps/plugin-store';
import { normalizeLegacyHubServerAddr } from '../config/hubUrl';

export interface SavedCredentials {
  username: string;
  channel: string;
  serverAddr: string;
  savedAt: string; // ISO timestamp
}

const STORE_FILENAME = 'credentials.dat';
const CREDENTIALS_KEY = 'user_credentials';

// Initialize the store
let store: Store | null = null;

async function getStore(): Promise<Store> {
  if (!store) {
    store = new Store(STORE_FILENAME);
  }
  return store;
}

/**
 * Save user credentials to secure storage
 */
export async function saveCredentials(
  username: string,
  channel: string,
  serverAddr: string,
  rememberMe: boolean
): Promise<void> {
  try {
    if (!rememberMe) {
      // If "Remember Me" is unchecked, clear any existing credentials
      await clearCredentials();
      return;
    }

    const credentials: SavedCredentials = {
      username,
      channel,
      serverAddr,
      savedAt: new Date().toISOString(),
    };

    const storeInstance = await getStore();
    await storeInstance.set(CREDENTIALS_KEY, credentials);
    await storeInstance.save();
    
    console.log('[SecureStorage] Credentials saved successfully');
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
    const credentials = await storeInstance.get<SavedCredentials>(CREDENTIALS_KEY);
    
    if (credentials) {
      const normalized = normalizeLegacyHubServerAddr(credentials.serverAddr);
      if (normalized !== credentials.serverAddr.trim()) {
        const updated: SavedCredentials = {
          ...credentials,
          serverAddr: normalized,
        };
        await storeInstance.set(CREDENTIALS_KEY, updated);
        await storeInstance.save();
        console.log('[SecureStorage] Migrated legacy hub port 8080 → 18765 in saved credentials');
        return updated;
      }
      console.log('[SecureStorage] Credentials loaded successfully');
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
    await storeInstance.save();
    
    console.log('[SecureStorage] Credentials cleared successfully');
  } catch (error) {
    console.error('[SecureStorage] Failed to clear credentials:', error);
    throw error;
  }
}

