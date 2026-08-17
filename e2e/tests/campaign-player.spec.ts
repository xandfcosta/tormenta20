import { type Page, expect, test } from '@playwright/test'
import { VIEWPORTS } from './support/viewports'

/**
 * A cena da campanha vista por um JOGADOR, não pelo mestre (ALE-24). Todo o
 * resto da suíte roda como o mestre da seed pelo storageState compartilhado;
 * este spec sai disso e entra como o jogador que é membro da campanha 1 e não
 * é dono de nada nela.
 *
 * As três asserções de AUSÊNCIA que moravam aqui (rail sem Config, nenhuma ação
 * de dono, `?tab=config` na mão) saíram na ALE-144: a trava de verdade é do
 * servidor e está em `api/authz_http_test.go`, e a metade de UX está em
 * `pages/campaigns/campaign-detail-page.test.tsx` ("o jogador com ?tab=config
 * cai na Visão"). Botão ausente nunca foi prova de trava, e a regra da casa é
 * que cada garantia fique na camada mais barata que a sustenta.
 *
 * O que sobra aqui é o que só o browser vê: o jogador ENTRA na sessão ao vivo,
 * e a cena não some enquanto a ficha dele carrega (ALE-96).
 */
test.use({ storageState: '.auth/player.json' })

const CAMPAIGN = '/campaigns/1' // do mestre; o jogador é só membro

/**
 * Opens a section and waits for the tome to actually be on screen.
 *
 * `networkidle` because in dev the first cold visit to a route makes
 * Vite re-optimize deps and force a reload, which strands the default wait on
 * the discarded document.
 */
async function openSection(page: Page, tab: string): Promise<void> {
  await page.goto(`${CAMPAIGN}?tab=${tab}`, { waitUntil: 'networkidle' })
  await expect(page.getByRole('heading', { name: /Snapshot Test ALE-33/i })).toBeVisible()
}

test.describe('Campanha vista pelo jogador', () => {
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
