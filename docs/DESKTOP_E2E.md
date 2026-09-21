# Desktop UI E2E (Playwright)

Click-through journeys that mirror real user paths in the desktop shell. Used by overnight away (`make desktop-e2e` / `scripts/desktop-e2e.sh`).

## Default mode (Vite + live hub)

```bash
# Hub already running (overnight sets this up)
export NEURAL_JUNKIE_HUB_URL=http://127.0.0.1:18765
make desktop-e2e
```

Playwright starts Vite on port **1420**, connects via the login screen (`data-testid=login-connect`), then:

| Spec intent | Selectors |
|-------------|-----------|
| Launch / shell | `app-shell` |
| Chat send | `chat-composer` + textarea Enter |
| Settings nav | `open-settings`, `settings-modal`, `settings-tab-about` |
| Domain packs | `settings-tab-domain-packs` |
| Approvals | `pending-approvals-bar`, `approval-approve` / `approval-reject` (when present) |

Reports: `docs/testing/desktop-e2e-*.md`, traces under `desktop/test-results/`.

## Optional Tauri WebDriver

Set `NJ_E2E_TAURI=1` and install [`tauri-driver`](https://v2.tauri.app/develop/tests/webdriver/) on the overnight Mac. Wire the driver URL via `NJ_E2E_BASE_URL` / `NJ_E2E_NO_WEBSERVER=1` once your local driver is listening. Prefer the Vite path until the driver is stable.

## Adding journeys

1. Add stable `data-testid` hooks in React.
2. Extend `desktop/e2e/user-journeys.spec.ts`.
3. Do **not** delete failing specs to go green — fix product or selectors.

## Dependency

`@playwright/test` is a desktop `devDependency`. First run may need:

```bash
cd desktop && npm ci && npx playwright install chromium
```
