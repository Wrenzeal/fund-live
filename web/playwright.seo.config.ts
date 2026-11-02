import { defineConfig, devices } from '@playwright/test'

// Build first with BACKEND_URL=http://127.0.0.1:18084 while the fixture is running.
// Server-side requests cannot be intercepted by browser page.route mocks.
export default defineConfig({
  testDir: './tests/e2e',
  testMatch: ['seo-foundation.spec.ts', 'seo-server-rendering.spec.ts', 'seo-policy.spec.ts'],
  timeout: 30_000,
  expect: { timeout: 7_000 },
  fullyParallel: true,
  workers: 2,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:3101',
    screenshot: 'only-on-failure', trace: 'retain-on-failure',
  },
  projects: [{ name: 'seo', use: { ...devices['Desktop Chrome'], ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}) } }],
  webServer: [
    { command: 'node tests/fixtures/seo-backend.mjs', url: 'http://127.0.0.1:18084/', reuseExistingServer: process.env.SEO_FIXTURE_RUNNING === 'true' || !process.env.CI },
    { command: 'node node_modules/next/dist/bin/next start --hostname 127.0.0.1 --port 3101', url: 'http://127.0.0.1:3101/', env: { BACKEND_URL: 'http://127.0.0.1:18084', NEXT_TELEMETRY_DISABLED: '1' }, reuseExistingServer: false, timeout: 60_000 },
  ],
})
