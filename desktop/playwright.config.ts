import { defineConfig, devices } from '@playwright/test';

/**
 * Desktop UI click journeys for overnight away ops.
 * Default: Vite web shell + live hub (NJ_E2E_HUB_URL).
 * Optional: NJ_E2E_TAURI=1 launches via tauri-driver (see docs/DESKTOP_E2E.md).
 */
const port = Number(process.env.NJ_E2E_PORT || 1420);
const baseURL = process.env.NJ_E2E_BASE_URL || `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'e2e-report' }]],
  timeout: 120_000,
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  webServer: process.env.NJ_E2E_NO_WEBSERVER
    ? undefined
    : {
        command: `npm run dev -- --host 127.0.0.1 --port ${port}`,
        url: baseURL,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
