import { expect, test } from '@playwright/test'

/**
 * O ÚNICO e2e do Hub, e o que ele guarda hoje é a PARTIDA: chegar em `/` sem
 * dizer mais nada e cair autenticado no Hub, que desde a ALE-231 é uma página
 * do servidor alcançada por desvio no mux — nenhuma outra camada atravessa esse
 * caminho inteiro.
 *
 * O clique no menu ficou junto porque ele é uma navegação de VERDADE entre duas
 * páginas do servidor. O bloco `O Hub`, mais abaixo, afirma os destinos do
 * menu pelas setas; aqui se afirma que o clique chega lá e que a cena
 * desenhou.
 *
 * (A referência antiga a `pages/home/hub.test.tsx` saiu: aquele arquivo foi
 * apagado quando o Hub virou servidor, e a menção sobreviveu ao arquivo.)
 *
 * ELE SOBREVIVEU A UM APAGAMENTO NA `main`, e a nota fica para o próximo merge
 * não o apagar em silêncio: a ALE-187 podou sete e2e "que não precisavam de
 * browser", e este era um deles — quando ele testava o Hub da SPA. Aqui na base
 * ele testa OUTRA COISA: o desvio do mux levando `/` a uma página do SERVIDOR, e
 * uma navegação de verdade entre duas delas. Isso precisa de browser, e nenhuma
 * outra camada atravessa o caminho inteiro.
 *
 * Quando a migração terminar e a `main` receber tudo, vale reler: se o Hub da
 * SPA já não existir, a poda daquela issue perde o alvo e este arquivo é o que
 * resta.
 */
test('o Hub sobe autenticado e o menu leva ao elenco', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /Tormenta 20/i })).toBeVisible()
  await expect(page.getByText('Ferramentas do Mestre')).toBeVisible()

  await page.getByText('Meus Heróis').click()
  await expect(page).toHaveURL(/\/personagens$/)
  await expect(page.getByRole('listbox', { name: 'Personagens' })).toBeVisible()
})

test.describe('O Hub', () => {
  test.use({ storageState: '.auth/user.json' })

  /**
   * A TESE da fatia, medida: o driver de teclado da SPA anda num menu que o
   * SERVIDOR desenhou, sem uma linha de adaptação.
   *
   * O menu só declara a forma dele (`data-nav-region`, `data-nav-layout`) e o
   * `scene-nav.ts` — compilado dos mesmos fontes que a SPA usa — lê isso do
   * DOM. Um DOM vindo do servidor é um DOM.
   *
   * E2E porque foco e geometria são do navegador: em jsdom todo elemento mede
   * zero e o driver não teria como escolher o vizinho.
   */
  test('as setas andam no menu que o servidor desenhou', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('link', { name: 'Meus Heróis' }).focus()

    await page.keyboard.press('ArrowDown')
    await expect(page.locator(':focus')).toHaveAttribute('href', '/campanhas')

    await page.keyboard.press('ArrowUp')
    await expect(page.locator(':focus')).toHaveAttribute('href', '/personagens')
  })

  /**
   * O popover do rodapé é a Popover API NATIVA no lugar do Kobalte, e o que se
   * afirma é o que a biblioteca entregava: camada de topo, `Esc` fecha, e o
   * foco VOLTA para o gatilho.
   *
   * O `Esc` é o guarda que importa. O driver de teclado escuta na CAPTURA para
   * pre-emptar o foco rotativo do Kobalte, e com isso ele também pre-emptava o
   * navegador: media na ALE-231 que o popover não fechava, porque o `Esc` virava
   * "voltar um nível" antes de chegar à dispensa nativa. A lista de "há camada
   * aberta?" do driver era Kobalte-shaped e passou a conhecer `:popover-open` e
   * `dialog[open]`.
   */
  test('o menu do jogador é popover nativo: Esc fecha e devolve o foco', async ({ page }) => {
    await page.goto('/')
    const gatilho = page.getByRole('button', { name: /^Menu de / })
    await gatilho.click()

    const menu = page.locator('#player-menu')
    await expect(menu).toBeVisible()
    expect(await menu.evaluate((el) => el.matches(':popover-open'))).toBe(true)

    await page.keyboard.press('Escape')
    await expect(menu).toBeHidden()
    await expect(gatilho).toBeFocused()
  })
})

test.describe('O Hub e as cenas dividem a preferência de som', () => {
  test.use({ storageState: '.auth/user.json' })

  /**
   * Enquanto as duas portas existirem, som e volume têm de ser a MESMA
   * preferência: quem liga o som no Hub do servidor não pode achá-lo desligado
   * na cena do tabuleiro. O contrato é a chave e a forma do `localStorage`
   * (`t20-ui` → `{state:{sfx,volume}}`), e o `scene.js` escreve pelo mesmo
   * `persistUi` que o `ui-store` da SPA usa.
   *
   * É este contrato que se afirma — não o rótulo do botão. O rótulo é
   * consequência; a chave é o que uma refatoração distraída quebraria sem que
   * nada mais reclamasse.
   */
  test('ligar o som no Hub grava na chave que as cenas leem', async ({ page }) => {
    // Limpa ANTES de a página carregar, e é aí que estava um defeito de ORDEM
    // deste teste: limpando depois do `goto`, o `data-init` já tinha lido a
    // preferência antiga para o sinal, e o rótulo nascia "Som ligado" se algum
    // teste anterior tivesse ligado. O `addInitScript` roda antes de qualquer
    // script da página, que é o único momento em que limpar significa alguma
    // coisa.
    await page.addInitScript(() => localStorage.removeItem('t20-ui'))
    await page.goto('/')

    await page.getByRole('button', { name: /^Menu de / }).click()
    const alternador = page.locator('#player-menu button').first()
    await expect(alternador).toHaveText(/Som desligado/)

    await alternador.click()
    await expect(alternador).toHaveText(/Som ligado/)
    // O slider só existe com o som ligado: controle sobre o mudo é controle morto.
    await expect(page.locator('#volume')).toBeVisible()

    expect(await page.evaluate(() => localStorage.getItem('t20-ui'))).toBe(
      '{"state":{"sfx":true,"volume":100}}',
    )
  })
})
