import { expect, type Page } from '@playwright/test'

/**
 * O TEXTO É DESENHADO COM A FACE QUE O APP ENTREGA (ALE-362).
 *
 * # Por que só um navegador responde isto
 *
 * O guarda em Go (`TestEveryFontTokenNamesAFaceTheAppShips`) prende a DECISÃO:
 * a folha declara a família e a põe na frente da pilha. Ele não prova nada sobre
 * o que foi DESENHADO — um `.woff2` com caminho errado, um 404, um subconjunto
 * corrompido ou uma CSP no caminho terminam todos na fonte da máquina, com o
 * teste de texto verde. É a mesma família do "artefato guarda a fonte quebrada".
 *
 * Quem responde é o CDP: `CSS.getPlatformFontsForNode` devolve a face que o
 * motor de fato usou, com `isCustomFont` dizendo se ela veio do app ou do
 * sistema operacional. Não há equivalente em jsdom, e não há como inferir isso
 * medindo largura — que é justamente a medida que esta garantia protege.
 *
 * # Por que isto importa mais do que parece
 *
 * Com a face vindo da máquina, TODA medida de largura deste repositório — corte
 * lateral, piso de toque, decomposição de camadas — é uma propriedade do
 * computador que rodou a suíte. A `main` passou três merges vermelha na CI com a
 * bancada verde por exatamente isso.
 *
 * # O que ele pula, e por que a regra do pulo não é uma lista
 *
 * As faces entregues cobrem latin-1, latin ext-A e a pontuação geral. A casa
 * escreve dezessete símbolos fora disso (`✕`, `⏎`, `▦`, `⌘`…), e esses caem
 * numa face do sistema por falta de glifo — não por a folha pedir errado.
 * Cobrá-los aqui seria cobrar uma fatia inteira de troca de símbolo por ícone.
 *
 * Então o pulo é MECÂNICO e não uma lista de casos: um texto entra na medição
 * quando TODA runa dele está nas faixas entregues. O contador dos pulados volta
 * junto, para "nada reprovou" nunca se confundir com "nada foi medido".
 */

// AS FAIXAS QUE O SUBCONJUNTO ENTREGA, e elas são as mesmas do `pyftsubset` que
// gerou os `.woff2`. Duas cópias de um número são duas velocidades de
// envelhecimento; o que as amarra é este comentário e o guarda em Go, que falha
// se a família sair da frente da pilha.
const SHIPPED_RANGES: Array<[number, number]> = [
  [0x0000, 0x00ff],
  [0x0100, 0x017f],
  [0x2000, 0x206f],
]

function isShipped(text: string): boolean {
  for (const rune of text) {
    const cp = rune.codePointAt(0) as number
    if (!SHIPPED_RANGES.some(([lo, hi]) => cp >= lo && cp <= hi)) return false
  }
  return true
}

type FontReport = { falhas: string[]; medidos: number; pulados: number }

/**
 * measureDrawnFaces percorre os nós de texto da página e devolve os que foram
 * desenhados com uma face do SISTEMA.
 *
 * @example const { falhas, medidos } = await measureDrawnFaces(page)
 */
export async function measureDrawnFaces(page: Page, ceiling = 30): Promise<FontReport> {
  const cdp = await page.context().newCDPSession(page)
  try {
    await cdp.send('DOM.enable')
    await cdp.send('CSS.enable')
    const { root } = await cdp.send('DOM.getDocument', { depth: -1 })

    const candidates: Array<{ nodeId: number; texto: string }> = []
    let skipped = 0
    // A SUBÁRVORE INTEIRA decide, e não o texto direto do nó.
    //
    // O `getPlatformFontsForNode` responde sobre tudo que o nó DESENHA, filhos
    // inclusive. Medido: o cartão "Abrir campanha" reprovava com a pilha certa
    // na regra, porque o `⏎` do atalho, dentro dele, caía numa monoespaçada do
    // sistema por falta de glifo. Olhar só o texto direto lê a face do símbolo
    // como se fosse a do rótulo (ALE-362).
    const visit = (node: { nodeId: number; nodeType: number; nodeValue?: string; children?: unknown[] }): string => {
      if (node.nodeType === 3) return node.nodeValue ?? ''
      let subtree = ''
      let directText = ''
      for (const raw of node.children ?? []) {
        const child = raw as { nodeId: number; nodeType: number; nodeValue?: string; children?: unknown[] }
        const chunk = visit(child)
        subtree += chunk
        if (child.nodeType === 3) directText += chunk
      }
      if (directText.trim().length > 0) {
        if (isShipped(subtree.trim())) candidates.push({ nodeId: node.nodeId, texto: directText.trim() })
        else skipped++
      }
      return subtree
    }
    visit(root as unknown as { nodeId: number; nodeType: number; children?: unknown[] })

    const failures: string[] = []
    const measured = Math.min(candidates.length, ceiling)
    for (const { nodeId, texto: text } of candidates.slice(0, ceiling)) {
      const { fonts } = await cdp.send('CSS.getPlatformFontsForNode', { nodeId })
      for (const face of fonts) {
        if (face.isCustomFont) continue
        // A PILHA PEDIDA entra na mensagem, e ela é a diferença entre "conserte
        // isto" e "procure": a face desenhada diz que deu errado, e só a pilha
        // diz QUAL regra de CSS pediu a fonte de fora.
        const { computedStyle } = await cdp.send('CSS.getComputedStyleForNode', { nodeId })
        const requested = computedStyle.find((p) => p.name === 'font-family')?.value ?? '?'
        failures.push(
          `"${text.slice(0, 24)}" foi desenhado com ${face.familyName}, que veio da MÁQUINA — a regra pediu \`${requested}\``,
        )
      }
    }
    return { falhas: failures, medidos: measured, pulados: skipped }
  } finally {
    // Limpeza com `catch`: um erro ao desligar a sessão substituiria a falha de
    // verdade pela falha do desligamento (ALE-245).
    await cdp.detach().catch(() => {})
  }
}

export async function expectTextIsDrawnWithAShippedFace(page: Page, where: string): Promise<void> {
  const { falhas: failures, medidos: measured, pulados: skipped } = await measureDrawnFaces(page)
  expect(
    measured,
    `${where}: o medidor não achou texto nenhum coberto pelo subconjunto entregue (${skipped} pulados) — a cena não carregou, e a asserção seguinte não seria evidência de nada`,
  ).toBeGreaterThan(3)
  expect(
    failures,
    `${where}: texto desenhado com fonte do SISTEMA. Largura medida com face de fora é propriedade da máquina — a bancada e a CI passam a medir telas diferentes. Confira se o \`.woff2\` está sendo servido em /fonts/ e se a família está na frente de \`--font-sans\``,
  ).toEqual([])
}
