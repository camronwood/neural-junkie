import { describe, expect, it } from 'vitest';
import { GeolocationUnavailableError, geolocationErrorMessage } from './geolocation';

describe('geolocationErrorMessage', () => {
  it('uses GeolocationUnavailableError message', () => {
    expect(
      geolocationErrorMessage(new GeolocationUnavailableError('denied', 'Location permission was denied.')),
    ).toBe('Location permission was denied.');
  });

  it('falls back for unknown errors', () => {
    expect(geolocationErrorMessage('nope')).toBe('Could not read this device location.');
  });
});
