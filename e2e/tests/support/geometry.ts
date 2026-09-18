import { type Page, expect } from '@playwright/test'

/**
 * Asserções de RELAÇÃO entre caixas.
 *
 * As asserções globais e negativas — a página não rola de lado, não rola para
 * baixo — só disparam quando a quebra chega ao elemento RAIZ, e todo defeito
 * achado por print quebrou DENTRO de um contêiner. Pior: a cena é feita de
 * `overflow-hidden` e `min-h-0` postos ali justamente para a página não rolar,
 * e cada um deles ABSORVE o sintoma — os consertos que fazem a asserção passar
 * são os mesmos que a cegam.
 *
 * O que falta é relação: alinhamento, proporção, preenchimento, containment.
 * Cada primitiva daqui nasceu de um defeito real, e o docstring dela cita qual
 * — nenhuma foi inventada por simetria.
 *
 * Todas medem por `getBoundingClientRect` num `evaluate` só (~1ms), sem
 * screenshot: são determinísticas e não têm baseline para apodrecer.
 *
 * **A divisão que importa:** o olho no Chrome JULGA ("isso está ruim"), e isto
 * aqui CONGELA o que o olho já julgou. Alinhamento e proporção são invariantes;
 * hierarquia e bom gosto não são, e automação nenhuma vai ter.
 */

/** Uma caixa medida, com o texto que identifica quem ela é no relatório. */
type Caixa = { x: number; y: number; width: number; height: number; texto: string }

async function medir(page: Page, seletor: string): Promise<Caixa[]> {
  return page.$$eval(seletor, (nodes) =>
    nodes
      .map((node) => {
        const r = node.getBoundingClientRect()
        return {
          x: Math.round(r.x),
          y: Math.round(r.y),
          width: Math.round(r.width),
          height: Math.round(r.height),
          texto:
            node.getAttribute('aria-label') ??
            (node.textContent ?? '').trim().slice(0, 24) ??
            node.tagName,
        }
      })
      .filter((box) => box.width > 0 && box.height > 0),
  )
}

/**
 * Botões do MESMO verbo, em linhas diferentes, ficam na MESMA coluna (ALE-141).
 *
 * O conteúdo pode variar por linha com razão — o olho de ocultar PV só existe
 * em linha com vida —, mas a POSIÇÃO não pode variar junto: sem lugar
 * reservado a fileira encolhe, e o `+` de uma linha cai onde está o lápis de
 * outra.
 *
 * @example await expectFormaColuna(page, 'button[aria-label^="Curar "]')
 */
export async function expectFormaColuna(page: Page, seletor: string, folga = 1): Promise<void> {
  const caixas = await medir(page, seletor)
  expect(caixas.length, `nenhuma caixa em ${seletor} — o seletor não casou nada`).toBeGreaterThan(0)
  const colunas = [...new Set(caixas.map((c) => c.x))].sort((a, b) => a - b)
  const espalhamento = colunas.length === 0 ? 0 : colunas[colunas.length - 1] - colunas[0]
  expect(
    espalhamento,
    `${seletor} em ${colunas.length} colunas (x: ${colunas.join(', ')}) — deveria ser uma`,
  ).toBeLessThanOrEqual(folga)
}

/**
 * A caixa tem a PROPORÇÃO que promete (ALE-126).
 *
 * Proporção é invariante: depois que alguém decidiu que o retrato é um
 * círculo, 1:1 vale para sempre. Largura fixa esticada pela altura do cartão
 * vira uma tira vertical que lê como erro de layout.
 *
 * @example await expectProporcao(page, '[data-slot=portrait]', 0.9, 1.1)
 */
export async function expectProporcao(
  page: Page,
  seletor: string,
  min: number,
  max: number,
): Promise<void> {
  const caixas = await medir(page, seletor)
  expect(caixas.length, `nenhuma caixa em ${seletor}`).toBeGreaterThan(0)
  for (const caixa of caixas) {
    const razao = caixa.width / caixa.height
    expect(
      razao,
      `${caixa.texto || seletor}: ${caixa.width}×${caixa.height} dá ${razao.toFixed(2)}, fora de ${min}–${max}`,
    ).toBeGreaterThanOrEqual(min)
    expect(razao).toBeLessThanOrEqual(max)
  }
}

