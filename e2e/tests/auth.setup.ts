import { type Page, expect, test as setup } from '@playwright/test'

// Contas da SEED (`engine-go/seed.sql`), aplicada pelo projeto `semente` num
// banco recriado a cada corrida (`data/e2e.db`). O mestre é dono da campanha 1;
// o jogador é só membro dela — o par de que os specs de papel precisam.
//
// AS DUAS VARREDURAS ABAIXO são REDUNDANTES com o banco novo por corrida: não
// há o que varrer num arquivo que acabou de nascer. Ficam porque custam ~1,5s e
// a ordem de corte da casa é "escrever o substituto, ver verde, DEPOIS apagar".
const PASSWORD = process.env.E2E_PASSWORD ?? 'mestre123456'
const GM_EMAIL = process.env.E2E_EMAIL ?? 'mestre@t20.local'
const PLAYER_EMAIL = process.env.E2E_PLAYER_EMAIL ?? 'jogador@t20.local'

/**
 * Entra pela TELA de verdade e guarda a sessão (token e cookies), para os specs
 * começarem num contexto autenticado.
 *
 * UMA vez por papel, e não num `beforeEach`: entrar dentro de cada teste custa
 * uma carga de página por teste, e a suíte inteira paga por uma garantia que
 * não muda entre eles.
 */
async function signIn(page: Page, email: string, file: string): Promise<void> {
  await page.goto('/entrar')
  await page.getByLabel('E-mail').fill(email)
  await page.getByLabel('Senha').fill(PASSWORD)
  await page.getByRole('button', { name: 'Entrar' }).click()

  // Caiu no Hub, que é o menu principal do jogo.
  await expect(page.getByText('Meus Heróis')).toBeVisible()
  await page.context().storageState({ path: file })
}

/**
 * Varre campanhas de teste que sobraram de uma execução anterior.
 *
 * O único spec que ESCREVE de verdade cria uma "E2E Descartável <timestamp>" e
 * a apaga no fim. Quando ele morre no meio, a campanha sobrevive e não fica
 * quieta: ela rouba o holofote da lista de Campanhas, e quem falha é OUTRO
 * spec, com um "element(s) not found" que não aponta para lugar nenhum.
 *
 * Mora aqui dentro do login, e não num `setup` irmão, porque precisa da sessão
 * já feita: com dois workers no CI dois setups irmãos podem correr juntos, e
 * uma varredura que roda deslogada não falha — ela não acha nada e volta
 * dizendo que limpou. Limpeza silenciosamente vazia é pior que nenhuma.
 *
 * Nomeia pelo PREFIXO, nunca por id: apagar por id seria apagar seed.
 */
async function varrerCronicasDeTeste(page: Page): Promise<void> {
  const list = await page.request.get('/api/campanhas')
  expect(list.ok(), 'a varredura de campanhas de teste rodou sem sessão').toBe(true)
  const chronicles = (await list.json()) as { id: number; name: string }[]
  const leftovers = chronicles.filter((c) => c.name.startsWith('E2E Descartável'))
  for (const rest of leftovers) await page.request.delete(`/api/campanhas/${rest.id}`)
  if (leftovers.length > 0) console.log(`[setup] ${leftovers.length} campanha(s) de teste varrida(s)`)
}

/**
 * Tira dos personagens da seed as condições que uma execução ANTERIOR deixou.
 *
 * O spec da sessão aplica Abalado, Agarrado e Cego para medir a faixa com a
 * fileira cheia, e as tira no fim. Só que a limpeza dele mora no CORPO do
 * teste: quando o teste falha por qualquer motivo, ela não roda, as condições
 * ficam gravadas na ficha — e a partir daí ele falha PARA SEMPRE, porque o
 * seletor não oferece uma condição que já está aplicada. Um teste que só passa
 * numa seed limpa não é repetível (F.I.R.S.T), e este se envenenava sozinho.
 *
 * Varre no SETUP, que roda antes de tudo e não depende de nenhum teste ter
 * terminado bem. Mesma escolha da varredura de campanhas acima.
 *
 * Roda nas DUAS sessões: `/api/personagens` lista só o que a sessão POSSUI, e a
 * ficha que o spec suja é do JOGADOR. Varrendo só com o mestre, a limpeza acha
 * zero e vai embora dizendo que limpou.
 */
async function varrerCondicoesDeTeste(page: Page): Promise<void> {
  const list = await page.request.get('/api/personagens')
  expect(list.ok(), 'a varredura de condições rodou sem sessão').toBe(true)
  const sheets = (await list.json()) as { id: number; activeConditions?: string }[]
  let swept = 0
  for (const sheet of sheets) {
    if (!sheet.activeConditions || sheet.activeConditions === '[]') continue
    const cleared = await page.request.patch(`/api/personagens/${sheet.id}/conditions`, {
      data: { activeConditions: [] },
    })
    expect(cleared.ok(), `não consegui limpar as condições da ficha ${sheet.id}`).toBe(true)
    swept++
  }
  if (swept > 0) console.log(`[setup] ${swept} ficha(s) com condição de teste varrida(s)`)
}

setup('authenticate', async ({ page }) => {
  await signIn(page, GM_EMAIL, '.auth/user.json')
  await varrerCronicasDeTeste(page)
  await varrerCondicoesDeTeste(page)
})

setup('authenticate as player', async ({ page }) => {
  await signIn(page, PLAYER_EMAIL, '.auth/player.json')
  await varrerCondicoesDeTeste(page)
})

