import { expect, type Page, test } from '@playwright/test'
import { openTheBoard, closeTheTracker, disposableTable, putACombatantInTheTracker } from './support/table'

/**
 * A FILA EXPLICA O QUE MUDOU: a piscada do vital e o pulso da vez (ALE-174).
 *
 * A issue nasceu de uma auditoria cujo veredito contraria a intuição: *"o CSS
 * de animação deste app é disciplinado e barato; nenhuma animação existente
 * compromete um frame"*. O problema não era polir o que existe — era o que
 * falta. **Numa lista de nove combatentes um número troca sozinho e ninguém viu
 * QUEM sangrou.**
 *
 * E2E porque a pergunta é sobre a LINHA DO TEMPO de uma animação disparada por
 * um remendo do servidor. Em jsdom não há `Element.animate`, não há duração e
 * não há morph.
 *
 * # O que se mede é o EFEITO, e a lição custou dois guardas na fatia anterior
 *
 * A piscada é um véu com `opacity` animada, e o pulso é `transform` + sombra.
 * Nenhum dos dois muda a geometria da linha — então a amostragem de posição que
 * serve para a peça não serve aqui. O que se amostra é o `opacity` computado do
 * véu (que existe SÓ durante a animação) e a contagem de `getAnimations()` da
 * linha, quadro a quadro.
 *
 * **Capturar tela não serviria**: o `screenshot()` do Playwright desliga
 * animação por padrão e FINALIZA as finitas antes de fotografar (medido na
 * ALE-174, três quadros byte a byte idênticos).
 */
test.use({ storageState: '.auth/user.json' })

/**
 * Amostra, quadro a quadro, o que está ANIMANDO na fila.
 *
 * Armada DEPOIS do arranjo e imediatamente antes do gesto, pela lição da fatia
 * anterior: uma sonda de vida longa mede tudo o que acontece na janela dela, e a
 * janela é parte do desenho.
 */
async function trackerAnimations(
  page: Page,
  gesto: () => Promise<void>,
  ms = 2500,
): Promise<{ veus: number; linhasAnimando: number }> {
  await page.evaluate((limite) => {
    const w = window as unknown as { __f: { veus: number; linhas: number } }
    w.__f = { veus: 0, linhas: 0 }
    const inicio = performance.now()
    const passo = () => {
      // O VÉU da piscada nasce e morre com a animação: contá-lo é contar a
      // piscada, sem depender de qual cor ela usou.
      w.__f.veus += document.querySelectorAll('#table-tracker li > div[aria-hidden="true"]').length
      for (const linha of document.querySelectorAll('#table-tracker li')) {
        if (linha.getAnimations().length > 0) w.__f.linhas++
      }
      if (performance.now() - inicio < limite) requestAnimationFrame(passo)
    }
    requestAnimationFrame(passo)
  }, ms)
  await gesto()
  await page.waitForTimeout(ms + 100)
  const m = await page.evaluate(() => (window as unknown as { __f: { veus: number; linhas: number } }).__f)
  return { veus: m.veus, linhasAnimando: m.linhas }
}

async function aTrackerWithTwo(page: Page) {
  const mesa = await disposableTable(page)
  await openTheBoard(page, mesa.mesa)
  await putACombatantInTheTracker(page, 'Ogro do E2E', 40)
  await putACombatantInTheTracker(page, 'Anao do E2E', 30)
  return mesa
}

