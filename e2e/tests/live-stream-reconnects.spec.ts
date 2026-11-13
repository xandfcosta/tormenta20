import { expect, test } from '@playwright/test'

/**
 * O FLUXO DA MESA SEMPRE VOLTA, e em cerca de um segundo (ALE-636).
 *
 * Desde a 1.0.5, o Datastar conta as novas tentativas pela vida inteira do `@get`, e não mais desde a
 * última conexão aceita: a espera dobra a cada queda até 30 s, e depois da décima o fluxo desiste até
 * alguém recarregar a página. Uma sessão de horas tem quedas de sobra — o deploy no meio do jogo derruba
 * o fluxo de propósito, e o Wi-Fi do celular pisca. Quem abre um fluxo longo passa pelo
 * `ui.LiveStream`, que tira o teto e o recuo.
 *
 * E2E porque a reconexão é do runtime do navegador; o Go só vê o texto do `@get`. As quedas são as
 * tentativas que a rota ABORTA, e cada uma conta como falha de rede, como o servidor fora do ar num
 * deploy. Não dá para derrubar uma conexão já aberta daqui (o `setOffline` do Chromium não corta um
 * fluxo em curso), então as doze primeiras tentativas caem, a inicial inclusive. Com o padrão da 1.0.5,
 * a décima primeira nunca sairia, e as dez primeiras já levariam minutos.
 */

const MESA = '/campanhas/1/sessoes/5'
const QUEDAS = 12

test('o fluxo da mesa volta depois de doze quedas seguidas, em cerca de um segundo cada', async ({ page }) => {
  let attempts = 0
  await page.route(/\/fluxo(\?|$)/, async (route) => {
    attempts++
    if (attempts <= QUEDAS) {
      await route.abort('connectionreset')
      return
    }
    await route.continue()
  })
  await page.goto(MESA)
  await expect
    .poll(() => attempts, { timeout: 20_000, message: `o fluxo não chegou à tentativa ${QUEDAS + 1} em 20 s` })
    .toBeGreaterThan(QUEDAS)
})
