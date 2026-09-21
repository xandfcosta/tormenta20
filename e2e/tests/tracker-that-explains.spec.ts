import { expect, type Page, test } from '@playwright/test'
import { openTheBoard, closeTheTracker, disposableTable, putACombatantInTheTracker } from './support/table'

/**
 * A FILA EXPLICA O QUE MUDOU: a piscada do vital e o pulso da vez.
 *
 * O que estes casos protegem é o que FALTA quando nada anima: **numa lista de
 * nove combatentes um número troca sozinho e ninguém viu QUEM sangrou.**
 *
 * E2E porque a pergunta é sobre a LINHA DO TEMPO de uma animação disparada por
 * um remendo do servidor. Em jsdom não há `Element.animate`, não há duração e
 * não há morph.
 *
 * # O que se mede é o EFEITO
 *
 * A piscada é um véu com `opacity` animada, e o pulso é `transform` + sombra.
 * Nenhum dos dois muda a geometria da linha — então a amostragem de posição que
 * serve para a peça não serve aqui. O que se amostra é o véu (que existe SÓ
 * durante a animação) e a contagem de `getAnimations()` da linha, quadro a
 * quadro.
 *
 * **Capturar tela não serviria**: o `screenshot()` do Playwright desliga
 * animação por padrão e FINALIZA as finitas antes de fotografar — três quadros
 * byte a byte idênticos.
 */
test.use({ storageState: '.auth/user.json' })

/**
 * Amostra, quadro a quadro, o que está ANIMANDO na fila.
 *
 * Armada DEPOIS do arranjo e imediatamente antes do gesto: uma sonda de vida
 * longa mede tudo o que acontece na janela dela, e a janela é parte do desenho.
 */
async function trackerAnimations(
  page: Page,
  gestureName: () => Promise<void>,
  ms = 2500,
): Promise<{ veus: number; linhasAnimando: number }> {
  await page.evaluate((limit) => {
    const w = window as unknown as { __f: { veus: number; linhas: number } }
    w.__f = { veus: 0, linhas: 0 }
    const start = performance.now()
    const step = () => {
      // O VÉU da piscada nasce e morre com a animação: contá-lo é contar a
      // piscada, sem depender de qual cor ela usou.
      w.__f.veus += document.querySelectorAll('#table-tracker li > div[aria-hidden="true"]').length
      for (const row of document.querySelectorAll('#table-tracker li')) {
        if (row.getAnimations().length > 0) w.__f.linhas++
      }
      if (performance.now() - start < limit) requestAnimationFrame(step)
    }
    requestAnimationFrame(step)
  }, ms)
  await gestureName()
  await page.waitForTimeout(ms + 100)
  const m = await page.evaluate(() => (window as unknown as { __f: { veus: number; linhas: number } }).__f)
  return { veus: m.veus, linhasAnimando: m.linhas }
}

async function aTrackerWithTwo(page: Page) {
  const table = await disposableTable(page)
  await openTheBoard(page, table.mesa)
  await putACombatantInTheTracker(page, 'Ogro do E2E', 40)
  await putACombatantInTheTracker(page, 'Anao do E2E', 30)
  return table
}

