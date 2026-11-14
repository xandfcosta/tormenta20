import { expect, test } from '@playwright/test'
import { VIEWPORTS, expectNoHorizontalOverflow } from './support/viewports'
import { tidyUp } from './support/tidy'

const CAMPAIGN = '/campanhas/1' // Snapshot Test ALE-33 (seed)

test.describe('Detalhe da campanha', () => {
  // 'troca de aba mostra o roster de membros' saiu daqui de propósito: clique
  // numa aba e nomes na tela não pedem browser nenhum. A garantia está partida
  // em duas, cada metade na camada que a sustenta:
  //   - a SEÇÃO É ENDEREÇO (link, URL, e o botão voltar) tem e2e próprio no
  //     bloco `A crônica`, no fim deste arquivo — histórico é do navegador e
  //     jsdom não o tem;
  //   - o ROSTER desenhado é `TestTheGmComesFirstInTheCast`, em
  //     `api/campaigns_one_test.go`, que é a camada mais barata que o sustenta.
})

/**
 * Um teste por cena, com os seis formatos DENTRO dele — e não um teste por par.
 * O leiaute responde ao `setViewportSize` na hora (media query é só de largura,
 * regra da casa), então pagar uma carga de página por formato não comprava
 * nada. Ver `support/viewports.ts` para o que isto prova e o que não prova.
 */
const SCENES = [
  { name: 'detalhe', path: `${CAMPAIGN}?tab=membros`, heading: /Snapshot Test ALE-33/i },
  { name: 'visão', path: `${CAMPAIGN}?tab=visao`, heading: /Snapshot Test ALE-33/i },
  { name: 'nova', path: '/campanhas/nova', heading: /Nova campanha/i },
  { name: 'convite', path: '/campanhas/entrar', heading: /Entrar numa mesa/i },
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
   * Apertar o respiro por `max-height` casa DUAS situações diferentes: o
   * celular deitado, que é quem a consulta quer atender, e o celular EM PÉ com
   * o teclado virtual aberto (390x844 vira ~390x494). Estas folhas hospedam
   * campo de texto, então o respiro encolheria debaixo do dedo no meio da
   * digitação. É por isso que a regra da casa manda chavear por LARGURA.
   *
   * O teste afirma as DUAS metades de propósito: só a primeira passaria verde
   * com a tampa simplesmente APAGADA, que é o conserto errado. Medido no
   * deitado, com a tampa aparecem 89% do botão e sem ela apenas 31% — o limiar
   * de 0,8 fica com folga dos dois lados desse vão. `ratio: 1` seria exigir
   * zero pixel de sobra, que é asserção que depende da fonte instalada.
   *
   * Só e2e: em jsdom não há viewport e nenhuma media query resolve.
   */
  test('o respiro da folha não muda quando o teclado abre, e o deitado continua cabendo', async ({
    page,
  }) => {
    await page.goto('/campanhas/nova')
    await expect(page.getByRole('heading', { name: /Nova campanha/i })).toBeVisible()

    // A testemunha é o RESPIRO da folha, e ele se lê no valor computado e não
    // na posição de um filho. Tentei os dois caminhos posicionais e os dois
    // medem outra coisa: no vertical entram as margens automáticas que centram
    // o formulário (173px numa janela alta contra 34 numa baixa, com o mesmo
    // respiro nas duas), e no horizontal entra 1px de arredondamento de barra
    // de rolagem — ruído da ordem do sinal, que era de 24px para 16.
    const breathingRoom = () =>
      page
        .locator('.ds-camps-form')
        .evaluate(
          (el) =>
            `${getComputedStyle(el).paddingLeft}/${getComputedStyle(el.firstElementChild!).rowGap}`,
        )

    await page.setViewportSize({ width: 390, height: 844 })
    const noKeyboard = await breathingRoom()

    // O teclado virtual do celular não muda a largura, só a altura.
    await page.setViewportSize({ width: 390, height: 494 })
    const withKeyboard = await breathingRoom()

    expect(
      withKeyboard,
      'o respiro da folha encolheu quando o teclado abriu — ela está chaveando por ALTURA',
    ).toBe(noKeyboard)

    await page.setViewportSize({ width: 844, height: 390 })
    await expect(
      page.getByRole('button', { name: /Criar campanha/i }),
      'no celular deitado o botão que fecha a tarefa saiu da tela',
    ).toBeInViewport({ ratio: 0.8 })
  })
})