/**
 * Os filhos ENCHEM a largura do pai (ALE-138).
 *
 * O oposto do containment: aqui o defeito é sobrar espaço morto, não faltar.
 * A regra da casa é que a cena preencha o espaço que recebe.
 *
 * @example await expectEnchePai(page, '[role=tablist]', '[role=tab]')
 */
export async function expectEnchePai(
  page: Page,
  pai: string,
  filho: string,
  folga = 4,
): Promise<void> {
  const sobra = await page.evaluate(
    ([seletorPai, seletorFilho]) => {
      const caixaPai = document.querySelector(seletorPai as string)
      if (!caixaPai) return { erro: `pai ${seletorPai} não existe` }
      const filhos = [...caixaPai.querySelectorAll(seletorFilho as string)].filter(
        (node) => node.getBoundingClientRect().width > 0,
      )
      if (filhos.length === 0) return { erro: `nenhum ${seletorFilho} dentro de ${seletorPai}` }
      const r = caixaPai.getBoundingClientRect()
      const direita = Math.max(...filhos.map((f) => f.getBoundingClientRect().right))
      // O padding do pai não é espaço morto: descontar é o que evita exigir que
      // o filho encoste na borda de um contêiner que tem respiro de propósito.
      const estilo = getComputedStyle(caixaPai)
      const limite = r.right - Number.parseFloat(estilo.paddingRight || '0')
      return { sobra: Math.round(limite - direita), largura: Math.round(r.width) }
    },
    [pai, filho],
  )
  expect(sobra.erro, sobra.erro ?? '').toBeUndefined()
  expect(
    sobra.sobra,
    `${filho} para ${sobra.sobra}px antes do fim de ${pai} (largura ${sobra.largura}px)`,
  ).toBeLessThanOrEqual(folga)
}

/**
 * NADA é pintado para fora da caixa do próprio pai (ALE-125, ALE-148).
 *
 * A relação é filho contra PAI, e não contra a cena: um crachá que vaza 6px do
 * cartão é desenhado sobre o vizinho muito antes de chegar perto da borda da
 * coluna, e a primeira versão desta asserção — que media contra a coluna —
 * passava VERDE sobre o defeito.
 *
 * Ignora `position: absolute` (que sai do fluxo de propósito — o ✕ de
 * desequipar) e pais que rolam na horizontal (onde transbordar é a função).
 *
 * @example await expectNadaEscapa(page, 'section[aria-label="Mochila"]')
 */
export async function expectNadaEscapa(page: Page, pai: string, filhos = '*'): Promise<void> {
  const escapando = await page.evaluate(
    ([seletorPai, seletorFilhos]) => {
      const raiz = document.querySelector(seletorPai as string)
      if (!raiz) return null
      return [...raiz.querySelectorAll(seletorFilhos as string)]
        .filter((node) => {
          const pai = node.parentElement
          if (!pai) return false
          const estilo = getComputedStyle(node)
          if (estilo.position === 'absolute' || estilo.position === 'fixed') return false
          if (getComputedStyle(pai).overflowX !== 'visible') return false
          const r = node.getBoundingClientRect()
          return r.width > 0 && r.right > pai.getBoundingClientRect().right + 1
        })
        .map((node) => (node.textContent ?? '').trim().slice(0, 30))
        .slice(0, 5)
    },
    [pai, filhos],
  )
  expect(escapando, `o pai ${pai} não existe na tela`).not.toBeNull()
  expect(escapando, `pintado para fora do pai, dentro de ${pai}`).toEqual([])
}

