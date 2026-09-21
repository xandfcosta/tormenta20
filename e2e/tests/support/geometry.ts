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

async function medir(page: Page, selector: string): Promise<Caixa[]> {
  return page.$$eval(selector, (nodes) =>
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
export async function expectFormaColuna(page: Page, selector: string, slack = 1): Promise<void> {
  const boxes = await medir(page, selector)
  expect(boxes.length, `nenhuma caixa em ${selector} — o seletor não casou nada`).toBeGreaterThan(0)
  const columns = [...new Set(boxes.map((c) => c.x))].sort((a, b) => a - b)
  const spread = columns.length === 0 ? 0 : columns[columns.length - 1] - columns[0]
  expect(
    spread,
    `${selector} em ${columns.length} colunas (x: ${columns.join(', ')}) — deveria ser uma`,
  ).toBeLessThanOrEqual(slack)
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
  selector: string,
  min: number,
  max: number,
): Promise<void> {
  const boxes = await medir(page, selector)
  expect(boxes.length, `nenhuma caixa em ${selector}`).toBeGreaterThan(0)
  for (const rect of boxes) {
    const aspect = rect.width / rect.height
    expect(
      aspect,
      `${rect.texto || selector}: ${rect.width}×${rect.height} dá ${aspect.toFixed(2)}, fora de ${min}–${max}`,
    ).toBeGreaterThanOrEqual(min)
    expect(aspect).toBeLessThanOrEqual(max)
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
  father: string,
  child: string,
  slack = 4,
): Promise<void> {
  const spare = await page.evaluate(
    ([parentSelector, childSelector]) => {
      const parentBox = document.querySelector(parentSelector as string)
      if (!parentBox) return { erro: `pai ${parentSelector} não existe` }
      const children = [...parentBox.querySelectorAll(childSelector as string)].filter(
        (node) => node.getBoundingClientRect().width > 0,
      )
      if (children.length === 0) return { erro: `nenhum ${childSelector} dentro de ${parentSelector}` }
      const r = parentBox.getBoundingClientRect()
      const right = Math.max(...children.map((f) => f.getBoundingClientRect().right))
      // O padding do pai não é espaço morto: descontar é o que evita exigir que
      // o filho encoste na borda de um contêiner que tem respiro de propósito.
      const style = getComputedStyle(parentBox)
      const limit = r.right - Number.parseFloat(style.paddingRight || '0')
      return { sobra: Math.round(limit - right), largura: Math.round(r.width) }
    },
    [father, child],
  )
  expect(spare.erro, spare.erro ?? '').toBeUndefined()
  expect(
    spare.sobra,
    `${child} para ${spare.sobra}px antes do fim de ${father} (largura ${spare.largura}px)`,
  ).toBeLessThanOrEqual(slack)
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
export async function expectNadaEscapa(page: Page, father: string, children = '*'): Promise<void> {
  const escaping = await page.evaluate(
    ([parentSelector, childSelectors]) => {
      const root = document.querySelector(parentSelector as string)
      if (!root) return null
      return [...root.querySelectorAll(childSelectors as string)]
        .filter((node) => {
          const parentSel = node.parentElement
          if (!parentSel) return false
          const computedStyle = getComputedStyle(node)
          if (computedStyle.position === 'absolute' || computedStyle.position === 'fixed') return false
          if (getComputedStyle(parentSel).overflowX !== 'visible') return false
          const r = node.getBoundingClientRect()
          return r.width > 0 && r.right > parentSel.getBoundingClientRect().right + 1
        })
        .map((node) => (node.textContent ?? '').trim().slice(0, 30))
        .slice(0, 5)
    },
    [father, children],
  )
  expect(escaping, `o pai ${father} não existe na tela`).not.toBeNull()
  expect(escaping, `pintado para fora do pai, dentro de ${father}`).toEqual([])
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
export async function expectDentroDaJanela(page: Page, root = 'body'): Promise<void> {
  const outside = await page.evaluate((rootSelector) => {
    const root = document.querySelector(rootSelector as string)
    if (!root) return null
    const scrollable = (node: Element, axis: 'x' | 'y'): boolean => {
      for (let current: Element | null = node; current; current = current.parentElement) {
        const style = getComputedStyle(current)
        const overflow = axis === 'x' ? style.overflowX : style.overflowY
        if (overflow === 'auto' || overflow === 'scroll') return true
      }
      return false
    }
    const viewport = { largura: window.innerWidth, altura: window.innerHeight }
    return [...root.querySelectorAll('a, button, input, select, textarea, [role="button"]')]
      .filter((node) => {
        const r = node.getBoundingClientRect()
        if (r.width <= 1 || r.height <= 1) return false // sr-only e afins
        const outsideX = r.right > viewport.largura + 1 || r.left < -1
        const outsideY = r.bottom > viewport.altura + 1 || r.top < -1
        return (outsideX && !scrollable(node, 'x')) || (outsideY && !scrollable(node, 'y'))
      })
      .map((node) => {
        const r = node.getBoundingClientRect()
        const label = node.getAttribute('aria-label') ?? (node.textContent ?? '').trim().slice(0, 24)
        return `${label || node.tagName} em x ${Math.round(r.left)}–${Math.round(r.right)}, y ${Math.round(r.top)}–${Math.round(r.bottom)}`
      })
      .slice(0, 5)
  }, root)

  expect(outside, `a raiz ${root} não existe na tela`).not.toBeNull()
  expect(
    outside,
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
export async function expectNadaRolaDeLado(page: Page, root = 'body'): Promise<void> {
  const scrolling = await page.evaluate((rootSelector) => {
    const root = document.querySelector(rootSelector as string)
    if (!root) return null
    return [...root.querySelectorAll<HTMLElement>('*')]
      .filter((node) => {
        if (node.closest('[data-rola-lado]')) return false
        const style = getComputedStyle(node)
        const canScroll = style.overflowX === 'auto' || style.overflowX === 'scroll'
        return canScroll && node.scrollWidth > node.clientWidth + 1
      })
      .map((node) => {
        const label = node.getAttribute('aria-label') ?? node.className.slice(0, 40) ?? node.tagName
        return `${label}: conteúdo de ${node.scrollWidth}px numa caixa de ${node.clientWidth}px`
      })
      .slice(0, 5)
  }, root)

  expect(scrolling, `a raiz ${root} não existe na tela`).not.toBeNull()
  expect(scrolling, 'painel rolando de lado dentro da cena, que não deveria rolar').toEqual([])
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
  stage: string,
  maxPx = 8,
): Promise<void> {
  const measurement = await page.evaluate(
    ({ seletor: selector }) => {
      const root = document.querySelector(selector as string)
      if (!root) return null
      const crate = root.getBoundingClientRect()

      const hasOwnText = (node: Element) =>
        [...node.childNodes].some((f) => f.nodeType === 3 && (f.textContent ?? '').trim() !== '')
      const isGraphic = (node: Element) =>
        ['IMG', 'SVG', 'CANVAS', 'VIDEO', 'INPUT', 'SELECT', 'TEXTAREA', 'HR'].includes(
          node.tagName.toUpperCase(),
        )

      // O recorte tem de ser feito contra o contêiner que ROLA, e não só contra
      // o palco: um item empurrado para fora de uma lista rolável continua
      // reportando a caixa dele lá embaixo, e sem este passo ele "cobre" a
      // faixa morta e a medição jura que o espaço está ocupado. Foi assim que
      // a primeira versão desta primitiva passou verde pela sabotagem.
      const parentWindow = (node: Element): DOMRect => {
        for (let current = node.parentElement; current; current = current.parentElement) {
          const style = getComputedStyle(current)
          if (style.overflowY === 'auto' || style.overflowY === 'scroll') {
            return current.getBoundingClientRect()
          }
          if (current === root) break
        }
        return crate
      }

      const paint = [...root.querySelectorAll('*')]
        .filter((node) => hasOwnText(node) || isGraphic(node))
        .map((node) => ({ r: node.getBoundingClientRect(), pai: parentWindow(node) }))
        .filter(({ r }) => r.width > 1 && r.height > 1)
        .map(({ r, pai: father }) => ({
          topo: Math.max(r.top, father.top, crate.top),
          base: Math.min(r.bottom, father.bottom, crate.bottom),
        }))
        .filter((f) => f.base > f.topo)
        .sort((a, b) => a.topo - b.topo)

      if (paint.length === 0) return { faixa: Math.round(crate.height), onde: 'o palco inteiro' }

      // Mede a sobra DEPOIS do último elemento, e só ela. Vão INTERNO não
      // entra de propósito: ele é o `gap` do arranjo, e acusá-lo seria brigar
      // com o sistema de espaçamento em vez de proteger a cena.
      const end = paint.reduce((max, f) => Math.max(max, f.base), crate.top)
      return {
        faixa: Math.round(Math.max(0, crate.bottom - end)),
        onde: `de y ${Math.round(end)} até o fim do palco em ${Math.round(crate.bottom)}`,
      }
    },
    { seletor: stage },
  )

  expect(measurement, `o palco ${stage} não existe na tela`).not.toBeNull()
  expect(
    measurement?.faixa ?? 999,
    `banda vazia de ${measurement?.faixa}px ${measurement?.onde} numa janela de ${page.viewportSize()?.height}px — o palco não preencheu o espaço que recebeu`,
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
  selector: string,
  widths: number[],
  height = 900,
): Promise<void> {
  const measured: { janela: number; conteiner: number; colunas: number }[] = []
  for (const width of widths) {
    await page.setViewportSize({ width: width, height: height })
    const measure = await page.evaluate((sel) => {
      const node = document.querySelector(sel as string)
      if (!node) return null
      return {
        conteiner: Math.round(node.getBoundingClientRect().width),
        colunas: getComputedStyle(node).gridTemplateColumns.split(' ').filter(Boolean).length,
      }
    }, selector)
    expect(measure, `${selector} não existe em ${width}×${height}`).not.toBeNull()
    if (measure) measured.push({ janela: width, ...measure })
  }

  const byContainer = [...measured].sort((a, b) => a.conteiner - b.conteiner)
  const drops = byContainer
    .map((current, i) => ({ atual: current, antes: byContainer[i - 1] }))
    .filter(({ atual: current, antes: before }) => before !== undefined && current.colunas < before.colunas)
    .map(
      ({ atual: current, antes: before }) =>
        `contêiner de ${before?.conteiner}px dá ${before?.colunas} coluna(s) (janela ${before?.janela}) e o de ${current.conteiner}px dá ${current.colunas} (janela ${current.janela})`,
    )

  expect(drops, 'crescer o contêiner custou uma coluna').toEqual([])
}

/**
 * Espera as animações em curso terminarem, e é obrigatório antes de medir
 * LARGURA.
 *
 * O palco entra deslizando: o cartão que chega passa alguns quadros FORA da
 * caixa, e nesses quadros o `scrollWidth` acusa 403px numa caixa de 390px. Sem
 * esta espera, a varredura das cenas reprovava `/personagens` e `/campanhas` —
 * e o que ela estava medindo era a animação, não o leiaute (ALE-342).
 *
 * `getAnimations()` e não um `waitForTimeout` escolhido a dedo: ele responde SE
 * existe transição, enquanto um tempo fixo é um palpite que acerta na máquina de
 * quem escreveu. O teto existe para uma animação INFINITA — o ponto do "AO VIVO"
 * pulsa para sempre — não pendurar a suíte.
 */
async function esperaAsAnimacoesPararem(page: Page): Promise<void> {
  await page
    .waitForFunction(() => document.getAnimations().every((a) => a.playState !== 'running'), null, {
      timeout: 3000,
    })
    .catch(() => {
      /* animação infinita: seguir e medir é melhor que não medir */
    })
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
 * O DENOMINADOR vem junto porque uma lista de reprovados vazia e um seletor que
 * não casa com nada se parecem no terminal.
 *
 * @example await expectNothingIsClippedSideways(page, '#catalogs')
 */
export async function expectNothingIsClippedSideways(page: Page, root: string): Promise<void> {
  const { cortados: clipped, medidos: measured } = await measureSidewaysClipping(page, root)
  expect(measured, `a varredura não achou nó nenhum dentro de ${root}`).toBeGreaterThan(5)
  expect(
    clipped,
    `conteúdo cortado de lado dentro de ${root} @ ${page.viewportSize()?.width}px — ele não rola, então quem lê nunca o alcança`,
  ).toEqual([])
}

/**
 * measureSidewaysClipping é a MEDIÇÃO sozinha, sem asserção.
 *
 * Ela existe separada porque há dois leitores com perguntas opostas: o
 * `expectNothingIsClippedSideways` afirma que a lista está vazia, e o caso da
 * face mais larga afirma, para a dívida registrada, que ela NÃO está — é a
 * segunda direção da catraca, a que faz a dívida encolher.
 */
export async function measureSidewaysClipping(
  page: Page,
  root: string,
): Promise<{ cortados: string[]; medidos: number }> {
  await esperaAsAnimacoesPararem(page)
  const measure = await page.evaluate((rootSelector) => {
    const root = document.querySelector(rootSelector as string)
    if (!root) return null
    const nodes = [...root.querySelectorAll<HTMLElement>('*')]
    const clipped = nodes
      .filter((node) => {
        if (getComputedStyle(node).overflowX !== 'visible') return false
        if (node.scrollWidth <= node.clientWidth + 1) return false
        // CAIXA QUE ESTÁ ANIMANDO não tem largura para medir: o ponto do "Ao
        // vivo" pulsa com `animate-ping` para sempre, e no auge do pulso o
        // filho tem 12px numa caixa de 10. A largura dele é um instante, não um
        // leiaute — e a espera lá em cima não o alcança porque ele nunca para.
        return node.getAnimations({ subtree: true }).every((a) => a.playState !== 'running')
      })
      .map((node) => {
        const label = node.getAttribute('aria-label') ?? node.className.slice(0, 40) ?? node.tagName
        return `${label}: conteúdo de ${node.scrollWidth}px numa caixa de ${node.clientWidth}px`
      })
      .slice(0, 5)
    return { cortados: clipped, medidos: nodes.length }
  }, root)

  expect(measure, `a raiz ${root} não existe na tela`).not.toBeNull()
  return measure as { cortados: string[]; medidos: number }
}

/**
 * Um CONTROLE DE ESTADO tem de DESENHAR o estado que ele declara.
 *
 * ## O defeito, e por que ele é invisível para quem escreve
 *
 * Um `role="switch"` ou um `aria-pressed` diz ao leitor de tela se está ligado.
 * Se o visual não acompanhar, a tela fica correta para quem ouve e MUDA para
 * quem olha: o mestre clica, nada muda, e ele clica de novo — desligando o que
 * tinha acabado de ligar (ALE-341).
 *
 * ## Por que isto vive no NAVEGADOR
 *
 * Porque "desenha estado" tem pelo menos cinco mecanismos, e nenhum instrumento
 * de FONTE os vê todos: `templ.KV` na classe, `data-class` ligado a sinal, a
 * variante `aria-pressed:` do Tailwind, troca de ÍCONE dentro do elemento, e
 * regra de folha de estilo casando o próprio atributo ARIA.
 *
 * Medido ao abrir a ALE-341: três varreduras de fonte deram 18, 14 e 10
 * suspeitos, e o navegador derrubou os primeiros que ele foi conferir. É a
 * lição da ALE-252 — quem mede pelo TEXTO do código mede o que foi escrito, e a
 * pergunta é sobre o que é DESENHADO.
 *
 * ## O que ele compara
 *
 * O estilo COMPUTADO e o HTML de dentro, antes e depois do clique. Qualquer um
 * dos dois mudando basta: trocar o ícone é desenhar estado tanto quanto trocar
 * a cor.
 *
 * ## O CONTROLE, e ele é obrigatório
 *
 * Um controle que o clique NÃO liga não testemunha nada — a asserção passaria
 * sobre um botão quebrado. Por isso o ARIA tem de MUDAR primeiro; quando ele
 * não muda, o caso falha dizendo isso, e não "o visual está mudo".
 */
export async function expectEveryToggleDrawsItsState(page: Page, root: string): Promise<void> {
  const controls = page.locator(`${root} [role="switch"], ${root} [aria-pressed]`)
  const howMany = await controls.count()
  expect(howMany, `nenhum controle de estado dentro de ${root} — a varredura mediria o vazio`).toBeGreaterThan(0)

  const silent: string[] = []
  let measured = 0
  for (let i = 0; i < howMany; i++) {
    const control = controls.nth(i)
    if (!(await control.isVisible())) continue
    const before = await retratoDo(control)
    await control.click({ force: true })
    await page.waitForTimeout(350)
    const after = await retratoDo(control).catch(() => null)
    // O nó pode SUMIR no clique (um filtro que recarrega a lista). Aí não há o
    // que comparar, e contá-lo como mudo seria acusar o que não se mediu.
    if (after === null) continue
    measured++
    if (before.aria === after.aria) continue // o clique não ligou nada: ver o CONTROLE acima
    if (before.estilo === after.estilo && before.html === after.html) {
      silent.push(`${after.nome}: aria foi de ${before.aria} para ${after.aria} e nada mudou na tela`)
    }
    await control.click({ force: true }).catch(() => {})
    await page.waitForTimeout(250)
  }

  expect(measured, `nenhum controle de ${root} respondeu ao clique — o seletor casa, mas o gesto não chega`).toBeGreaterThan(0)
  expect(
    silent,
    `controles que declaram estado por ARIA e não o DESENHAM, em ${root} — ` +
      `a tela está certa para quem ouve e muda para quem olha`,
  ).toEqual([])
}

type RetratoDoControle = { nome: string; estilo: string; html: string; aria: string }

async function retratoDo(control: ReturnType<Page['locator']>): Promise<RetratoDoControle> {
  return control.evaluate((n) => {
    const cs = getComputedStyle(n as HTMLElement)
    const painting = [
      'backgroundColor', 'color', 'borderColor', 'borderWidth', 'borderStyle',
      'opacity', 'boxShadow', 'fontWeight', 'textDecorationLine', 'outlineColor',
    ] as const
    return {
      nome: n.getAttribute('aria-label') ?? n.textContent?.trim().slice(0, 40) ?? n.tagName,
      estilo: painting.map((k) => cs[k]).join('|'),
      html: (n as HTMLElement).innerHTML,
      aria: n.getAttribute('aria-checked') ?? n.getAttribute('aria-pressed') ?? '',
    }
  })
}
