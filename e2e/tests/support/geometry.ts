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
