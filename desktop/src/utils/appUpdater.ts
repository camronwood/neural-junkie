import { isTauri, invoke } from '@tauri-apps/api/core';
import type { Update } from '@tauri-apps/plugin-updater';

export const UPDATE_CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000;
export const UPDATE_CHECK_TIMEOUT_MS = 10_000;
export const UPDATE_DOWNLOAD_TIMEOUT_MS = 30 * 60 * 1000;
export const BANNER_AUTO_DISMISS_MS = 15_000;

const INSTALLATION_ID_KEY = 'nj-updater-installation-id-v1';

export type UpdateSeverity = 'normal' | 'critical';
export type UpdateEnforcement = 'optional' | 'mandatory';

export interface AppUpdatePolicy {
  schemaVersion: number;
  channel: 'beta' | 'stable';
  severity: UpdateSeverity;
  enforcement: UpdateEnforcement;
  mandatoryAfter?: string;
  minimumSupportedVersion?: string;
  rolloutPercentage: number;
  rolloutSeed: string;
}

export interface AppUpdateInfo {
  available: boolean;
  version: string;
  notes?: string;
  policy: AppUpdatePolicy;
  mandatory: boolean;
}

export type UpdateCheckResult =
  | { status: 'available'; update: AppUpdateInfo }
  | { status: 'current' }
  | { status: 'deferred' }
  | { status: 'unavailable'; reason: string };

let pendingUpdate: Update | null = null;
let pendingInfo: AppUpdateInfo | null = null;
let pendingUpdateDownloaded = false;

async function closePendingUpdate(): Promise<void> {
  const update = pendingUpdate;
  pendingUpdate = null;
  pendingInfo = null;
  pendingUpdateDownloaded = false;
  if (update) {
    await update.close().catch(() => undefined);
  }
}

function defaultChannel(version: string): 'beta' | 'stable' {
  const configured = import.meta.env.VITE_UPDATE_CHANNEL;
  if (configured === 'beta' || configured === 'stable') return configured;
  return version.includes('-beta') ? 'beta' : 'stable';
}

function parsePolicy(raw: Record<string, unknown>, version: string): AppUpdatePolicy {
  const value = raw.policy;
  const policy = value && typeof value === 'object' ? value as Record<string, unknown> : {};
  const rollout = policy.rollout && typeof policy.rollout === 'object'
    ? policy.rollout as Record<string, unknown>
    : {};
  const percentage = typeof rollout.percentage === 'number' ? rollout.percentage : 100;
  return {
    schemaVersion: typeof policy.schema_version === 'number' ? policy.schema_version : 1,
    channel: policy.channel === 'beta' || policy.channel === 'stable'
      ? policy.channel
      : defaultChannel(version),
    severity: policy.severity === 'critical' ? 'critical' : 'normal',
    enforcement: policy.enforcement === 'mandatory' ? 'mandatory' : 'optional',
    mandatoryAfter: typeof policy.mandatory_after === 'string' ? policy.mandatory_after : undefined,
    minimumSupportedVersion:
      typeof policy.minimum_supported_version === 'string'
        ? policy.minimum_supported_version
        : undefined,
    rolloutPercentage: Math.max(0, Math.min(100, percentage)),
    rolloutSeed: typeof rollout.seed === 'string' ? rollout.seed : version,
  };
}

function installationId(): string {
  const existing = localStorage.getItem(INSTALLATION_ID_KEY);
  if (existing) return existing;
  const value = crypto.randomUUID();
  localStorage.setItem(INSTALLATION_ID_KEY, value);
  return value;
}

/** Stable FNV-1a bucket, exported for deterministic rollout tests. */
export function rolloutBucket(seed: string, id: string): number {
  let hash = 0x811c9dc5;
  for (const char of `${seed}:${id}`) {
    hash ^= char.charCodeAt(0);
    hash = Math.imul(hash, 0x01000193);
  }
  return (hash >>> 0) % 10_000;
}

export function isRolloutEligible(policy: AppUpdatePolicy, id = installationId()): boolean {
  return rolloutBucket(policy.rolloutSeed, id) < policy.rolloutPercentage * 100;
}

interface ParsedVersion {
  core: number[];
  prerelease: Array<number | string> | null;
}

function parseVersion(version: string): ParsedVersion {
  const normalized = version.replace(/^v/, '').split('+', 1)[0];
  const separator = normalized.indexOf('-');
  const coreText = separator >= 0 ? normalized.slice(0, separator) : normalized;
  const prereleaseText = separator >= 0 ? normalized.slice(separator + 1) : '';
  const core = coreText.split('.').map((part) => Number.parseInt(part, 10) || 0);
  let prerelease = prereleaseText
    ? prereleaseText.split('.').map((part) => (/^\d+$/.test(part) ? Number(part) : part.toLowerCase()))
    : null;
  // Windows bundle versions map 1.2.0-beta.7 to 1.2.0-7.
  if (prerelease?.length === 1 && typeof prerelease[0] === 'number') {
    prerelease = ['beta', prerelease[0]];
  }
  return { core, prerelease };
}