test.describe('A folha em branco', () => {
  test.use({ storageState: '.auth/user.json' })

  // Tela nova se valida nos seis formatos, e aqui importa mais que de costume:
  // a folha hospeda campos de texto, e num telefone deitado o botão de enviar
  // é o primeiro a cair para fora da tela.
  test('a folha cabe nos seis formatos', async ({ page }) => {
    await page.goto('/campanhas/nova')
    await expect(page.getByRole('button', { name: 'Criar campanha' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

  /**
   * O LIMITE AVISA ENQUANTO SE ESCREVE: o `maxlength` NATIVO para a digitação no
   * teto, em vez de deixar escrever 3000 caracteres e recusar no fim. É o que só
   * o navegador faz; que a recusa do servidor devolve o texto digitado é do
   * `TestTheRefusalGivesBackWhatWasTyped`.
   */
  test('o limite da descrição para a digitação no teto', async ({ page }) => {
    await page.goto('/campanhas/nova')
    const description = page.getByLabel('Descrição')

    await description.fill('x'.repeat(2500))
    expect((await description.inputValue()).length, 'o navegador deixou passar do teto').toBe(2000)
  })

})

test.describe('A crônica', () => {
  test.use({ storageState: '.auth/user.json' })

  test('a crônica cabe nos seis formatos', async ({ page }) => {
    await page.goto('/campanhas/1')
    // UM `h1` só, e é o nome da campanha: a casca só desenha o dela quando há
    // título, e a crônica não passa um — ela tem o próprio.
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
  })

  /**
   * A ABA É ENDEREÇO, e é essa a decisão que a cena inteira apoia: o `?tab=`
   * chega com o pedido, e o que se afirma é a consequência disso para quem usa
   * — o link é colável e o histórico funciona.
   *
   * E2E porque histórico é do navegador: `goBack` não existe em jsdom.
   */
  test('a seção é endereço: ela sobrevive ao link colado e ao botão voltar', async ({ page }) => {
    await page.goto('/campanhas/1?tab=membros')
    await expect(page.locator('[aria-current="page"]')).toHaveText('Membros')

    await page.getByRole('link', { name: 'Sessões', exact: true }).click()
    await expect(page.locator('[aria-current="page"]')).toHaveText('Sessões')

    await page.goBack()
    await expect(page.locator('[aria-current="page"]'), 'o voltar não devolveu a seção').toHaveText(
      'Membros',
    )
  })

  /**
   * O INTERRUPTOR das regras é a única ação da crônica que NÃO navega, e por
   * isso é a única com Datastar: alternar um ajuste no meio de uma lista e
   * recarregar a página perderia a posição de quem está lendo.
   *
   * E2E porque o remendo é SSE trocando um pedaço do DOM — o guarda em Go
   * prova o estado do banco, e este prova que a tela acompanhou sem recarregar.
   *
   * Ele DEVOLVE o interruptor ao estado original, num `finally`: a regra é da
   * semente, e deixá-la trocada mudaria a carga de todo personagem da campanha 1
   * para o próximo teste.
   */
  test('alternar a regra opcional troca o estado sem recarregar a página', async ({ page }) => {
    await page.goto('/campanhas/1?tab=regras')
    const key = page.getByRole('switch', { name: 'Limites de carga' })
    const before = await key.isChecked()

    // A navegação NÃO pode acontecer: se acontecesse, este marcador sumiria.
    await page.evaluate(() => {
      ;(window as unknown as { __mesmaPagina: boolean }).__mesmaPagina = true
    })

    try {
      await key.click()
      await expect(key, 'o interruptor não trocou de estado').toBeChecked({ checked: !before })
      expect(
        await page.evaluate(() => (window as unknown as { __mesmaPagina?: boolean }).__mesmaPagina),
        'a página recarregou — o remendo virou navegação',
      ).toBe(true)
    } finally {
      await tidyUp('os Limites de carga da campanha 1 trocados', async () => {
        if ((await key.isChecked()) === before) return
        await key.click()
        await expect(key).toBeChecked({ checked: before, timeout: 5000 })
      })
    }
  })

})
