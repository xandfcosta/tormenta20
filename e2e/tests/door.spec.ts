import { expect, test } from '@playwright/test'

/**
 * NÃO existe aqui um guarda de "o morph preserva o foco", e a ausência é
 * deliberada e medida.
 *
 * Eu escrevi um: o risco parecia óbvio, porque o servidor substitui o `<main>`
 * inteiro a cada mudança e um jogador digitando o d20 no meio do turno perderia
 * o cursor. Ele passou verde, e depois passou verde SABOTADO duas vezes — com o
 * patch em `mode: replace` em vez de morph, e com o servidor renderizando o
 * `value` do campo. Nas duas o foco e o texto digitado sobreviveram.
 *
 * Então o guarda não protegia nada: era um e2e — o teste mais caro do
 * repositório — sem modo de falha demonstrável. O que ficou é o ACHADO, que
 * vale mais que ele: o morph do Datastar reaproveita o nó por identidade, e
 * essa classe de defeito, que eu apostava contra, não acontece.
 */
test.describe('A porta', () => {
  // ANÔNIMA: são estas telas que criam a sessão, e com o estado de login o
  // servidor redireciona para dentro do app — o guarda mediria a tela errada.
  test.use({ storageState: { cookies: [], origins: [] } })

  /**
   * A porta é a tela-título, e a tela-título tem duas coisas que nenhuma outra
   * superfície do app tinha: o brilho do `scene-title-glow` sobre a pedra, e
   * o `text-muted-foreground` do rodapé sobre o fundo mais escuro da cena.
   *
   * Nenhuma delas dá para medir fora do navegador: converter oklch para sRGB é
   * trabalho dele, e em jsdom o `getComputedStyle` devolve o oklch cru.
   */

  /**
   * A senha é conferida sem sinal do Datastar: o `data-on:input` lê o campo
   * irmão pelo DOM e usa `setCustomValidity`.
   *
   * E2E porque `setCustomValidity` e `validationMessage` são do navegador —
   * jsdom aceita a chamada e não faz nada com ela, então lá o guarda passaria
   * verde com a implementação apagada.
   */
  test('a confirmação de senha avisa o typo sem pôr a senha em estado de cliente', async ({
    page,
  }) => {
    await page.goto('/criar-conta?convite=nao-importa')
    await page.locator('#senha').fill('uma senha boa')
    await page.locator('#confirm').fill('outra coisa')

    expect(
      await page.locator('#confirm').evaluate((el: HTMLInputElement) => el.validationMessage),
    ).toBe('As senhas não conferem')

    await page.locator('#confirm').fill('uma senha boa')
    expect(
      await page.locator('#confirm').evaluate((el: HTMLInputElement) => el.checkValidity()),
    ).toBe(true)
  })
})
