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
 * servidor e está em `api/authz_http_test.go`, e a metade de UX mudou de casa
 * na ALE-255: com a crônica virando cena do servidor, quem prova que o jogador
 * pedindo `?tab=config` cai na visão geral é o
 * `TestJogadorPedindoConfigCaiParaAVisaoGeral`, em `api/piloto_cronica_test.go`.
 * Botão ausente nunca foi prova de trava, e a regra da casa é que cada garantia
 * fique na camada mais barata que a sustenta — que aqui ficou mais barata
 * ainda, porque a aba deixou de existir no servidor em vez de ser escondida.
 *
 * O que sobra aqui é o que só o browser vê: o jogador ENTRA na sessão ao vivo.
 */
test.use({ storageState: '.auth/player.json' })

const CAMPAIGN = '/piloto/campanhas/1' // do mestre; o jogador é só membro

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
  // O jogador PERDE ações de escrita, não a mesa: ele continua lendo a campanha
  // e entrando na sessão ao vivo.
  // 'o jogador ainda lê a campanha e entra na sessão ao vivo' saiu na ALE-187:
  // eram três `toBeVisible` de texto, nada que o jsdom não veja. O vizinho
  // abaixo FICA, e a diferença entre os dois é exatamente o critério da casa.

  /**
   * O TESTE DA ALE-96 SAIU DAQUI, e é a única forma honesta de fechá-lo.
   *
   * Ele media que a partida não desaparecia enquanto a ficha do jogador estava
   * em voo: ler `.data` de uma query PENDENTE suspende, e o boundary mais
   * próximo é o `Suspense` que o solid-router põe em todo route match — então a
   * cena inteira era desanexada por uma requisição que nem era dela.
   *
   * A crônica virou página do SERVIDOR na ALE-255. Não há route match, não há
   * Suspense, e não há ficha em voo nesta página: os dados chegam desenhados.
   * O teste morreu com o mecanismo que media, e mantê-lo apontado para a cena
   * nova o transformaria num teste que passa por não ter o que medir — que é
   * pior que teste nenhum, porque parece cobertura.
   *
   * A técnica dele era boa e o histórico a guarda
   * (`git show a94ddc5:e2e/tests/campaign-player.spec.ts`): a resposta ficava
   * SEGURA em vez de só atrasada, para a asserção ser sobre um estado e não
   * sobre uma corrida.
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

    // LINK e não `button`: `/campaigns/$id` encaminha para a crônica do piloto
    // desde a ALE-255, e lá a afordância é um `<a href>` para a sessão. O papel
    // errado não falha explicando — ele espera 30s por um botão que não existe
    // e morre em timeout, com cara de tela quebrada.
    //
    // O que este guarda mede continua na SPA: a sessão ao vivo
    // (`/campaigns/$id/sessions/$sid`) é a última tela não portada, e é dela a
    // garantia de que a partida não pisca enquanto a ficha carrega.
    await page.getByRole('link', { name: /Continuar a sessão/ }).click()

    // Só olhar a tela DEPOIS que a ficha entrou em voo. Sem isto o teste
    // apanha a janela em que a partida ainda está pintada — antes de os
    // membros chegarem e a query do personagem começar — e passa por sorte.
    await inFlight

    // Um marco do shell da partida e um do bloco do jogador — o snapshot da
    // falha original tinha SÓ a região de notificações, nada mais.
    // "Ao vivo" sozinho é ambíguo: o rail da sessão também carrega o selo.
    await expect(page.getByRole('link', { name: 'Sair da sessão' })).toBeVisible()
    // O TÍTULO do cabeçalho, e não mais o `· Sessão N` do estado ao vivo: a
    // ALE-201 apagou aquela cópia de propósito — o título ao lado já dizia o
    // mesmo, e a repetição estourava o cabeçalho a 390px. A marca continua
    // sendo "o shell da partida está pintado", que é o que este teste mede;
    // só mudou qual elemento a carrega.
    await expect(page.getByText(/^Sessão \d+ ·/)).toBeVisible()

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
