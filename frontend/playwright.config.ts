import { defineConfig, devices } from '@playwright/test';

/**
 * End-to-end tests run against a live backend + frontend pair.
 *
 * Required environment:
 *   E2E_BASE_URL   frontend origin (default http://localhost:5173)
 *   E2E_API_URL    backend API base (default http://127.0.0.1:8080/api/v1)
 *   E2E_EMAIL      owner account (default owner@fixphone.local)
 *   E2E_PASSWORD   owner password (default Owner#12345)
 *
 * Start the stack with scripts/dev-db.ps1, `go run ./cmd/api`, and
 * `npm run dev`; see docs/18-implementasi-tahap-11.md.
 */
export default defineConfig({
  testDir: './e2e',
  timeout: 90_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    locale: 'id-ID',
    timezoneId: 'Asia/Jakarta',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