/**
 * Nada que se possa CLICAR fica fora da janela sem caminho até ele (ALE-160).
 *
 * É a irmã do `expectNadaEscapa`, e a lacuna era estrutural: aquele pula, DE
 * PROPÓSITO, todo pai cujo `overflow-x` não é `visible`, porque ali transbordar
 * é a função. Só que a cena inteira é feita de `overflow-x-hidden`, e por baixo
 * dele um botão vai parar em x=392 numa tela de 390 — fora da tela, sem rolagem
 * que chegue nele, com `document.scrollWidth` jurando que não há estouro.
 *
 * A diferença que faz a asserção funcionar é `rolavel`: estar fora da viewport
 * é NORMAL — é o que acontece com tudo abaixo da dobra de uma lista. O defeito
 * é estar fora e **não haver eixo que role até lá**.
 *
 * Mede só o que é interativo, porque o que se perde quando isso quebra é a
 * AÇÃO. E ignora `sr-only`, que mede ~1px por definição e acusaria em toda
 * tela (o crachá de pendências da ficha é o caso conhecido).
 *
 * @example await expectDentroDaJanela(page, 'main')
 */
export async function expectDentroDaJanela(page: Page, raiz = 'body'): Promise<void> {
  const fora = await page.evaluate((seletorRaiz) => {
    const root = document.querySelector(seletorRaiz as string)
    if (!root) return null
    const rolavel = (node: Element, eixo: 'x' | 'y'): boolean => {
      for (let atual: Element | null = node; atual; atual = atual.parentElement) {
        const estilo = getComputedStyle(atual)
        const overflow = eixo === 'x' ? estilo.overflowX : estilo.overflowY
        if (overflow === 'auto' || overflow === 'scroll') return true
      }
      return false
    }
    const janela = { largura: window.innerWidth, altura: window.innerHeight }
    return [...root.querySelectorAll('a, button, input, select, textarea, [role="button"]')]
      .filter((node) => {
        const r = node.getBoundingClientRect()
        if (r.width <= 1 || r.height <= 1) return false // sr-only e afins
        const foraX = r.right > janela.largura + 1 || r.left < -1
        const foraY = r.bottom > janela.altura + 1 || r.top < -1
        return (foraX && !rolavel(node, 'x')) || (foraY && !rolavel(node, 'y'))
      })
      .map((node) => {
        const r = node.getBoundingClientRect()
        const nome = node.getAttribute('aria-label') ?? (node.textContent ?? '').trim().slice(0, 24)
        return `${nome || node.tagName} em x ${Math.round(r.left)}–${Math.round(r.right)}, y ${Math.round(r.top)}–${Math.round(r.bottom)}`
      })
      .slice(0, 5)
  }, raiz)

  expect(fora, `a raiz ${raiz} não existe na tela`).not.toBeNull()
  expect(
    fora,
    `alcançável por ninguém: fora da janela de ${page.viewportSize()?.width}×${page.viewportSize()?.height} e sem rolagem que chegue lá`,
  ).toEqual([])
}

/**
 * Nenhum contêiner DENTRO da cena rola de lado (ALE-178).
 *
 * A regra da casa é que a cena não rola horizontalmente, e o
 * `expectNoHorizontalOverflow` a afirma — só que na RAIZ. Quando o estouro
 * acontece num painel interno, a raiz continua limpa e a asserção passa.
 *
 * E o `expectDentroDaJanela` também não pega: um ✕ de encerrar empurrado para
 * x=466 numa tela de 390 é ALCANÇÁVEL se o painel rola, então aquele guarda
 * passa verde com razão. Rolar de lado para achar o botão de fechar é
 * justamente a experiência que esta asserção existe para impedir.
 *
 * Ignora quem rola de lado DE PROPÓSITO, marcado com `data-rola-lado`.
 *
 * @example await expectNadaRolaDeLado(page, '.scene-grimorio')
 */
export async function expectNadaRolaDeLado(page: Page, raiz = 'body'): Promise<void> {
  const rolando = await page.evaluate((seletorRaiz) => {
    const root = document.querySelector(seletorRaiz as string)
    if (!root) return null
    return [...root.querySelectorAll<HTMLElement>('*')]
      .filter((node) => {
        if (node.closest('[data-rola-lado]')) return false
        const estilo = getComputedStyle(node)
        const podeRolar = estilo.overflowX === 'auto' || estilo.overflowX === 'scroll'
        return podeRolar && node.scrollWidth > node.clientWidth + 1
      })
      .map((node) => {
        const nome = node.getAttribute('aria-label') ?? node.className.slice(0, 40) ?? node.tagName
        return `${nome}: conteúdo de ${node.scrollWidth}px numa caixa de ${node.clientWidth}px`
      })
      .slice(0, 5)
  }, raiz)

  expect(rolando, `a raiz ${raiz} não existe na tela`).not.toBeNull()
  expect(rolando, 'painel rolando de lado dentro da cena, que não deveria rolar').toEqual([])
}

