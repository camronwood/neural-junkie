import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PackSidecarTrustWarning } from './PackSidecarTrustWarning';
import { PACK_SIDECAR_TRUST_COPY } from './packSidecarTrust';

describe('PackSidecarTrustWarning', () => {
  it('renders trust copy when visible', () => {
    render(<PackSidecarTrustWarning />);
    expect(screen.getByTestId('pack-sidecar-trust-warning')).toBeTruthy();
    expect(screen.getByText(PACK_SIDECAR_TRUST_COPY)).toBeTruthy();
  });

  it('hides when visible is false', () => {
    const { container } = render(<PackSidecarTrustWarning visible={false} />);
    expect(container.querySelector('[data-testid="pack-sidecar-trust-warning"]')).toBeNull();
  });
});
