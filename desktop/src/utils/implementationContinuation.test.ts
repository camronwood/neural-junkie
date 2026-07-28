import { describe, expect, it } from 'vitest';
import {
  hasContentDeliverySignals,
  hasFileExportSignals,
  hasImplementationContinuationSignals,
  hasImplementationRequestSignals,
  hasImplementationStatusCheckSignals,
  hasErrorLogFollowUpSignals,
  hasPriorReferenceExportSignals,
  channelHasImplementationThread,
} from './implementationContinuation';

describe('implementationContinuation (phrase banks removed)', () => {
  it('returns false for former NL signal helpers', () => {
    expect(hasImplementationContinuationSignals('yes please go ahead')).toBe(false);
    expect(hasImplementationRequestSignals('the app is not booting up can you fix it?')).toBe(
      false
    );
    expect(hasImplementationStatusCheckSignals('is it fixed?')).toBe(false);
    expect(hasContentDeliverySignals('Can you create a linkedin article about this app?')).toBe(
      false
    );
    expect(hasFileExportSignals('store that artical in a markdown file')).toBe(false);
    expect(
      hasPriorReferenceExportSignals('save what you wrote a few messages back to a markdown file')
    ).toBe(false);
    expect(hasErrorLogFollowUpSignals('still getting error TS2307')).toBe(false);
  });

  it('channelHasImplementationThread uses metadata only', () => {
    expect(
      channelHasImplementationThread([{ metadata: { implementation_session: true } }])
    ).toBe(true);
    expect(channelHasImplementationThread([{ type: 'text', metadata: {} }])).toBe(false);
  });
});
