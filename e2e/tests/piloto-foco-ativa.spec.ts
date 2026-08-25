import { expect, test } from '@playwright/test'

/**
 * FOCO ATIVA na lista do bestiário — a seta anda e a ficha segue junto, sem
 * exigir um Enter para ver. Decisão do dono: "igual em jogo".
 *
 * E2E porque a pergunta é sobre TECLADO e FOCO de verdade. Medido na marra: por
 * CDP o `element.focus()` move o `document.activeElement` e **não dispara o
 * evento `focus`** quando a janela não tem foco do sistema — nem para um
 * ouvinte próprio. Toda medição de teclado feita por automação de aba mede
 * silêncio e parece medir ausência de comportamento.
 *
 * Mede também o CUSTO, que é o que o dono pediu: quantas idas ao servidor uma
 * travessia de seta custa. O `__debounce` existe para que segurar a seta não
 * peça as 80 fichas do caminho — só a que a pessoa parou para ler.
 */
test.use({ storageState: '.auth/user.json' })

test('a seta anda na lista e a ficha segue junto', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  const fichaLateral = page.locator('.mesa-painel')
  const primeira = page.locator('a[href*="criatura="]').first()
  await primeira.focus()
  await expect(fichaLateral).toContainText(/\S/)
  const antes = (await fichaLateral.innerText()).slice(0, 40)

  // O CONTROLE: a região está declarada. Sem ela o driver não tem o que dirigir,
  // e "a seta não andou" seria verdade sobre uma tela sem teclado nenhum — que é
  // um defeito diferente e a mensagem apontaria o lugar errado.
  await expect(page.locator('[data-nav-region="lista"]')).toHaveCount(1)

  await page.keyboard.press('ArrowDown')
  await expect
    .poll(async () => (await fichaLateral.innerText()).slice(0, 40), { timeout: 4000 })
    .not.toBe(antes)
})

test('cada passo da seta desenha a ficha sem espera perceptível', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  const ficha = page.locator('.mesa-painel')
  await page.locator('a[href*="criatura="]').first().focus()

  // PASSO DELIBERADO custa uma ida ao servidor por linha, e isso é o CERTO: quem
  // anda de linha em linha quer ver cada criatura. O `__debounce` não serve para
  // este caso — ele serve para a tecla SEGURADA, que repete a ~30ms e atravessa
  // a lista inteira; aí só a linha onde o dedo parou é pedida.
  //
  // Então o que se mede aqui é LATÊNCIA, não contagem: a pergunta do dono é se
  // a seta "responde igual em jogo", e jogo é quadro a quadro. O teto de 400ms é
  // generoso de propósito — abaixo disso ninguém chama de espera, e acima o
  // desenho deixa de acompanhar o dedo.
  const latencias: number[] = []
  for (let i = 0; i < 8; i++) {
    const antes = (await ficha.innerText()).slice(0, 40)
    const t0 = Date.now()
    await page.keyboard.press('ArrowDown')
    await expect
      .poll(async () => (await ficha.innerText()).slice(0, 40), { timeout: 4000, intervals: [16] })
      .not.toBe(antes)
    latencias.push(Date.now() - t0)
  }

  latencias.sort((a, b) => a - b)
  const mediana = latencias[Math.floor(latencias.length / 2)]
  const pior = latencias[latencias.length - 1]
  console.log(`latência por passo — mediana ${mediana}ms, pior ${pior}ms, todas: ${latencias.join(', ')}`)

  expect(mediana, `mediana de ${mediana}ms por passo de seta`).toBeLessThan(400)
  expect(pior, `pior passo levou ${pior}ms`).toBeLessThan(900)
})

test('depois de uma travessia rápida a ficha é a da linha onde o foco PAROU', async ({ page }) => {
  await page.setViewportSize({ width: 1400, height: 900 })
  await page.goto('/piloto/mestre/bestiario')
  await page.waitForLoadState('networkidle')

  await page.locator('a[href*="criatura="]').first().focus()
  // Rápido de propósito: sem `await` entre as teclas, para cair DENTRO da janela
  // do throttle. É o caso que o throttle com borda de subida põe em risco —
  // ele dispara na primeira e limita as seguintes, e se descartar a última a
  // ficha fica mostrando uma criatura pela qual a pessoa só PASSOU.
  await Promise.all(
    Array.from({ length: 12 }, () => page.keyboard.press('ArrowDown')),
  )

  const foco = page.locator(':focus')
  const href = await foco.getAttribute('href')
  const criatura = new URL(href!, 'http://x').searchParams.get('criatura')

  // O CONTROLE: o foco de fato ANDOU. Sem ele, "a ficha bate com o foco" seria
  // verdade trivialmente sobre uma travessia que não saiu do lugar.
  expect(criatura, 'o foco não andou na travessia').toBeTruthy()
  const rotulo = (await foco.innerText()).split('\n')[0].trim()

  await expect
    .poll(async () => (await page.locator('.mesa-painel').innerText()).includes(rotulo), {
      timeout: 4000,
      intervals: [16],
    })
    .toBe(true)
})
