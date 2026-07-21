import {
  isPolicyMandatory,
  isVersionBelow,
  type AppUpdateInfo,
  type AppUpdatePolicy,
} from './appUpdater';

const CACHE_KEY = 'nj-updater-accepted-update-v1';

interface CachedUpdateEvidence {
  schemaVersion: 1;
  targetVersion: string;
  notes?: string;
  policy: AppUpdatePolicy;
  bundleVerifiedAt: string;
}

function isPolicy(value: unknown): value is AppUpdatePolicy {
  if (!value || typeof value !== 'object') return false;
  const policy = value as Partial<AppUpdatePolicy>;
  return (
    policy.schemaVersion === 1 &&
    (policy.channel === 'beta' || policy.channel === 'stable') &&
    (policy.severity === 'normal' || policy.severity === 'critical') &&
    (policy.enforcement === 'optional' || policy.enforcement === 'mandatory') &&
    typeof policy.rolloutPercentage === 'number' &&
    Number.isFinite(policy.rolloutPercentage) &&
    policy.rolloutPercentage >= 0 &&
    policy.rolloutPercentage <= 100 &&
    typeof policy.rolloutSeed === 'string' &&
    policy.rolloutSeed.length > 0 &&
    (policy.mandatoryAfter === undefined ||
      (typeof policy.mandatoryAfter === 'string' &&
        Number.isFinite(Date.parse(policy.mandatoryAfter)))) &&
    (policy.minimumSupportedVersion === undefined ||
      typeof policy.minimumSupportedVersion === 'string')
  );
}

function parseEvidence(value: unknown): CachedUpdateEvidence | null {
  if (!value || typeof value !== 'object') return null;
  const evidence = value as Partial<CachedUpdateEvidence>;
  if (
    evidence.schemaVersion !== 1 ||
    typeof evidence.targetVersion !== 'string' ||
    evidence.targetVersion.length === 0 ||
    (evidence.notes !== undefined && typeof evidence.notes !== 'string') ||
    !isPolicy(evidence.policy) ||
    typeof evidence.bundleVerifiedAt !== 'string' ||
    !Number.isFinite(Date.parse(evidence.bundleVerifiedAt))
  ) {
    return null;
  }
  return evidence as CachedUpdateEvidence;
}

export function saveAcceptedUpdate(info: AppUpdateInfo): void {
  const evidence: CachedUpdateEvidence = {
    schemaVersion: 1,
    targetVersion: info.version,
    notes: info.notes,
    policy: info.policy,
    bundleVerifiedAt: new Date().toISOString(),
  };
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify(evidence));
  } catch {
    // Cache continuity must never prevent an update from being installed.
  }
}

export function clearAcceptedUpdate(): void {
  try {
    localStorage.removeItem(CACHE_KEY);
  } catch {
    // Local storage may be unavailable in hardened webviews.
  }
}

export function loadAcceptedUpdate(
  currentVersion: string,
  channel: 'beta' | 'stable',
): AppUpdateInfo | null {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;
    const evidence = parseEvidence(JSON.parse(raw));
    if (
      !evidence ||
      evidence.policy.channel !== channel ||
      !isVersionBelow(currentVersion, evidence.targetVersion)
    ) {
      clearAcceptedUpdate();
      return null;
    }
    return {
      available: true,
      version: evidence.targetVersion,
      notes: evidence.notes,
      policy: evidence.policy,
      mandatory: isPolicyMandatory(evidence.policy, Date.now(), currentVersion),
    };
  } catch {
    clearAcceptedUpdate();
    return null;
  }
}