/**
 * O palco não deixa BANDA VAZIA embaixo do que ele mostra (ALE-175).
 *
 * A regra da casa — "uma cena preenche o espaço que recebe" — só tinha
 * asserção de LARGURA no `expectEnchePai`. Esta mede a banda vazia DEPOIS do
 * último elemento do palco, que é onde mora o sintoma de altura: uma lista com
 * tampa de `45vh` transborda por cima e pinta um quarto da tela de nada.
 *
 * Tinta é definida por exclusão de contêiner: conta a caixa de quem tem texto
 * próprio, de quem é interativo e de quem é gráfico. Um `div` de arranjo não
 * conta, porque é justamente ele quem se estica por cima da faixa morta e faria
 * a medição jurar que o espaço está ocupado.
 *
 * O que este guarda NÃO faz é prender altura. Altura é consequência do formato
 * e prendê-la seria prender o número errado; o que a cena promete é não deixar
 * banda vazia depois do último elemento.
 *
 * Cuidados que a medição exige:
 * - o palco tem de estar TRANSBORDANDO, senão a faixa vazia é uma lista que
 *   coube e a asserção não prova nada. Quem chama afirma isso antes.
 * - ignora caixa de 1px, que é `sr-only` por definição e existe em toda tela.
 *
 * @example await expectSemFaixaMorta(page, '[aria-labelledby=table-bestiary-panel]')
 */
export async function expectSemFaixaMorta(
  page: Page,
  palco: string,
  maxPx = 8,
): Promise<void> {
  const medida = await page.evaluate(
    ({ seletor }) => {
      const raiz = document.querySelector(seletor as string)
      if (!raiz) return null
      const caixa = raiz.getBoundingClientRect()

      const temTextoProprio = (node: Element) =>
        [...node.childNodes].some((f) => f.nodeType === 3 && (f.textContent ?? '').trim() !== '')
      const eGrafico = (node: Element) =>
        ['IMG', 'SVG', 'CANVAS', 'VIDEO', 'INPUT', 'SELECT', 'TEXTAREA', 'HR'].includes(
          node.tagName.toUpperCase(),
        )

      // O recorte tem de ser feito contra o contêiner que ROLA, e não só contra
      // o palco: um item empurrado para fora de uma lista rolável continua
      // reportando a caixa dele lá embaixo, e sem este passo ele "cobre" a
      // faixa morta e a medição jura que o espaço está ocupado. Foi assim que
      // a primeira versão desta primitiva passou verde pela sabotagem.
      const janelaDoPai = (node: Element): DOMRect => {
        for (let atual = node.parentElement; atual; atual = atual.parentElement) {
          const estilo = getComputedStyle(atual)
          if (estilo.overflowY === 'auto' || estilo.overflowY === 'scroll') {
            return atual.getBoundingClientRect()
          }
          if (atual === raiz) break
        }
        return caixa
      }

      const tinta = [...raiz.querySelectorAll('*')]
        .filter((node) => temTextoProprio(node) || eGrafico(node))
        .map((node) => ({ r: node.getBoundingClientRect(), pai: janelaDoPai(node) }))
        .filter(({ r }) => r.width > 1 && r.height > 1)
        .map(({ r, pai }) => ({
          topo: Math.max(r.top, pai.top, caixa.top),
          base: Math.min(r.bottom, pai.bottom, caixa.bottom),
        }))
        .filter((f) => f.base > f.topo)
        .sort((a, b) => a.topo - b.topo)

      if (tinta.length === 0) return { faixa: Math.round(caixa.height), onde: 'o palco inteiro' }

      // Mede a sobra DEPOIS do último elemento, e só ela. Vão INTERNO não
      // entra de propósito: ele é o `gap` do arranjo, e acusá-lo seria brigar
      // com o sistema de espaçamento em vez de proteger a cena.
      const fim = tinta.reduce((maior, f) => Math.max(maior, f.base), caixa.top)
      return {
        faixa: Math.round(Math.max(0, caixa.bottom - fim)),
        onde: `de y ${Math.round(fim)} até o fim do palco em ${Math.round(caixa.bottom)}`,
      }
    },
    { seletor: palco },
  )

  expect(medida, `o palco ${palco} não existe na tela`).not.toBeNull()
  expect(
    medida?.faixa ?? 999,
    `banda vazia de ${medida?.faixa}px ${medida?.onde} numa janela de ${page.viewportSize()?.height}px — o palco não preencheu o espaço que recebeu`,
  ).toBeLessThanOrEqual(maxPx)
}

