import { expect, test } from '@playwright/test'

test.describe('Administração', () => {
  // A tela é do ADMIN, então o estado de login é o do mestre — o `requireAdmin`
  // responde 403 para o jogador, e é o servidor que decide, não a tela.
  test.use({ storageState: '.auth/user.json' })

  /**
   * O LINK DE UMA PESSOA NÃO PODE APARECER SOB O NOME DE OUTRA.
   *
   * O token chega por remendo do servidor num `<div>` fixo (`#reset-link`), e o
   * diálogo é UM só reaproveitado por todas as linhas. Sem limpar ao abrir,
   * gerar o link da primeira conta, fechar, e abrir a caixa da segunda mostra o
   * link da PRIMEIRA sob o nome da SEGUNDA — e quem estiver com pressa entrega
   * a chave da conta errada.
   *
   * E2E porque a garantia é de ESTADO DE DOM ATRAVESSANDO duas aberturas de um
   * `<dialog>` nativo: em jsdom não há `showModal`, e o guarda em Go só
   * consegue afirmar que a limpeza está escrita no marcador, não que ela
   * acontece.
   *
   * Ele grava uma linha em `password_resets`, e isso é aceitável: nenhuma tela
   * a lista e nenhuma asserção a conta. Cunhar CONVITE seria diferente — aquilo
   * aparece num painel e a tela não sabe revogar —, e por isso aquela garantia
   * ficou em Go.
   */
  test('o link de redefinição não vaza para a caixa do jogador seguinte', async ({ page }) => {
    await page.goto('/admin')
    const triggers = page.getByRole('button', { name: /^Redefinir a senha de/ })
    await expect(triggers.first()).toBeVisible()

    await triggers.first().click()
    await page.getByRole('button', { name: 'Gerar link' }).click()
    const field = page.locator('#reset-url')
    await expect(field).toBeVisible()
    const first = await field.inputValue()
    expect(first, 'o link nasceu sem token').toContain('token=')
    // A origem é a do NAVEGADOR, e não a do servidor: com o `r.Host` o link
    // nasceria apontando para a porta errada atrás de qualquer intermediário.
    expect(first).toContain(new URL(page.url()).origin)

    await page.getByRole('button', { name: 'Fechar' }).click()
    await triggers.nth(1).click()

    await expect(page.locator('dialog#reset')).toBeVisible()
    await expect(field, 'o link do primeiro jogador sobreviveu na caixa do segundo').toHaveCount(0)
  })

  /**
   * A confirmação do DESTRUTIVO é um modal de verdade.
   *
   * O guarda afirma as quatro propriedades do `<dialog>` nativo: ele é `:modal`
   * (o resto da página fica inerte), o foco entra nele, ele tem nome acessível,
   * e ao fechar o foco VOLTA para o gatilho.
   *
   * E2E porque nada disto existe em jsdom: não há `showModal`, não há `:modal`,
   * e o foco é uma ficção.
   */
  test('o diálogo de apagar conta é modal, nomeado, e devolve o foco', async ({ page }) => {
    await page.goto('/admin')
    const trigger = page.getByRole('button', { name: /^Apagar a conta de/ }).first()
    await trigger.focus()
    await trigger.press('Enter')

    const dialog = page.locator('#confirm')
    await expect(dialog).toBeVisible()

    const state = await page.evaluate(() => {
      const d = document.getElementById('confirm') as HTMLDialogElement
      return {
        modal: d.matches(':modal'),
        focoDentro: d.contains(document.activeElement),
        nome: document.getElementById(d.getAttribute('aria-labelledby') ?? '')?.textContent?.trim() ?? '',
      }
    })
    expect(state.modal, 'o fundo precisa ficar inerte').toBe(true)
    expect(state.focoDentro, 'o foco precisa entrar no diálogo').toBe(true)
    expect(state.nome, 'o diálogo precisa de nome acessível').toContain('Apagar a conta de')

    await page.keyboard.press('Escape')
    await expect(dialog).toBeHidden()
    await expect(trigger).toBeFocused()
  })
})
