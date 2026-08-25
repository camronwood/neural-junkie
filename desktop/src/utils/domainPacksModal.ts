/** Cross-surface opener for the Domain packs modal (Settings deep-link, toolbar, coach). */
export const OPEN_DOMAIN_PACKS_EVENT = 'nj-open-domain-packs';

export function requestOpenDomainPacksModal(): void {
  window.dispatchEvent(new CustomEvent(OPEN_DOMAIN_PACKS_EVENT));
}

export function onOpenDomainPacksModal(listener: () => void): () => void {
  window.addEventListener(OPEN_DOMAIN_PACKS_EVENT, listener);
  return () => window.removeEventListener(OPEN_DOMAIN_PACKS_EVENT, listener);
}
