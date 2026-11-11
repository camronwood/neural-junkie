/** Shared Advanced debug disclosure for collaboration status banners. */
export function CollabAdvancedDebug({
  collabIdPrefix,
  testId,
}: {
  collabIdPrefix?: string;
  testId: string;
}) {
  const cmd = collabIdPrefix
    ? `make debug-collab COLAB=${collabIdPrefix} LIVE=1`
    : 'make debug-collab LIVE=1';
  return (
    <details data-testid={testId} style={{ marginTop: 8 }}>
      <summary style={{ fontSize: 11, color: 'var(--text-secondary, #888)', cursor: 'pointer' }}>
        Advanced debug
      </summary>
      <p style={{ margin: '6px 0 0', fontSize: 11, color: 'var(--text-secondary, #999)', lineHeight: 1.4 }}>
        From a terminal in the Neural Junkie repo:{' '}
        <code style={{ fontSize: 11 }}>{cmd}</code>
      </p>
    </details>
  );
}