test('ferir um combatente pisca a LINHA dele, e curar pisca de outra cor', async ({ page }) => {
  const { apagar } = await aTrackerWithTwo(page)
  try {
    const ferindo = await trackerAnimations(page, () =>
      page.getByRole('button', { name: /^Ferir/ }).first().click(),
    )
    expect(
      ferindo.veus,
      'ninguém piscou ao perder PV: a fila continua trocando o número em silêncio',
    ).toBeGreaterThan(0)

    // A COR do véu diz o sinal, e é a única coisa que separa "levei 12" de
    // "curei 12". Medida no véu que está no ar durante a cura.
    const corDaCura = await page.evaluate(async () => {
      const antes = document.querySelectorAll('#table-tracker li > div[aria-hidden="true"]').length
      const curar = [...document.querySelectorAll('button')].find((b) =>
        (b.getAttribute('aria-label') ?? '').startsWith('Curar'),
      )
      curar?.click()
      for (let i = 0; i < 40; i++) {
        await new Promise((p) => requestAnimationFrame(p))
        const veus = [...document.querySelectorAll('#table-tracker li > div[aria-hidden="true"]')]
        if (veus.length > antes) return getComputedStyle(veus[veus.length - 1]).background
      }
      return ''
    })
    expect(corDaCura, 'a cura não pintou véu nenhum').not.toBe('')
    // `--hp-full` é a cor de vida cheia; `--hp-critical` é a do dano. Elas têm
    // de ser DIFERENTES, senão o sinal não é sinal.
    const corDoDano = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue('--hp-critical').trim(),
    )
    expect(
      corDaCura.includes(corDoDano) && corDoDano !== '',
      `a cura pintou com a cor do dano (${corDoDano}): perder e ganhar PV ficam iguais`,
    ).toBe(false)
  } finally {
    await apagar()
  }
})

test('entrar na vez pulsa a linha do combatente que entrou', async ({ page }) => {
  const { apagar } = await aTrackerWithTwo(page)
  try {
    // Pelo ajudante e não pelo clique cru: ele ESPERA a gaveta fechar.
    await closeTheTracker(page)

    // A CENA PRECISA COMEÇAR, senão não há vez para entrar: o `PodeAvancar` é
    // `SceneActive && len(Initiative) > 0`, e sem ele o botão de avanço nasce
    // `disabled`. O sintoma é um timeout dizendo "waiting for element to be
    // visible, enabled and stable" — e o que falha é o ENABLED, com o botão
    // parado e visível na tela o tempo todo.
    await page.getByRole('button', { name: 'Iniciar cena' }).click()
    await expect(page.getByRole('button', { name: /^Começar:/ })).toBeEnabled()

    const comecando = await trackerAnimations(page, () =>
      page.getByRole('button', { name: /^Começar:/ }).first().click(),
    )
    expect(
      comecando.linhasAnimando,
      'ninguém pulsou ao entrar na vez: o holofote teleporta e a mesa não o vê andar',
    ).toBeGreaterThan(0)

  } finally {
    await apagar()
  }
})

/**
 * A CONDIÇÃO que chega SURGE (ALE-174, P4).
 *
 * É a única dos cinco disparos que é de mount de verdade, e o guarda mede o que
 * a mesa vê: o crachá é pintado com opacidade crescente durante 150ms. Contar
 * `getAnimations()` nele responde a pergunta sem depender de qual keyframe foi
 * escolhido.
 */
test('aplicar uma condição faz o crachá dela surgir', async ({ page }) => {
  const { apagar } = await aTrackerWithTwo(page)
  try {
    await page.evaluate(() => {
      const w = window as unknown as { __c: number }
      w.__c = 0
      const t0 = performance.now()
      const passo = () => {
        for (const c of document.querySelectorAll('[data-condicao]')) {
          if (c.getAnimations().length > 0) w.__c++
        }
        if (performance.now() - t0 < 900) requestAnimationFrame(passo)
      }
      requestAnimationFrame(passo)
    })

    await page.getByRole('button', { name: 'Condições de Ogro do E2E' }).click()
    await page.getByRole('button', { name: 'Abalado', exact: true }).click()
    await page.waitForTimeout(1000)

    const quadros = await page.evaluate(() => (window as unknown as { __c: number }).__c)
    expect(
      quadros,
      'o crachá da condição apareceu sem animar: o estado chegou em silêncio',
    ).toBeGreaterThan(0)
  } finally {
    await apagar()
  }
})
