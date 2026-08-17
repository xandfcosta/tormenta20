import { type Page, expect, test } from '@playwright/test'
import { VIEWPORTS } from './support/viewports'

/**
 * The campaign scene seen by a PLAYER, not the GM (ALE-24). Every other spec
 * runs as the seeded GM through the shared storageState; this one opts out and
 * signs in as the player who is a member of campaign 1 but owns nothing there.
 *
 * The real gate is the server's — this is the UX half: offering a member a
 * button that can only 403 is worse than not offering it.
 */
test.use({ storageState: '.auth/player.json' })

const CAMPAIGN = '/campaigns/1' // do mestre; o jogador é só membro

/**
 * Opens a section and waits for the tome to actually be on screen.
 *
 * Every check here is about what must NOT be there, and an assertion of absence
 * passes on a blank page — so each one has to start from a scene proven to have
 * rendered. `networkidle` because in dev the first cold visit to a route makes
 * Vite re-optimize deps and force a reload, which strands the default wait on
 * the discarded document.
 */
async function openSection(page: Page, tab: string): Promise<void> {
  await page.goto(`${CAMPAIGN}?tab=${tab}`, { waitUntil: 'networkidle' })
  await expect(page.getByRole('heading', { name: /Snapshot Test ALE-33/i })).toBeVisible()
}

test.describe('Campanha vista pelo jogador', () => {
  test('o rail não oferece a seção do mestre', async ({ page }) => {
    await openSection(page, 'visao')

    await expect(page.getByRole('tab', { name: 'Visão geral' })).toBeVisible()
    await expect(page.getByRole('tab', { name: 'Membros' })).toBeVisible()
    await expect(page.getByRole('tab', { name: 'Config' })).toHaveCount(0)
  })

  test('nenhuma ação de dono aparece nas seções', async ({ page }) => {
    await openSection(page, 'visao')

    // Percorre o rail clicando, como o jogador faria — e não com três loads,
    // que é o que a cena existe para evitar.
    for (const secao of ['Visão geral', 'Sessões', 'Membros']) {
      await page.getByRole('tab', { name: secao }).click()
      await expect(page.getByRole('tab', { name: secao })).toHaveAttribute('data-selected', '')

      await expect(page.getByRole('button', { name: /Convite/ })).toHaveCount(0)
      await expect(page.getByRole('button', { name: /^Sessão \d+$/ })).toHaveCount(0)
      await expect(page.getByRole('button', { name: 'Editar' })).toHaveCount(0)
      await expect(page.getByRole('button', { name: /Excluir campanha/ })).toHaveCount(0)
      await expect(page.getByLabel(/^Remover /)).toHaveCount(0)
    }
  })

  // `?tab=config` digitado à mão não pode revelar a seção — e os bumpers
  // (PgUp/PgDn) também só andam pelas seções que o rail do jogador tem.
  test('pedir a seção do mestre pela URL cai na visão geral', async ({ page }) => {
    await openSection(page, 'config')

    // Caiu numa seção de verdade, não numa tela vazia.
    await expect(page.getByRole('tab', { name: 'Visão geral' })).toHaveAttribute('data-selected', '')
    await expect(page.getByText('Zona de perigo')).toHaveCount(0)
    await expect(page.getByRole('button', { name: /Excluir campanha/ })).toHaveCount(0)
  })

  // O jogador PERDE ações de escrita, não a mesa: ele continua lendo a crônica
  // e entrando na sessão ao vivo.
  test('o jogador ainda lê a crônica e entra na sessão ao vivo', async ({ page }) => {
    await openSection(page, 'sessoes')

    await expect(page.getByText('JOGANDO')).toBeVisible()
    await expect(page.getByRole('button', { name: /Continuar a sessão/ })).toBeVisible()
    await expect(page.getByText('Sessão 5')).toBeVisible()
  })

  /**
   * ALE-96. Ler `.data` de uma query PENDENTE suspende, e o boundary mais
   * próximo é o `Suspense` que o solid-router põe em todo route match — então a
   * partida INTEIRA (banner "Ao vivo", presença, a saída) é desanexada enquanto
   * a ficha do próprio jogador está em voo. O `PlayerSheet` tinha um Skeleton
   * escrito para exatamente esse momento que nunca podia pintar, porque o
   * suspend acontecia antes de o `Show` ser avaliado.
   *
   * Por que e2e: só um browser de verdade testemunha. Uma montagem em jsdom não
   * tem router, logo não tem Suspense, e ali a leitura pendente devolve
   * `undefined` e o Skeleton aparece — verde sobre a tela quebrada.
   *
   * A resposta da ficha fica SEGURA (não só atrasada) para que a asserção seja
   * sobre um estado e não sobre uma corrida: enquanto o teste não soltar, a
   * partida tem de continuar na tela.
   */
  test('a partida não some da tela enquanto a ficha do jogador carrega', async ({ page }) => {
    await openSection(page, 'sessoes')

    let release = (): void => {}
    const held = new Promise<void>((resolve) => {
      release = resolve
    })
    let requested = (): void => {}
    const inFlight = new Promise<void>((resolve) => {
      requested = resolve
    })
    // O personagem 13 é o do jogador na mesa 1 (seed). A rota do /sheet não
    // casa: o glob termina no id.
    await page.route('**/api/characters/13', async (route) => {
      requested()
      await held
      await route.continue()
    })

    await page.getByRole('button', { name: /Continuar a sessão/ }).click()

    // Só olhar a tela DEPOIS que a ficha entrou em voo. Sem isto o teste
    // apanha a janela em que a partida ainda está pintada — antes de os
    // membros chegarem e a query do personagem começar — e passa por sorte.
    await inFlight

    // Um marco do shell da partida e um do bloco do jogador — o snapshot da
    // falha original tinha SÓ a região de notificações, nada mais.
    // "Ao vivo" sozinho é ambíguo: o rail da sessão também carrega o selo.
    await expect(page.getByRole('link', { name: 'Sair da sessão' })).toBeVisible()
    await expect(page.getByText(/· Sessão \d+/)).toBeVisible()

    release()
    await expect(page.getByRole('tab', { name: 'Mochila' })).toBeVisible()
  })
})

