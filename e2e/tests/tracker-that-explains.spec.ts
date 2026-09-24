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
      //
      // Ele é irmão do `<body>` e não filho da linha, e por isso o seletor é o
      // DATA-ATRIBUTO: dentro da linha ele era removido pelo segundo remendo do
      // gesto (ALE-322). Casar forma de DOM aqui nos faria medir o lugar antigo.
      w.__f.veus += document.querySelectorAll('[data-vital-blink]').length
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
    // A CAUSA RAIZ FOI ACHADA, e não era a sonda: o véu morava DENTRO da linha e
    // o segundo remendo do gesto o levava embora. Medido em 50 piscadas antes do
    // conserto — 43 truncadas em ~200ms, SEIS abaixo de 30ms, uma completa — e 50
    // de 50 completas depois (ALE-322). A mensagem de falha continua carregando o
    // estado: o que ela diagnostica agora é a próxima causa, não esta.
    const measurement = await page.evaluate(async () => {
      const veils = () => [...document.querySelectorAll('[data-vital-blink]')]
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

/**
 * A PISCADA DE QUEM APANHA DUAS VEZES SEGUIDAS (ALE-322).
 *
 * O caso acima prende que a piscada EXISTE. Este prende que ela não é apagada
 * pelo gesto seguinte — e o que estava quebrado era isto.
 *
 * O véu é um nó que o JS pendura; o reconciliador do Datastar remove todo filho
 * que não veio no HTML do servidor. Enquanto ele morava dentro da `<li>`, o
 * remendo do PRÓXIMO gesto o apagava: medido, **um gesto isolado deixava o véu
 * viver 361ms e dois gestos a 60ms de distância matavam o primeiro em 17ms**.
 * Quem sangra duas vezes seguidas não via a primeira piscada.
 *
 * O `requestAnimationFrame` do observador não cobre isso: ele só ganha do morph
 * do próprio gesto.
 *
 * E2E porque a pergunta é sobre a LINHA DO TEMPO de um nó que um morph de
 * verdade tenta remover. Não há morph no jsdom, e um teste de unidade que
 * chamasse `piscarVital` mediria o NASCIMENTO — que nunca foi o defeito.
 */
test('ferir duas vezes seguidas não apaga a piscada da primeira', async ({ page }) => {
  const { apagar: erase } = await aTrackerWithTwo(page)
  try {
    const life = await page.evaluate(async () => {
      const blinks = () => document.querySelectorAll('[data-vital-blink]')
      const frame = () => new Promise((paint) => requestAnimationFrame(paint))
      const hurt = () =>
        [...document.querySelectorAll('button')].find((b) =>
          (b.getAttribute('aria-label') ?? '').startsWith('Ferir'),
        )

      // REPOUSO PRIMEIRO: com um véu pendurado de antes, "existe" responderia
      // sobre a piscada anterior — ver "mostrador cujo REPOUSO é igual ao
      // sucesso" no guia da raiz.
      const clearing = performance.now()
      while (blinks().length > 0 && performance.now() - clearing < 3000) await frame()
      if (blinks().length > 0) return { born: false, secondGesture: false, ms: 0 }
      if (!hurt()) return { born: false, secondGesture: false, ms: 0 }

      hurt()?.click()
      const waiting = performance.now()
      while (blinks().length === 0 && performance.now() - waiting < 3000) await frame()
      const first = blinks()[0]
      if (!first) return { born: false, secondGesture: false, ms: 0 }
      const born = performance.now()

      // O SEGUNDO GESTO é o que este caso vem medir, e ele tem de ACONTECER: um
      // clique que não pega deixaria a duração passar por não haver quem a
      // interrompesse.
      //
      // O controle conta véus que NASCERAM, e não véus no ar ao mesmo tempo. A
      // primeira versão contava concorrentes e era INÚTIL: com o defeito presente
      // o primeiro véu já morreu quando o segundo nasce, então "dois no ar" nunca
      // acontece — o controle falhava junto com o defeito e entregava a mensagem
      // errada. Um controle tem de poder passar enquanto a asserção reprova.
      let bornCount = 0
      new MutationObserver((records) => {
        for (const record of records) {
          for (const node of record.addedNodes) {
            if (node instanceof Element && node.hasAttribute('data-vital-blink')) bornCount++
          }
        }
      }).observe(document.body, { subtree: true, childList: true })

      await new Promise((later) => setTimeout(later, 60))
      const secondButton = hurt()
      secondButton?.click()
      const counting = performance.now()
      while (bornCount === 0 && performance.now() - counting < 1500) await frame()

      while (first.isConnected && performance.now() - born < 2000) await frame()
      return { born: true, secondGesture: !!secondButton && bornCount > 0, ms: Math.round(performance.now() - born) }
    })

    expect(life.born, 'a piscada não nasceu: sem ela este caso não mede duração nenhuma').toBe(true)
    expect(
      life.secondGesture,
      'o SEGUNDO gesto não chegou a pôr um véu no ar, e é ele que este caso vem ' +
        'medir — sem ele a duração passaria por não haver quem a interrompesse (ALE-322)',
    ).toBe(true)
    // A animação pede 380ms; o piso de 340 não prende jitter de quadro. O que ele
    // barra é a ordem de grandeza do defeito: 17ms, medidos.
    expect(
      life.ms,
      `a piscada do primeiro golpe viveu ${life.ms}ms e a animação pede 380 — o ` +
        'remendo do segundo golpe a apagou, o que quer dizer que o véu voltou a ' +
        'morar DENTRO da linha que o morph reconcilia (ALE-322)',
    ).toBeGreaterThan(340)
  } finally {
    await erase()
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