/**
 * Crescer o contêiner nunca pode CUSTAR uma coluna (ALE-172).
 *
 * A contagem de colunas não é monotônica na JANELA, e isso é correto: abaixo
 * de `lg` um catálogo fica com o palco inteiro e precisa de MAIS colunas do que
 * em `lg`, onde ele divide com um painel. Mas ela tem de ser monotônica no
 * CONTÊINER, que é o espaço que a grade de fato recebe — um gate que olha a
 * janela dá DUAS colunas num contêiner de 800px e UMA num de 968px, porque a
 * coluna de ferramentas devolve largura à direita conforme a janela encolhe.
 *
 * Varre LARGURAS com a altura FIXA, e isso é essencial: a decisão "cabe painel
 * lateral?" tem duas dimensões, então um mesmo contêiner de 812px responde
 * diferente num tablet deitado (768px de altura, cabe) e num celular deitado
 * (390px, não cabe). Comparar caixas de alturas diferentes acusaria como
 * defeito a exceção que é justamente o conserto.
 *
 * @example await expectColunasMonotonicas(page, 'section .grid', [1920, 1024, 900, 800])
 */
export async function expectColunasMonotonicas(
  page: Page,
  seletor: string,
  larguras: number[],
  altura = 900,
): Promise<void> {
  const medidas: { janela: number; conteiner: number; colunas: number }[] = []
  for (const largura of larguras) {
    await page.setViewportSize({ width: largura, height: altura })
    const medida = await page.evaluate((sel) => {
      const node = document.querySelector(sel as string)
      if (!node) return null
      return {
        conteiner: Math.round(node.getBoundingClientRect().width),
        colunas: getComputedStyle(node).gridTemplateColumns.split(' ').filter(Boolean).length,
      }
    }, seletor)
    expect(medida, `${seletor} não existe em ${largura}×${altura}`).not.toBeNull()
    if (medida) medidas.push({ janela: largura, ...medida })
  }

  const porConteiner = [...medidas].sort((a, b) => a.conteiner - b.conteiner)
  const quedas = porConteiner
    .map((atual, i) => ({ atual, antes: porConteiner[i - 1] }))
    .filter(({ atual, antes }) => antes !== undefined && atual.colunas < antes.colunas)
    .map(
      ({ atual, antes }) =>
        `contêiner de ${antes?.conteiner}px dá ${antes?.colunas} coluna(s) (janela ${antes?.janela}) e o de ${atual.conteiner}px dá ${atual.colunas} (janela ${atual.janela})`,
    )

  expect(quedas, 'crescer o contêiner custou uma coluna').toEqual([])
}

