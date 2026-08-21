import { type Page, expect } from '@playwright/test'

/**
 * Asserções de RELAÇÃO entre caixas (ALE-144).
 *
 * A suíte só sabia afirmar duas coisas, ambas GLOBAIS e NEGATIVAS: a página não
 * rola de lado, a página não rola para baixo. Elas só disparam quando a quebra
 * chega ao elemento raiz — e todo defeito que o dono achou por print quebrou
 * DENTRO de um contêiner. Pior: a cena é feita de `overflow-hidden` e `min-h-0`
 * postos ali justamente para a página não rolar, e cada um deles ABSORVE o
 * sintoma. Os consertos que fazem a asserção passar são os mesmos que a cegam.
 *
 * O que falta é relação: alinhamento, proporção, preenchimento, containment.
 * Cada primitiva daqui nasceu de um defeito real, e o nome dele está no
 * docstring — nenhuma foi inventada por simetria.
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
 * outra. Medido antes do conserto: "Curar" em dois X, 256 e 220.
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
 * O retrato do combatente era `w-24` de largura fixa esticado pela altura do
 * cartão: virava uma tira vertical com as iniciais perdidas no meio, e lia como
 * erro de layout em vez de retrato. Proporção é invariante — depois que alguém
 * decidiu que é um círculo, 1:1 vale para sempre.
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
 * Uma fileira de abas que para no meio do cartão deixa uma faixa vazia que o
 * olho lê como coisa quebrada, e a regra da casa é que a cena preencha o espaço
 * que recebe.
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
 * A relação é filho contra PAI, e não contra a cena: o crachá de bônus vazava
 * 6px do cartão dele e era desenhado sobre o vizinho, muito antes de chegar
 * perto da borda da coluna. A primeira versão desta asserção media contra a
 * coluna e passava VERDE sobre o defeito.
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
 * Esta é a irmã que faltava ao `expectNadaEscapa`, e a lacuna era estrutural:
 * aquela pula, DE PROPÓSITO, todo pai cujo `overflow-x` não é `visible` —
 * porque ali transbordar é a função. Só que a cena inteira é feita de
 * `overflow-x-hidden`, e foi por baixo dele que o botão "Convite" foi parar em
 * x=392 numa tela de 390: fora da tela, sem rolagem que chegasse nele, com
 * `document.scrollWidth` jurando que não havia estouro nenhum.
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
 * acontece num painel interno, a raiz continua limpa e a asserção passa: é o
 * mesmo ponto cego que já custou o `expectNadaEscapa` e o `expectDentroDaJanela`.
 *
 * O caso que trouxe esta aqui: com o tabuleiro povoado a 390px, a fileira de
 * controles do cabeçalho empurrava o ✕ de encerrar para x=466 numa tela de 390.
 * Ele não estava inalcançável — o painel rolava, `scrollWidth` 545 contra 390 de
 * largura —, mas rolar de lado para achar o botão de fechar é a experiência que
 * a regra existe para impedir. O `expectDentroDaJanela` passava verde com razão,
 * porque pela definição dele havia como chegar lá.
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
 * A regra da casa — "uma cena preenche o espaço que recebe" — nunca teve
 * asserção VERTICAL: o `expectEnchePai` mede largura, e o defeito da ALE-175
 * era altura. A lista do bestiário tinha uma tampa de `45vh` e, num tablet em
 * pé, mostrava 459px de 5216 de conteúdo com 243px mortos embaixo — um quarto
 * da tela pintada de nada, com a lista transbordando logo acima.
 *
 * Mede a banda vazia DEPOIS do último elemento do palco. Tinta é definida por
 * exclusão de contêiner: conta a caixa de quem tem texto próprio, de quem é
 * interativo e de quem é gráfico. Um `div` de arranjo não conta, porque é
 * justamente ele quem se estica por cima da faixa morta e faria a medição
 * jurar que o espaço está ocupado.
 *
 * Vão INTERNO não entra, e isso é decisão e não descuido: medida no bestiário,
 * a cena tem 22px entre dois blocos, que é o `gap` do arranjo. Acusar isso
 * seria brigar com o sistema de espaçamento; o defeito da ALE-175 nunca esteve
 * no meio, eram 243px depois do fim da lista.
 *
 * O que este guarda NÃO faz é prender altura. Altura é consequência do formato
 * e prendê-la seria prender o número errado; o que a cena promete é não deixar
 * banda vazia depois do último elemento.
 *
 * Cuidados que a medição exige, os dois aprendidos na ALE-175:
 * - o palco tem de estar TRANSBORDANDO, senão a faixa vazia é uma lista que
 *   coube e a asserção não prova nada. Quem chama afirma isso antes.
 * - ignora caixa de 1px, que é `sr-only` por definição e existe em toda tela.
 *
 * @example await expectSemFaixaMorta(page, '[aria-labelledby=mesa-bestiario]')
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

      // Mede a sobra DEPOIS do último elemento, e só ela. Vão INTERNO não entra
      // de propósito: medido nesta mesma cena, há 22px entre y 301 e 323, que
      // é o `gap` do arranjo — respiro que o design pede. Uma primitiva que o
      // acusasse brigaria com o sistema de espaçamento em vez de proteger a
      // cena, e o defeito da ALE-175 nunca esteve no meio: eram 243px depois
      // do fim da lista.
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
 * A contagem de colunas não é monotônica na JANELA — o `frontend/CLAUDE.md`
 * já avisa disso, e por bom motivo: abaixo de `lg` um catálogo fica com o
 * palco inteiro e precisa de MAIS colunas do que em `lg`, onde ele divide com
 * um painel. Mas ela tem de ser monotônica no CONTÊINER, que é o espaço que a
 * grade de fato recebe.
 *
 * O defeito que a batizou: no bestiário, um contêiner de 800px dava DUAS
 * colunas e um de 968px dava UMA, porque o gate olhava a janela e a coluna de
 * ferramentas devolve largura à direita conforme a janela encolhe.
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
