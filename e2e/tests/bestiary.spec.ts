import { expect, test } from '@playwright/test'
import { expectDentroDaJanela } from './support/geometry'
import { expectNoHorizontalOverflow, VIEWPORTS } from './support/viewports'

test.describe('O bestiário', () => {
  test.use({ storageState: '.auth/user.json' })

  const BESTIARIO = '/mestre/bestiario'

  test('o bestiário cabe nos seis formatos', async ({ page }) => {
    await page.goto(BESTIARIO)
    await expect(page.getByRole('heading', { name: 'Bestiário' })).toBeVisible()

    await expectNoHorizontalOverflow(page, VIEWPORTS)
    for (const viewport of VIEWPORTS) {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await expectDentroDaJanela(page)
    }
  })

  /**
   * A METADE ESTREITA, e é o único teste desta cena que precisa mesmo de browser.
   *
   * O `.mesa-painel` só existe a partir de 50rem de CONTÊINER — consulta de
   * contêiner, não de mídia, e nem o jsdom nem uma asserção de classe sabem
   * resolver isso. Abaixo do ponto de troca a ficha tem de estar no diálogo, e
   * acima dela tem de estar no painel; a faixa em que as duas somem, ou em que
   * as duas aparecem, é o defeito que este teste existe para pegar.
   *
   * Ele mede VISIBILIDADE REAL (`toBeVisible`), que é o que resolve a cascata
   * inteira — a consulta de contêiner, o `display:none` da folha e o
   * `data-show` do Datastar decidindo juntos.
   */
  test('a ficha vive no painel quando cabe, e no diálogo quando não cabe', async ({ page }) => {
    await page.goto(BESTIARIO)
    const painel = page.getByRole('region', { name: 'Criatura escolhida' })
    const dialogo = page.getByRole('dialog')

    await page.setViewportSize({ width: 1440, height: 900 })
    await expect(painel, 'o painel sumiu numa largura que comporta duas colunas').toBeVisible()
    await expect(dialogo, 'o diálogo apareceu por cima do painel').toBeHidden()

    // No telefone o painel não cabe: a ficha só é alcançável pelo diálogo, e é
    // ele que impede a lista de virar uma lista sem detalhe nenhum.
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(painel, 'o painel ficou visível onde não cabe').toBeHidden()
    await expect(dialogo, 'o diálogo abriu sozinho').toBeHidden()

    await page.getByRole('listitem').first().getByRole('link').click()
    await expect(dialogo, 'tocar na linha não abriu a ficha no telefone').toBeVisible()

    await page.keyboard.press('Escape')
    await expect(dialogo, 'o Esc não fechou a ficha').toBeHidden()
  })

  /**
   * A CORRIDA ENTRE OS DOIS PEDIDOS DE UM CLIQUE SÓ (ALE-272).
   *
   * O clique do mouse também FOCA, e por uma fatia inteira a linha saía com dois
   * pedidos: o do foco, que só pré-visualiza e não leva `abrir=1`, e o do
   * clique, que leva. Os dois remendam o `#bestiary`, que redeclara
   * `fichaAberta` a cada remendo — então quem CHEGA por último manda, e a ordem
   * de chegada não é a de saída. Na bancada o do clique chegava depois e a ficha
   * abria; no CI a ordem inverteu e o teste acima pegou a criatura escolhida com
   * a ficha fechada, duas vezes seguidas.
   *
   * O teste acima NÃO segura isto: ele passa por sorte de cronometragem, que é o
   * que o deixou verde aqui enquanto o CI ficava vermelho. Este aqui INVERTE a
   * ordem de propósito — atrasa a resposta do pedido sem `abrir` — e aí a
   * garantia deixa de depender de quem é mais rápido.
   *
   * Browser é a única testemunha possível: são dois `fetch` em voo disparados
   * pelo MESMO gesto de ponteiro, e é o `:focus-visible` do navegador que separa
   * o foco do mouse do foco da seta.
   */
  test('a ficha abre no clique mesmo se a resposta do foco chegar depois', async ({ page }) => {
    await page.goto(BESTIARIO)
    await page.setViewportSize({ width: 390, height: 844 })
    await expect(page.getByRole('listitem').first()).toBeVisible()

    // Só o pedido SEM `abrir=1` é atrasado: é o do foco, e é ele que chegaria
    // por último para redeclarar `fichaAberta: false` por cima do clique.
    await page.route('**/mestre/bestiario?*', async (route) => {
      if (new URL(route.request().url()).searchParams.has('abrir')) {
        await route.continue()
        return
      }
      await new Promise((resolve) => setTimeout(resolve, 600))
      await route.continue()
    })

    await page.getByRole('listitem').first().getByRole('link').click()

    // O ESTADO ASSENTADO, e não o primeiro quadro — e esta espera é a diferença
    // entre um teste e um teste que mente. Medido: com o defeito no lugar a
    // ficha ABRE com a resposta do clique e só fecha 600ms depois, quando a do
    // foco chega. Um `toBeVisible` cru passa dentro dessa janela, e a primeira
    // versão deste teste nasceu VERDE sobre o defeito que ela existia para
    // pegar. A janela é de 600ms porque é ESTE teste que a injeta acima, então
    // esperar mais que ela é determinístico e não um palpite de cronometragem.
    await page.waitForTimeout(1500)

    // O CLIQUE ATERRISSOU: sem isto, um seletor que um dia pare de achar a
    // linha faria o teste passar sem nunca ter clicado em nada, que é a forma
    // desta família — ausência de estímulo com cara de conserto. Com o defeito
    // no lugar esta asserção passa e a de baixo falha, e é esse par que
    // distingue "a ficha não abriu" de "o gesto não aconteceu".
    await expect(
      page.getByRole('listitem').first().getByRole('link'),
      'o clique não escolheu a criatura: o gesto não aconteceu',
    ).toHaveAttribute('aria-current', 'true')

    await expect(
      page.getByRole('dialog'),
      'a resposta do foco chegou por último e fechou a ficha que o clique abriu',
    ).toBeVisible()
  })
})