export function isVersionBelow(current: string, minimum: string): boolean {
  const left = parseVersion(current);
  const right = parseVersion(minimum);
  const length = Math.max(left.core.length, right.core.length);
  for (let index = 0; index < length; index += 1) {
    const a = left.core[index] ?? 0;
    const b = right.core[index] ?? 0;
    if (a === b) continue;
    return a < b;
  }
  if (left.prerelease === null || right.prerelease === null) {
    return left.prerelease !== null && right.prerelease === null;
  }
  const prereleaseLength = Math.max(left.prerelease.length, right.prerelease.length);
  for (let index = 0; index < prereleaseLength; index += 1) {
    const a = left.prerelease[index];
    const b = right.prerelease[index];
    if (a === undefined || b === undefined) return a === undefined;
    if (a === b) continue;
    if (typeof a === 'number' && typeof b === 'number') return a < b;
    if (typeof a === 'number') return true;
    if (typeof b === 'number') return false;
    return a.localeCompare(b) < 0;
  }
  return false;
}

export function isPolicyMandatory(
  policy: AppUpdatePolicy,
  now = Date.now(),
  currentVersion?: string
): boolean {
  if (
    currentVersion &&
    policy.minimumSupportedVersion &&
    isVersionBelow(currentVersion, policy.minimumSupportedVersion)
  ) {
    return true;
  }
  if (policy.enforcement !== 'mandatory') return false;
  if (!policy.mandatoryAfter) return true;
  const deadline = Date.parse(policy.mandatoryAfter);
  return Number.isFinite(deadline) && now >= deadline;
}

export function getUpdateChannelLabel(version: string, policy?: AppUpdatePolicy): string {
  return (policy?.channel ?? defaultChannel(version)) === 'beta' ? 'Beta updates' : 'Stable updates';
}

export const SILENT_UPDATE_UNAVAILABLE_REASONS = new Set([
  'Updates are only available in the desktop app.',
  'Update checks are disabled in dev builds.',
  'Automatic updates are not yet enabled for Linux packages.',
]);

async function isSupportedPlatform(): Promise<boolean> {
  const { type } = await import('@tauri-apps/plugin-os');
  return type() === 'macos' || type() === 'windows';
}

export async function checkForAppUpdate(): Promise<UpdateCheckResult> {
  if (import.meta.env.DEV && import.meta.env.MODE !== 'test') {
    return { status: 'unavailable', reason: 'Update checks are disabled in dev builds.' };
  }
  if (typeof window === 'undefined' || !isTauri()) {
    return { status: 'unavailable', reason: 'Updates are only available in the desktop app.' };
  }
  if (!(await isSupportedPlatform())) {
    return {
      status: 'unavailable',
      reason: 'Automatic updates are not yet enabled for Linux packages.',
    };
  }

  try {
    const { check } = await import('@tauri-apps/plugin-updater');
    const update = await check({ timeout: UPDATE_CHECK_TIMEOUT_MS });
    if (!update) {
      await closePendingUpdate();
      return { status: 'current' };
    }
    const policy = parsePolicy(update.rawJson, update.version);
    const mandatory = isPolicyMandatory(policy, Date.now(), update.currentVersion);
    if (!mandatory && !isRolloutEligible(policy)) {
      await update.close();
      await closePendingUpdate();
      return { status: 'deferred' };
    }
    await closePendingUpdate();
    pendingUpdate = update;
    pendingUpdateDownloaded = false;
    pendingInfo = {
      available: true,
      version: update.version,
      notes: update.body,
      policy,
      mandatory,
    };
    return { status: 'available', update: pendingInfo };
  } catch (error) {
    return {
      status: 'unavailable',
      reason: error instanceof Error ? error.message : 'Update check failed',
    };
  }
}

export async function downloadAppUpdate(onProgress?: (percent: number | null) => void): Promise<void> {
  if (!pendingUpdate) throw new Error('No checked update is available to download');
  let downloaded = 0;
  let total: number | undefined;
  await pendingUpdate.download((event) => {
    if (event.event === 'Started') {
      total = event.data.contentLength;
      onProgress?.(total ? 0 : null);
    } else if (event.event === 'Progress') {
      downloaded += event.data.chunkLength;
      onProgress?.(total ? Math.min(100, Math.round((downloaded / total) * 100)) : null);
    } else {
      onProgress?.(100);
    }
  }, { timeout: UPDATE_DOWNLOAD_TIMEOUT_MS });
  pendingUpdateDownloaded = true;
}

export async function installDownloadedAppUpdate(): Promise<void> {
  if (!pendingUpdate || !pendingUpdateDownloaded) {
    throw new Error('The downloaded update is no longer available');
  }
  await invoke('prepare_for_update');
  const { relaunch } = await import('@tauri-apps/plugin-process');
  try {
    await pendingUpdate.install();
  } catch (error) {
    // prepare_for_update intentionally stops managed services. Relaunch the
    // current version if installation fails so the app is not left half-down.
    await relaunch();
    throw error;
  }
  await relaunch();
}

/** Compatibility helper for callers that explicitly request an immediate update. */
export async function installAppUpdate(onProgress?: (percent: number) => void): Promise<void> {
  if (!pendingInfo) {
    const result = await checkForAppUpdate();
    if (result.status !== 'available') throw new Error('No update is available');
  }
  await downloadAppUpdate((value) => onProgress?.(value ?? 0));
  await installDownloadedAppUpdate();
}

export function getPendingUpdateInfo(): AppUpdateInfo | null {
  return pendingInfo;
}

export function hasDownloadedAppUpdate(): boolean {
  return pendingUpdate !== null && pendingUpdateDownloaded;
}