/**
 * O HUD do jogador DENTRO da sessão (ALE-127).
 *
 * O dono mandou um print em que as caixas de estatística não fechavam a mesma
 * linha de base e "ATQ DIST" quebrava em duas linhas enquanto as vizinhas não —
 * o HUD estava espremido no rail de 22rem que a sessão dava a ele. A ALE-129
 * tirou o rail e deu a tela inteira à ficha, e com isso o desalinhamento sumiu.
 *
 * Este teste existe para ele não VOLTAR, e é da classe de asserção que faltava
 * na suíte: RELAÇÃO entre caixas irmãs. "A página não rola" nunca veria isto.
 */
test('as caixas do HUD fecham a mesma linha em todo formato', async ({ page }) => {
  await page.goto('/campaigns/1/sessions/4')
  await expect(page.getByRole('button', { name: /Minha ficha/ })).toBeVisible()

  for (const viewport of VIEWPORTS) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height })
    const medida = await page.evaluate(() => {
      // As caixas trazem rótulo e número colados: "Defesa16", "Atq Dist+11".
      const padrao = /^(Defesa|Atq CaC|Atq Dist|Fort|Refl|Vont)[+\-−]?\d/
      const caixas = [...document.querySelectorAll('button')]
        .filter((b) => padrao.test((b.textContent ?? '').trim()))
        .map((b) => ({
          alto: Math.round(b.getBoundingClientRect().height),
          rotulo: Math.round(b.querySelector('span')?.getBoundingClientRect().height ?? 0),
        }))
        .filter((c) => c.alto > 0) // no telefone o HUD largo dá lugar à seção Vitais
      if (caixas.length === 0) return null
      const alturas = caixas.map((c) => c.alto)
      const rotulos = caixas.map((c) => c.rotulo)
      return {
        desvioAltura: Math.max(...alturas) - Math.min(...alturas),
        rotuloQuebrou: Math.max(...rotulos) > Math.min(...rotulos),
      }
    })
    if (medida === null) continue
    expect(medida.desvioAltura, `${viewport.name}: as caixas do HUD têm alturas diferentes`).toBe(0)
    expect(medida.rotuloQuebrou, `${viewport.name}: um rótulo quebrou e os outros não`).toBe(false)
  }
})