/**
 * Nada dentro da cena é CORTADO de lado (ALE-337).
 *
 * É a outra metade da pergunta do `expectNadaRolaDeLado`, e as duas juntas
 * fecham o eixo horizontal:
 *
 *   - lá, o contêiner declara `overflow-x: auto` e ROLA — o conteúdo está
 *     alcançável, e o defeito é ter de rolar de lado para alcançá-lo;
 *   - aqui, o contêiner declara `overflow-x: visible` e o conteúdo TRANSBORDA —
 *     ele não rola em lugar nenhum, e a casca (`h-dvh overflow-hidden`) o
 *     recorta em silêncio. O que passa da borda não existe para quem lê.
 *
 * O terceiro guarda do eixo, o `expectNoHorizontalOverflow`, não alcança nenhuma
 * das duas: ele lê `documentElement.scrollWidth`, e a casca garante que o
 * documento nunca cresce. Medido a 390px nos catálogos — o cartão media 352px
 * numa caixa de 308, três palavras cortadas no meio, e o documento em 390 de
 * 390.
 *
 * Ignora quem transborda DE PROPÓSITO, marcado com `data-transborda` — hoje um
 * nó só, o ponto do "Ao vivo", cujo `animate-ping` escala 1,33× e sai 1,6px de
 * cada lado. A marca existe em vez de uma folga maior porque folga esconde corte
 * de verdade: o cartão da ALE-337 passava 44px, mas o `<p>` de `/campanhas/1`
 * passava 4.
 *
 * O DENOMINADOR vem junto porque uma lista de reprovados vazia e um seletor que
 * não casa com nada se parecem no terminal.
 *
 * @example await expectNothingIsClippedSideways(page, '#catalogs')
 */
export async function expectNothingIsClippedSideways(page: Page, raiz: string): Promise<void> {
  await waitForTheSceneToSettle(page)
  const medida = await page.evaluate((seletorRaiz) => {
    const root = document.querySelector(seletorRaiz as string)
    if (!root) return null
    const nos = [...root.querySelectorAll<HTMLElement>('*')]
    const cortados = nos
      .filter((node) => {
        if (node.closest('[data-transborda]')) return false
        if (getComputedStyle(node).overflowX !== 'visible') return false
        return node.scrollWidth > node.clientWidth + 1
      })
      .map((node) => {
        const nome = node.getAttribute('aria-label') ?? node.className.slice(0, 40) ?? node.tagName
        return `${nome}: conteúdo de ${node.scrollWidth}px numa caixa de ${node.clientWidth}px`
      })
      .slice(0, 5)
    return { cortados, medidos: nos.length }
  }, raiz)

  expect(medida, `a raiz ${raiz} não existe na tela`).not.toBeNull()
  const { cortados, medidos } = medida as { cortados: string[]; medidos: number }
  expect(medidos, `a varredura não achou nó nenhum dentro de ${raiz}`).toBeGreaterThan(5)
  expect(
    cortados,
    `conteúdo cortado de lado dentro de ${raiz} @ ${page.viewportSize()?.width}px — ele não rola, então quem lê nunca o alcança`,
  ).toEqual([])
}

/**
 * Espera a cena PARAR de se mexer antes de medir caixa.
 *
 * Toda medida de geometria deste arquivo lê `getBoundingClientRect` ou
 * `scrollWidth`, e os dois respondem sobre o QUADRO ATUAL. Enquanto a entrada do
 * palco roda — `palcoEntraAdiante` e `placaSobe` —, os números são de um estado
 * intermediário que ninguém desenhou de propósito, e escolher outro instante não
 * conserta: é trocar o erro de lugar (ALE-318).
 *
 * MEDIDO, e por isso esta função existe: uma varredura de corte lateral acusou
 * quatro nós transbordando 13px em `/campanhas` e `/personagens` a 390px. Seiscentos
 * milissegundos depois eram ZERO, com zero animações rodando. Eu tinha escrito issue
 * com os dois como defeito (ALE-342) — eram o medidor lendo no meio da entrada.
 *
 * Ela ignora quem NUNCA para: `animate-pulse` e `animate-ping` têm iterações
 * infinitas, e esperar por eles seria esperar para sempre. Um transbordo causado
 * por esses continua sendo visto — a marca de isenção é outra conversa.
 */
export async function waitForTheSceneToSettle(page: Page, teto = 2_000): Promise<void> {
  await page
    .waitForFunction(
      () =>
        document.getAnimations().filter((a) => {
          if (a.playState !== 'running') return false
          return a.effect?.getComputedTiming().iterations !== Number.POSITIVE_INFINITY
        }).length === 0,
      null,
      { timeout: teto },
    )
    .catch(() => {
      // Estourar o teto NÃO é motivo para falhar: a medida segue, e se ela
      // reprovar a mensagem dela é que interessa. Falhar aqui trocaria "o cartão
      // está cortado" por "uma animação demorou", que é a limpeza falando mais
      // alto que o defeito (ALE-245).
    })
}
