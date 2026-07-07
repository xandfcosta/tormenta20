import { defineConfig, devices } from '@playwright/test'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

/**
 * Visual regression baseline for the Controlled Decay UI/UX revamp.
 *
 * Two layers:
 *  1. Public routes (`visual.spec.ts`) — landing/login/register, no
 *     auth. Sanity check that global tokens don't regress.
 *  2. Authenticated routes (`visual-authed.spec.ts`) — character
 *     sheet, campaigns, session tracker. Requires the backend to be
 *     running with the seed applied (`backend/prisma/seed.ts`).
 *
 * Auth is captured once by the `setup` project which logs in via the
 * seeded user and writes storage state to `.auth/user.json`; the
 * `authed-*` projects reuse that state so every scenario starts
 * pre-authenticated without repeating the login flow.
 */

const AUTH_STATE = path.resolve(__dirname, '.auth', 'user.json')

const PHONE_VIEWPORT = { width: 390, height: 844 }
const TABLET_VIEWPORT = { width: 820, height: 1180 }
const DESKTOP_VIEWPORT = { width: 1440, height: 900 }

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    launchOptions: {
      args: ['--force-prefers-reduced-motion'],
    },
  },
  expect: {
    toHaveScreenshot: {
      /* 0.08 accommodates font-rendering drift between the local
       * dev OS (Arch — libfreetype 2.14) and CI (Ubuntu 24.04 —
       * libfreetype 2.13). Heavy sheet routes hit ~6% anti-alias
       * diff at rest. Follow-up: run baselines through a Docker
       * image pinned to the CI toolchain, then tighten this. */
      maxDiffPixelRatio: 0.08,
      animations: 'disabled',
    },
  },
  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.ts$/,
      use: { ...devices['Desktop Chrome'], viewport: DESKTOP_VIEWPORT },
    },
    /* Public routes — no storageState. */
    {
      name: 'public-phone',
      testMatch: /visual\.spec\.ts$/,
      use: { ...devices['Pixel 7'], viewport: PHONE_VIEWPORT },
    },
    {
      name: 'public-tablet',
      testMatch: /visual\.spec\.ts$/,
      use: { ...devices['Desktop Chrome'], viewport: TABLET_VIEWPORT },
    },
    {
      name: 'public-desktop',
      testMatch: /visual\.spec\.ts$/,
      use: { ...devices['Desktop Chrome'], viewport: DESKTOP_VIEWPORT },
    },
    /* Authed routes — depend on `setup` for cookies. */
    {
      name: 'authed-phone',
      testMatch: /visual-authed\.spec\.ts$/,
      dependencies: ['setup'],
      use: {
        ...devices['Pixel 7'],
        viewport: PHONE_VIEWPORT,
        storageState: AUTH_STATE,
      },
    },
    {
      name: 'authed-tablet',
      testMatch: /visual-authed\.spec\.ts$/,
      dependencies: ['setup'],
      use: {
        ...devices['Desktop Chrome'],
        viewport: TABLET_VIEWPORT,
        storageState: AUTH_STATE,
      },
    },
    {
      name: 'authed-desktop',
      testMatch: /visual-authed\.spec\.ts$/,
      dependencies: ['setup'],
      use: {
        ...devices['Desktop Chrome'],
        viewport: DESKTOP_VIEWPORT,
        storageState: AUTH_STATE,
      },
    },
  ],
  /* Always reuse an existing frontend on :5173. CI boots backend +
   * frontend explicitly (needs seed first); locally the dev shell is
   * usually already running. If nothing's up, playwright starts one. */
  webServer: {
    command: 'pnpm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: true,
    timeout: 60_000,
  },
})
