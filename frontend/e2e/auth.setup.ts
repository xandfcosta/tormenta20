import { expect, test as setup } from '@playwright/test'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

/**
 * Playwright auth setup — runs once at the start of the run. Logs in
 * via the frontend so cookies land in the browser context, then
 * persists storage state to `.auth/user.json`. Every `authed-*`
 * project reuses that file via `storageState`.
 *
 * Deps: the backend must be seeded with the deterministic user (see
 * `backend/prisma/seed.ts`).
 */

const AUTH_FILE = path.resolve(__dirname, '..', '.auth', 'user.json')

setup('login as seed user', async ({ page }) => {
  fs.mkdirSync(path.dirname(AUTH_FILE), { recursive: true })

  await page.goto('/login')
  await page.getByLabel(/email/i).fill('seed@t20.dev')
  await page.getByLabel(/password/i).fill('seed-password-1234')
  await page.getByRole('button', { name: /sign in/i }).click()

  /* Successful login redirects to `/`. If auth fails we land on
   * `/login?error=…`; the URL assertion catches that faster than
   * waiting for a nav timeout. */
  await expect(page).toHaveURL(/^http:\/\/localhost:5173\/(characters)?$/, {
    timeout: 15_000,
  })

  await page.context().storageState({ path: AUTH_FILE })
})
