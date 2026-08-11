/** Browser/WebView geolocation for Maps session share and maps_locate. */

export interface DevicePosition {
  lat: number;
  lon: number;
  accuracy_m: number;
  captured_at: string;
}

export class GeolocationUnavailableError extends Error {
  readonly code: 'unsupported' | 'denied' | 'unavailable' | 'timeout';

  constructor(code: GeolocationUnavailableError['code'], message: string) {
    super(message);
    this.name = 'GeolocationUnavailableError';
    this.code = code;
  }
}

export function geolocationSupported(): boolean {
  return typeof navigator !== 'undefined' && typeof navigator.geolocation?.getCurrentPosition === 'function';
}

export function geolocationErrorMessage(err: unknown): string {
  if (err instanceof GeolocationUnavailableError) {
    return err.message;
  }
  if (err instanceof Error && err.message.trim()) {
    return err.message;
  }
  return 'Could not read this device location.';
}

function mapPositionError(err: GeolocationPositionError): GeolocationUnavailableError {
  switch (err.code) {
    case err.PERMISSION_DENIED:
      return new GeolocationUnavailableError(
        'denied',
        'Location permission was denied. Enable it in system settings to share your location.',
      );
    case err.TIMEOUT:
      return new GeolocationUnavailableError('timeout', 'Timed out waiting for a location reading.');
    default:
      return new GeolocationUnavailableError(
        'unavailable',
        'Location is unavailable on this device. Check that Location Services are on.',
      );
  }
}

/** One-shot GPS read. High accuracy is off by default to keep the OS prompt cheap. */
export function getCurrentDevicePosition(options?: {
  timeoutMs?: number;
  enableHighAccuracy?: boolean;
  maximumAgeMs?: number;
}): Promise<DevicePosition> {
  if (!geolocationSupported()) {
    return Promise.reject(
      new GeolocationUnavailableError(
        'unsupported',
        'This window cannot read device location (geolocation is unavailable).',
      ),
    );
  }
  const timeoutMs = options?.timeoutMs ?? 15000;
  const enableHighAccuracy = options?.enableHighAccuracy === true;
  const maximumAgeMs = options?.maximumAgeMs ?? 0;

  return new Promise((resolve, reject) => {
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        resolve({
          lat: pos.coords.latitude,
          lon: pos.coords.longitude,
          accuracy_m: Number.isFinite(pos.coords.accuracy) ? pos.coords.accuracy : 0,
          captured_at: new Date(pos.timestamp || Date.now()).toISOString(),
        });
      },
      (err) => reject(mapPositionError(err)),
      {
        enableHighAccuracy,
        timeout: timeoutMs,
        maximumAge: maximumAgeMs,
      },
    );
  });
}
