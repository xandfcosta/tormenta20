import { expect, type Page, test } from '@playwright/test'

/**
 * O ARRASTO DA PEÇA NO RASCUNHO, com DUAS peças de propósito.
 *
 * O defeito que isto prende precisa de duas: com `$dragging` guardando um
 * literal igual para todas, todo `pointerup__window` passa na guarda e o
 * primeiro do DOM vence — pegar qualquer peça move a PRIMEIRA. Com uma peça só,
 * o primeiro do DOM É o arrastado e o gesto certo e o errado dão o mesmo
 * resultado.
 *
 * E2E porque só o navegador tem o gesto: ponteiro com passos intermediários,
 * a ORDEM em que N ouvintes de janela disparam, e a peça desenhada por
 * `transform` durante o arrasto. Nenhum dos três existe fora dele.
 */
test.use({ storageState: '.auth/user.json' })

/** Um lugar do acervo com as peças pedidas, e como se livrar dele. */
async function aDraftWith(
  page: Page,
  tokens: Array<{ nome: string; x: number; y: number }>,
): Promise<{ endereco: string; apagar: () => Promise<void> }> {
  const created = await page.request.post('/api/campanhas', {
    data: { name: `E2E rascunho ${Date.now()}-${Math.floor(Math.random() * 1e6)}`, description: 'ALE-299' },
  })
  expect(created.ok(), `criar a campanha: ${created.status()}`).toBeTruthy()
  const campaign = (await created.json()).id as number

  const nova = await page.request.post(`/campanhas/${campaign}/lugares/novo`, {
    form: { name: 'Cripta do E2E', ground: 'stone' },
    maxRedirects: 0,
  })
  const url = nova.headers().location
  expect(url, `criar o lugar: ${nova.status()}`).toContain('/lugares/')

  for (const p of tokens) {
    const placed = await page.request.post(`${url}/tabuleiro/pecas/nova`, {
      data: {
        from: { X: p.x, Y: p.y },
        new_token_name: p.nome,
        new_token_size: 1,
        new_token_look: 'object',
      },
    })
    expect(placed.ok(), `pôr a peça ${p.nome}: ${placed.status()}`).toBeTruthy()
  }
  await page.goto(url)
  await page.locator('.board-scene').waitFor({ timeout: 10_000 })
  return {
    endereco: url,
    // A LIMPEZA NÃO PODE FALAR MAIS ALTO QUE O DEFEITO: um `finally` que estoura
    // substitui o erro de verdade.
    apagar: async () => {
      try {
        await page.request.delete(`/api/campanhas/${campaign}`)
      } catch {
        // O lugar fica para trás. É o preço certo.
      }
    },
  }
}

/** Onde cada peça está GRAVADA, pela coordenada que o servidor devolveu. */
function whereEachTokenIs(page: Page) {
  return page.evaluate(() =>
    [...document.querySelectorAll('.board-token')].map((e) => e.getAttribute('aria-label') ?? '?'),
  )
}

/** Quem está DESENHADO deslocado agora, no meio do gesto. */
function whoIsSlidingNow(page: Page) {
  return page.evaluate(() =>
    [...document.querySelectorAll('.board-token.board-dragging')].map(
      (e) => e.getAttribute('aria-label')?.split(' em ')[0] ?? '?',
    ),
  )
}

async function theSquareSide(page: Page) {
  return page.evaluate(() =>
    Number.parseFloat(getComputedStyle(document.querySelector('.board-token')!).getPropertyValue('--quadrado')),
  )
}

test('no rascunho, arrastar a segunda peça move a SEGUNDA — e nenhuma outra', async ({ page }) => {
  const { apagar: remove } = await aDraftWith(page, [
    { nome: 'Alfa', x: 3, y: 3 },
    { nome: 'Beta', x: 8, y: 3 },
  ])
  try {
    await expect(page.locator('.board-token'), 'as duas peças não entraram').toHaveCount(2)
    expect(await whereEachTokenIs(page)).toEqual(['Alfa em 3, 3', 'Beta em 8, 3'])

    const beta = page.locator('.board-token').nth(1)
    const box = await beta.boundingBox()
    if (!box) throw new Error('a peça não tem caixa: o arrasto não tem de onde partir')
    const square = await theSquareSide(page)
    const middle = { x: box.x + box.width / 2, y: box.y + box.height / 2 }

    await page.mouse.move(middle.x, middle.y)
    await page.mouse.down()
    // PASSOS INTERMEDIÁRIOS: um salto direto não atravessa casa nenhuma.
    for (let i = 1; i <= 6; i++) await page.mouse.move(middle.x + (square * 2 * i) / 6, middle.y)

    // QUEM DESLIZA SOB O DEDO, e este pedaço é metade do defeito: com ele no
    // lugar quem ganha a classe é o ALFA, que ninguém pegou — a peça errada
    // corre atrás do dedo desde o primeiro quadro.
    expect(await whoIsSlidingNow(page), 'a peça que desliza não é a que foi pega').toEqual(['Beta'])

    await page.mouse.up()
    await expect
      .poll(() => whereEachTokenIs(page), { timeout: 5_000 })
      .toEqual(['Alfa em 3, 3', 'Beta em 10, 3'])
  } finally {
    await remove()
  }
})

/**
 * O CONTROLE, e ele não é decoração: sem esta metade, um seletor que não casasse
 * com nada faria o caso de cima passar por não achar peça deslizando nenhuma.
 * Aqui o arrasto TEM de mover, e com uma peça só o defeito antigo não aparece —
 * que é exatamente por que ele sobreviveu.
 */
test('no rascunho com UMA peça, arrastar move essa peça', async ({ page }) => {
  const { apagar: remove } = await aDraftWith(page, [{ nome: 'Alfa', x: 3, y: 3 }])
  try {
    const alpha = page.locator('.board-token').first()
    const box = await alpha.boundingBox()
    if (!box) throw new Error('a peça não tem caixa')
    const square = await theSquareSide(page)
    const middle = { x: box.x + box.width / 2, y: box.y + box.height / 2 }
    await page.mouse.move(middle.x, middle.y)
    await page.mouse.down()
    for (let i = 1; i <= 6; i++) await page.mouse.move(middle.x, middle.y + (square * 2 * i) / 6)
    expect(await whoIsSlidingNow(page), 'a peça pega não deslizou').toEqual(['Alfa'])
    await page.mouse.up()
    await expect.poll(() => whereEachTokenIs(page), { timeout: 5_000 }).toEqual(['Alfa em 3, 5'])
  } finally {
    await remove()
  }
})
