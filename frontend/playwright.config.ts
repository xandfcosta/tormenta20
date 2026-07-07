import { defineConfig, devices } from '@playwright/test'

/**
 * Visual regression baseline for the Controlled Decay UI/UX revamp.
 *
 * Scope MVP: unauth-only routes (landing/login/register). Captures
 * light + dark themes across 3 viewports. Authed pages (character
 * sheet, session tracker) need deterministic backend seed and land in
 * a follow-up once the seed script exists.
 *
 * Snapshots are committed under `e2e/__screenshots__/`. Diffs beyond
 * the `maxDiffPixelRatio` threshold fail the run.
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    /* Consistent baseline: disable animations so we don't race the
     * noise-layer redraw or font-load reflow. */
    launchOptions: {
      args: ['--force-prefers-reduced-motion'],
    },
  },
  expect: {
    toHaveScreenshot: {
      maxDiffPixelRatio: 0.02,
      animations: 'disabled',
    },
  },
  /* Chromium for all viewports: visual-regression baselines want a
   * single rendering engine so cross-browser font/glyph rasterization
   * doesn't inflate diffs. Device presets are used only for viewport
   * + userAgent hints. */
  projects: [
    {
      name: 'phone',
      use: {
        ...devices['Pixel 7'],
        viewport: { width: 390, height: 844 },
      },
    },
    {
      name: 'tablet',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 820, height: 1180 },
      },
    },
    {
      name: 'desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
      },
    },
  ],
  webServer: {
    command: 'pnpm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
})
