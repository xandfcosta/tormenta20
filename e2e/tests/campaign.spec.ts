import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'

const CAMPAIGN = '/piloto/campanhas/1' // Snapshot Test ALE-33 (seed)

test.describe('Detalhe da campanha', () => {
  // 'troca de aba mostra o roster de membros' saiu na ALE-187: clique numa aba
  // e nomes na tela, sem medida que precise de browser. A garantia — e a URL,
  // que o e2e também afirmava — está em `campaign-detail-page.test.tsx`, que
  // monta a página com router de memória e cobre os dois sentidos.
})

/**
 * Criar e excluir uma campanha (ALE-79, ALE-80): o único caso daqui que ESCREVE
 * de verdade, e por isso ele apaga o que criou — a seed é compartilhada com
 * todos os specs e uma execução que deixa campanha para trás envenena a
 * próxima.
 *
 * O bloco "Entrar por convite" saiu na ALE-144: resolução do alvo
 * (`entities/campaign/join-target.test.ts`), prévia do convite
 * (`entities/queries.test.ts`) e convite morto
 * (`features/campaign-join/hero-picker.test.tsx`) já respondem em vitest, e a
 * página `/campaigns/join` continua sendo carregada nos seis formatos pelo
 * bloco responsivo abaixo.
 */
test.describe('Abrir e fechar uma campanha', () => {
  test('criar leva direto para a nova campanha, e excluir traz de volta', async ({
    page,
  }) => {
    const name = `E2E Descartável ${Date.now()}`
    await page.goto('/campaigns/new')

    await page.getByLabel('Nome').fill(name)
    await page.getByLabel('Descrição').fill('Criada e excluída pelo E2E.')
    await page.getByRole('button', { name: 'Abrir campanha' }).click()

    // Landed on the new chronicle's own page — a do SERVIDOR desde a ALE-255.
    await expect(page).toHaveURL(/\/piloto\/campanhas\/\d+/)
    await expect(page.getByRole('heading', { name, level: 1 })).toBeVisible()

    // Clean up through the UI, which also exercises the ALE-79 delete path.
    await page.goto(`${new URL(page.url()).pathname}?tab=config`)
    await page.getByRole('button', { name: /Excluir campanha/ }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Excluir' }).click()
    // Volta para a LISTA, e o que se afirma é a lista — não a URL dela. Desde a
    // ALE-234 a cena é do servidor e `/campaigns` encaminha para
    // `/piloto/campanhas`; prender o teste ao endereço faria ele quebrar de novo
    // quando o prefixo `/piloto` cair, sem que nada de verdade tivesse mudado.
    await expect(page.getByRole('listbox', { name: 'Campanhas' })).toBeVisible()
    await expect(page.getByRole('option', { name: new RegExp(name) })).toHaveCount(0)
  })
})

/**
 * One test per scene, six viewports inside each — not one test per pair. The
 * layout answers `setViewportSize` live (media queries are width-only, a house
 * rule), so paying a full page load per viewport bought nothing: this block was
 * 18 tests and 134s. See `support/viewports.ts` for what it does and does not
 * prove.
 */
const SCENES = [
  { name: 'detalhe', path: `${CAMPAIGN}?tab=membros`, heading: /Snapshot Test ALE-33/i },
  { name: 'nova', path: '/piloto/campanhas/nova', heading: /Abrir nova campanha/i },
  { name: 'convite', path: '/piloto/campanhas/entrar', heading: /Entrar na mesa/i },
]

test.describe('Campanha — responsivo (sem overflow horizontal)', () => {
  for (const scene of SCENES) {
    test(`${scene.name}: sem scroll horizontal nos seis formatos`, async ({ page }) => {
      await page.goto(scene.path)
      await expect(page.getByRole('heading', { name: scene.heading })).toBeVisible()

      await expectNoHorizontalOverflow(page, VIEWPORTS)
    })
  }

  /**
   * O defeito da ALE-160: a 390px os painéis da visão geral eram pintados 169px
   * fora do pai, e o botão "Convite" ia parar em x 392–487 numa tela de 390 —
   * fora da janela e sem eixo que rolasse até ele. A causa é `min-width: auto`
   * em item de grid, que dimensiona a trilha pelo MIN-CONTENT: 457px numa caixa
   * de 288.
   *
   * O `expectNoHorizontalOverflow` acima passava VERDE sobre isso, e não por
   * descuido: o `overflow-x-hidden` da cena zera o `scrollWidth` da raiz. É
   * preciso medir contra a JANELA, e é o que a asserção nova faz.
   */
  test('nada clicável fica fora da janela na campanha, em nenhum formato', async ({ page }) => {
    await page.goto(`${CAMPAIGN}?tab=visao`)
    await expect(page.getByRole('heading', { name: /Snapshot Test ALE-33/i })).toBeVisible()

    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * ALE-176. A folha do grimório apertava o respiro por `max-height: 520px`, e
   * essa consulta casa DUAS situações diferentes: o celular deitado, que é
   * quem ela queria atender, e o celular EM PÉ com o teclado virtual aberto
   * (390x844 vira ~390x494). Estas folhas hospedam campo de texto — "nova
   * campanha" e o convite —, então o respiro encolhia debaixo do dedo no meio
   * da digitação. É por isso que a regra da casa manda chavear por LARGURA.
   *
   * O teste afirma as DUAS metades de propósito. Só a primeira passaria verde
   * com a tampa simplesmente APAGADA, que é o conserto errado: medido no
   * deitado, com a tampa aparecem 89% do botão "Abrir campanha" e sem ela
   * apenas 31% (y=379,6..412,7 numa janela de 390). O limiar de 0,8 fica com
   * folga dos dois lados desse vão, e não é número mágico: está aqui porque a
   * tampa não faz o botão CABER — ela o traz de quase escondido para quase
   * inteiro, e prender `ratio: 1` seria exigir zero pixel de sobra, que é o
   * tipo de asserção que a ALE-184 mostrou depender da fonte instalada.
   *
   * Só e2e: em jsdom não há viewport e nenhuma media query resolve.
   */
  test('o respiro da folha não muda quando o teclado abre, e o deitado continua cabendo', async ({
    page,
  }) => {
    await page.goto('/campaigns/new')
    await expect(page.getByRole('heading', { name: /Abrir nova campanha/i })).toBeVisible()

    // A testemunha é o RESPIRO da folha, e ele se lê no valor computado e não
    // na posição de um filho. Tentei os dois caminhos posicionais e os dois
    // medem outra coisa: no vertical entram as margens automáticas que centram
    // o formulário (173px numa janela alta contra 34 numa baixa, com o mesmo
    // respiro nas duas), e no horizontal entra 1px de arredondamento de barra
    // de rolagem — ruído da ordem do sinal, que era de 24px para 16.
    const respiro = () =>
      page
        .locator('[data-tome-root]')
        .evaluate((el) => `${getComputedStyle(el).paddingLeft}/${getComputedStyle(el).rowGap}`)

    await page.setViewportSize({ width: 390, height: 844 })
    const semTeclado = await respiro()

    // O teclado virtual do celular não muda a largura, só a altura.
    await page.setViewportSize({ width: 390, height: 494 })
    const comTeclado = await respiro()

    expect(
      comTeclado,
      'o respiro da folha encolheu quando o teclado abriu — ela está chaveando por ALTURA',
    ).toBe(semTeclado)

    await page.setViewportSize({ width: 844, height: 390 })
    await expect(
      page.getByRole('button', { name: /Abrir campanha/i }),
      'no celular deitado o botão que fecha a tarefa saiu da tela',
    ).toBeInViewport({ ratio: 0.8 })
  })
})
