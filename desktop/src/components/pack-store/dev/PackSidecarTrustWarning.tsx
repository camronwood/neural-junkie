import { PACK_SIDECAR_TRUST_COPY } from './packSidecarTrust';

interface PackSidecarTrustWarningProps {
  /** When false, hide the banner (caller already decided no sidecar risk). */
  visible?: boolean;
  className?: string;
}

export function PackSidecarTrustWarning({
  visible = true,
  className = '',
}: PackSidecarTrustWarningProps) {
  if (!visible) return null;
  return (
    <div
      role="status"
      data-testid="pack-sidecar-trust-warning"
      className={`rounded-lg border border-amber-700/40 bg-amber-950/30 px-3 py-2 text-xs text-amber-100/90 ${className}`}
    >
      <p className="font-medium text-amber-100">Sidecar trust warning</p>
      <p className="mt-1 leading-relaxed text-amber-100/80">{PACK_SIDECAR_TRUST_COPY}</p>
    </div>
  );
}