test('ferir um combatente pisca a LINHA dele, e curar pisca de outra cor', async ({ page }) => {
  const { apagar: eraseBtn } = await aTrackerWithTwo(page)
  try {
    const hurting = await trackerAnimations(page, () =>
      page.getByRole('button', { name: /^Ferir/ }).first().click(),
    )
    expect(
      hurting.veus,
      'ninguém piscou ao perder PV: a fila continua trocando o número em silêncio',
    ).toBeGreaterThan(0)

    // A COR do véu diz o sinal, e é a única coisa que separa "levei 12" de
    // "curei 12". Medida no véu que está no ar durante a cura.
    //
    // # Ela ESPERA O ZERO antes do gesto, e o orçamento é em TEMPO
    //
    // Guardar `antes = veus.length` e esperar `> antes` tem duas fragilidades:
    //
    //   - se um véu SAI e outro ENTRA, a contagem não sobe e o sucesso passa por
    //     ausência — é o "mostrador cujo REPOUSO é igual ao sucesso" do guia,
    //     com um contador no lugar do rótulo;
    //   - um orçamento em QUADROS não é o relógio do que se espera, que é uma
    //     ida ao SERVIDOR: clique, POST, remendo SSE, morph. O de quadros não
    //     estica quando o outro fica lento.
    //
    // Esperar a contagem chegar a ZERO torna o "apareceu" inequívoco.
    //
    // A CAUSA RAIZ DA INTERMITÊNCIA SEGUE ABERTA — nunca reproduzida sob
    // instrumentação —, e é por isso que a mensagem de falha carrega o estado:
    // a próxima ocorrência chega diagnosticada em vez de exigir outra caçada.
    const measurement = await page.evaluate(async () => {
      const veils = () => [...document.querySelectorAll('#table-tracker li > div[aria-hidden="true"]')]
      const frame = () => new Promise((p) => requestAnimationFrame(p))

      const clearing = performance.now()
      while (veils().length > 0 && performance.now() - clearing < 3000) await frame()
      const veilsBefore = veils().length

      const heal = [...document.querySelectorAll('button')].find((b) =>
        (b.getAttribute('aria-label') ?? '').startsWith('Curar'),
      )
      const foundButton = !!heal
      heal?.click()

      const t0 = performance.now()
      while (performance.now() - t0 < 3000) {
        await frame()
        const now = veils()
        if (now.length > 0) {
          return { cor: getComputedStyle(now[now.length - 1]).background,
                   veusAntes: veilsBefore, achouBotao: foundButton, ms: Math.round(performance.now() - t0) }
        }
      }
      return { cor: '', veusAntes: veilsBefore, achouBotao: foundButton, ms: Math.round(performance.now() - t0) }
    })
    const healColor = measurement.cor
    expect(
      healColor,
      `a cura não pintou véu nenhum em ${measurement.ms}ms — ` +
        `véus pendurados antes do gesto: ${measurement.veusAntes} (zero é o esperado), ` +
        `botão Curar encontrado: ${measurement.achouBotao}. ` +
        'Se os véus antes forem > 0, o véu do DANO não tinha sumido; se o botão for ' +
        'false, o gesto não chegou a acontecer; se os dois estiverem certos, a ' +
        'piscada não foi disparada — e aí o suspeito é o MutationObserver de ' +
        '`aria-valuenow` ou o morph apagando o véu na janela de um quadro (ALE-322).',
    ).not.toBe('')
    // `--hp-full` é a cor de vida cheia; `--hp-critical` é a do dano. Elas têm
    // de ser DIFERENTES, senão o sinal não é sinal.
    const damageColor = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue('--hp-critical').trim(),
    )
    expect(
      healColor.includes(damageColor) && damageColor !== '',
      `a cura pintou com a cor do dano (${damageColor}): perder e ganhar PV ficam iguais`,
    ).toBe(false)
  } finally {
    await eraseBtn()
  }
})

test('entrar na vez pulsa a linha do combatente que entrou', async ({ page }) => {
  const { apagar: remove } = await aTrackerWithTwo(page)
  try {
    // Pelo ajudante e não pelo clique cru: ele ESPERA a gaveta fechar.
    await closeTheTracker(page)

    // A CENA PRECISA COMEÇAR, senão não há vez para entrar: o `PodeAvancar` é
    // `SceneActive && len(Initiative) > 0`, e sem ele o botão de avanço nasce
    // `disabled`. O sintoma é um timeout dizendo "waiting for element to be
    // visible, enabled and stable" — e o que falha é o ENABLED, com o botão
    // parado e visível na tela o tempo todo.
    // DOIS cliques: "Iniciar cena" abre os três tipos (p252), e a fila só
    // existe na de AÇÃO.
    await page.locator('summary[aria-label="Iniciar uma cena"]').click()
    await page.getByRole('button', { name: 'Iniciar uma cena de Ação' }).click()
    await expect(page.getByRole('button', { name: /^Começar:/ })).toBeEnabled()

    const starting = await trackerAnimations(page, () =>
      page.getByRole('button', { name: /^Começar:/ }).first().click(),
    )
    expect(
      starting.linhasAnimando,
      'ninguém pulsou ao entrar na vez: o holofote teleporta e a mesa não o vê andar',
    ).toBeGreaterThan(0)

  } finally {
    await remove()
  }
})

/**
 * A CONDIÇÃO que chega SURGE.
 *
 * É o único dos cinco disparos que é de mount de verdade, e o guarda mede o que
 * a mesa vê: o crachá é pintado com opacidade crescente durante 150ms. Contar
 * `getAnimations()` nele responde a pergunta sem depender de qual keyframe foi
 * escolhido.
 */
test('aplicar uma condição faz o crachá dela surgir', async ({ page }) => {
  const { apagar: erase } = await aTrackerWithTwo(page)
  try {
    await page.evaluate(() => {
      const w = window as unknown as { __c: number }
      w.__c = 0
      const t0 = performance.now()
      const step = () => {
        for (const c of document.querySelectorAll('[data-condicao]')) {
          if (c.getAnimations().length > 0) w.__c++
        }
        if (performance.now() - t0 < 900) requestAnimationFrame(step)
      }
      requestAnimationFrame(step)
    })

    await page.getByRole('button', { name: 'Condições de Ogro do E2E' }).click()
    await page.getByRole('button', { name: 'Abalado', exact: true }).click()
    await page.waitForTimeout(1000)

    const tiles = await page.evaluate(() => (window as unknown as { __c: number }).__c)
    expect(
      tiles,
      'o crachá da condição apareceu sem animar: o estado chegou em silêncio',
    ).toBeGreaterThan(0)
  } finally {
    await erase()
  }
})
