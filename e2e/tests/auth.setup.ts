import { type Page, expect, test as setup } from '@playwright/test'

// Seeded accounts from the Go backend (engine-go/data/t20-dev.db). The GM owns
// campaign 1; the player is only a member of it — the pair the role-gating
// specs need (ALE-24).
const PASSWORD = process.env.E2E_PASSWORD ?? 'mestre123456'
const GM_EMAIL = process.env.E2E_EMAIL ?? 'mestre@t20.local'
const PLAYER_EMAIL = process.env.E2E_PLAYER_EMAIL ?? 'jogador@t20.local'

/**
 * Logs in through the real UI and persists the session (localStorage token +
 * cookies) so the specs start on an authenticated context — the
 * framework-agnostic way to handle auth (survives the Solid migration).
 *
 * Done ONCE per role here rather than in a beforeEach: signing in inside every
 * test adds a full page load per test, and against the dev server that raced
 * Vite's dependency re-optimization reload and made the suite flaky.
 */
async function signIn(page: Page, email: string, file: string): Promise<void> {
  await page.goto('/login')
  await page.getByLabel('E-mail').fill(email)
  await page.getByLabel('Senha').fill(PASSWORD)
  await page.getByRole('button', { name: 'Entrar' }).click()

  // Landed on the Hub (the game's main menu).
  await expect(page.getByText('Meus Heróis')).toBeVisible()
  await page.context().storageState({ path: file })
}

/**
 * Varre crônicas de teste que sobraram de uma execução anterior.
 *
 * O único spec que ESCREVE de verdade cria uma "E2E Descartável <timestamp>" e
 * a apaga no fim. Quando ele morre no meio — ou quando alguém interrompe a
 * suíte —, a campanha sobrevive, e ela não fica quieta: rouba o holofote da
 * lista de Crônicas, e quem falha é OUTRO spec, o que espera "Continuar a
 * sessão" da campanha 1, com um "element(s) not found" que não aponta para
 * lugar nenhum. O banco de desenvolvimento não é recriado entre execuções,
 * então o resto de ontem derruba a suíte de hoje.
 *
 * Mora aqui dentro do login, e não num `setup` irmão, porque precisa da sessão
 * já feita: com dois workers no CI dois setups irmãos podem correr juntos, e
 * uma varredura que roda deslogada não falha — ela não acha nada e volta
 * dizendo que limpou. Limpeza silenciosamente vazia é pior que nenhuma.
 *
 * Nomeia pelo PREFIXO, nunca por id: apagar por id seria apagar seed.
 */
async function varrerCronicasDeTeste(page: Page): Promise<void> {
  const lista = await page.request.get('/api/campaigns')
  expect(lista.ok(), 'a varredura de crônicas de teste rodou sem sessão').toBe(true)
  const cronicas = (await lista.json()) as { id: number; name: string }[]
  const restos = cronicas.filter((c) => c.name.startsWith('E2E Descartável'))
  for (const resto of restos) await page.request.delete(`/api/campaigns/${resto.id}`)
  if (restos.length > 0) console.log(`[setup] ${restos.length} crônica(s) de teste varrida(s)`)
}

setup('authenticate', async ({ page }) => {
  await signIn(page, GM_EMAIL, '.auth/user.json')
  await varrerCronicasDeTeste(page)
})

setup('authenticate as player', async ({ page }) => {
  await signIn(page, PLAYER_EMAIL, '.auth/player.json')
})

