import { expect, test } from '@playwright/test'

/**
 * O ÚNICO e2e do Hub, e o que ele guarda é a PARTIDA: chegar em `/` sem dizer
 * mais nada e cair autenticado no Hub, que é uma página do servidor alcançada
 * por desvio no mux — nenhuma outra camada atravessa esse caminho inteiro.
 *
 * O clique no menu ficou junto porque ele é uma navegação de VERDADE entre duas
 * páginas do servidor. O bloco `O Hub`, mais abaixo, afirma os destinos do menu
 * pelas setas; aqui se afirma que o clique chega lá e que a cena desenhou.
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
   * O menu só declara a forma dele (`data-nav-region`, `data-nav-layout`) e o
   * `scene-nav.ts` lê isso do DOM — nenhuma linha de adaptação por cena.
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
   * O popover do rodapé é a Popover API NATIVA: camada de topo, `Esc` fecha, e o
   * foco VOLTA para o gatilho.
   *
   * O `Esc` é o guarda que importa. O driver de teclado escuta na CAPTURA, e com
   * isso ele pre-empta o NAVEGADOR: o `Esc` vira "voltar um nível" antes de
   * chegar à dispensa nativa, e o popover não fecha. Quem o impede é a lista de
   * "há camada aberta?" do driver conhecer `:popover-open` e `dialog[open]`.
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
   * Som e volume são a MESMA preferência em toda cena: quem liga o som no Hub
   * não pode achá-lo desligado no tabuleiro. O contrato é a chave e a FORMA do
   * `localStorage` (`t20-ui` → `{state:{sfx,volume}}`), escrita pelo `persistUi`
   * do `ui-store`.
   *
   * É este contrato que se afirma — não o rótulo do botão. O rótulo é
   * consequência; a chave é o que uma refatoração distraída quebraria sem que
   * nada mais reclamasse.
   */
  test('ligar o som no Hub grava na chave que as cenas leem', async ({ page }) => {
    // Limpa ANTES de a página carregar: depois do `goto` o `data-init` já leu a
    // preferência antiga para o sinal, e o rótulo nasce "Som ligado" se outro
    // teste tiver ligado. O `addInitScript` roda antes de qualquer script da
    // página, que é o único momento em que limpar significa alguma coisa.
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
