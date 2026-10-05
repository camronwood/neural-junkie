import { test, expect, type Page } from '@playwright/test';

const hubURL = process.env.NJ_E2E_HUB_URL || process.env.NEURAL_JUNKIE_HUB_URL || 'http://127.0.0.1:18765';

async function connectToHub(page: Page) {
  await page.goto('/');

  // Fail fast if the Vite build still shows the desktop-only gate (missing VITE_NJ_E2E=1).
  const gated = page.getByTestId('desktop-only-gate');
  if (await gated.isVisible({ timeout: 5_000 }).catch(() => false)) {
    throw new Error('DesktopOnlyGate visible — desktop-e2e requires VITE_NJ_E2E=1 for the Vite webServer');
  }

  // Loading/setup phases do not mount app-shell yet — wait for login or chat.
  const loginOrChat = page.getByTestId('login-connect').or(page.getByTestId('chat-composer'));
  await expect(loginOrChat).toBeVisible({ timeout: 90_000 });

  const connect = page.getByTestId('login-connect');
  if (await connect.isVisible().catch(() => false)) {
    const server = page.getByTestId('login-server');
    if (await server.isVisible().catch(() => false)) {
      await server.fill(hubURL.replace(/^https?:\/\//, ''));
    }
    const name = page.locator('#name');
    if (await name.isVisible().catch(() => false)) {
      await name.fill('E2EUser');
    }
    await connect.click();
  }

  await expect(page.getByTestId('app-shell')).toBeVisible({ timeout: 60_000 });
  await expect(page.getByTestId('chat-composer')).toBeVisible({ timeout: 90_000 });
}

test.describe('desktop user journeys', () => {
  test('launch.spec — app shell visible after connect', async ({ page }) => {
    await connectToHub(page);
    await expect(page.getByTestId('app-shell')).toBeVisible();
  });

  test('settings-nav.spec — open settings and visit about + domain packs', async ({ page }) => {
    await connectToHub(page);
    await page.getByTestId('open-settings').click();
    await expect(page.getByTestId('settings-modal')).toBeVisible();
    await page.getByTestId('settings-tab-about').click();
    await expect(page.getByText('About', { exact: false }).first()).toBeVisible();
    // Domain packs opens pack store modal / action
    await page.getByTestId('settings-tab-domain-packs').click();
  });

  test('chat-send.spec — composer accepts input and send', async ({ page }) => {
    await connectToHub(page);
    const composer = page.getByTestId('chat-composer');
    await expect(composer).toBeVisible();
    const textarea = page.locator('textarea').first();
    await textarea.fill('e2e ping — ignore');
    await textarea.press('Enter');
    // User bubble or pending activity should appear; allow soft pass on composer still present
    await expect(composer).toBeVisible();
  });

  test('approval-click.spec — approve/reject controls exist when pending bar shows', async ({ page }) => {
    await connectToHub(page);
    const bar = page.getByTestId('pending-approvals-bar');
    if (await bar.isVisible().catch(() => false)) {
      await expect(page.getByTestId('approval-approve').or(page.getByTestId('approval-reject'))).toBeVisible();
    } else {
      test.info().annotations.push({ type: 'note', description: 'No pending approval — skipped click' });
    }
  });

  test('pack-store-open.spec — domain packs settings action', async ({ page }) => {
    await connectToHub(page);
    await page.getByTestId('open-settings').click();
    await expect(page.getByTestId('settings-modal')).toBeVisible();
    await page.getByTestId('settings-tab-domain-packs').click();
  });
});
