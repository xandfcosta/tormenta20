/**
 * Um markdown PEQUENO, do tamanho de uma nota de mesa (ALE-122).
 *
 * Ele produz uma árvore, nunca HTML: quem renderiza monta elementos Solid a
 * partir daqui, então não existe `innerHTML` no caminho e injeção é impossível
 * por construção — sem parser de terceiro e sem sanitizador atrás dele.
 *
 * O subconjunto é o que um mestre escreve durante a sessão: títulos, listas,
 * citação, negrito, itálico, código e link. O que não está aqui aparece como
 * texto, que é a falha certa para uma nota — nunca comer o que foi escrito.
 *
 * @example parseMarkdown('# Cena 1\n- Ogro **fugiu**')
 */

export type Inline =
  | { kind: 'text'; text: string }
  | { kind: 'strong'; text: string }
  | { kind: 'em'; text: string }
  | { kind: 'code'; text: string }
  | { kind: 'link'; text: string; href: string }

export type Block =
  | { kind: 'heading'; level: 1 | 2 | 3; spans: Inline[] }
  | { kind: 'paragraph'; spans: Inline[] }
  | { kind: 'list'; ordered: boolean; items: Inline[][] }
  | { kind: 'quote'; spans: Inline[] }
  | { kind: 'rule' }

const HEADING = /^(#{1,3})\s+(.*)$/
const BULLET = /^[-*]\s+(.*)$/
const ORDERED = /^\d+[.)]\s+(.*)$/
const QUOTE = /^>\s?(.*)$/
const RULE = /^(-{3,}|\*{3,})$/

export function parseMarkdown(source: string): Block[] {
  const blocks: Block[] = []
  let paragraph: string[] = []

  const flushParagraph = () => {
    if (paragraph.length === 0) return
    blocks.push({ kind: 'paragraph', spans: parseInline(paragraph.join(' ')) })
    paragraph = []
  }

  const lines = source.replace(/\r\n/g, '\n').split('\n')
  for (let index = 0; index < lines.length; index++) {
    const line = lines[index].trim()
    if (line === '') {
      flushParagraph()
      continue
    }
    const heading = HEADING.exec(line)
    if (heading) {
      flushParagraph()
      blocks.push({
        kind: 'heading',
        level: heading[1].length as 1 | 2 | 3,
        spans: parseInline(heading[2]),
      })
      continue
    }
    if (RULE.test(line)) {
      flushParagraph()
      blocks.push({ kind: 'rule' })
      continue
    }
    if (BULLET.test(line) || ORDERED.test(line)) {
      flushParagraph()
      index = collectList(lines, index, blocks) // consome a lista inteira
      continue
    }
    const quote = QUOTE.exec(line)
    if (quote) {
      flushParagraph()
      blocks.push({ kind: 'quote', spans: parseInline(quote[1]) })
      continue
    }
    paragraph.push(line)
  }
  flushParagraph()
  return blocks
}

/**
 * Junta as linhas seguidas de uma lista num bloco só e devolve o índice da
 * última consumida — itens soltos viravam um bloco por linha, e a marcação de
 * lista se perdia.
 */
function collectList(lines: string[], start: number, blocks: Block[]): number {
  const ordered = ORDERED.test(lines[start].trim())
  const items: Inline[][] = []
  let index = start
  for (; index < lines.length; index++) {
    const line = lines[index].trim()
    const match = ordered ? ORDERED.exec(line) : BULLET.exec(line)
    if (!match) break
    items.push(parseInline(match[1]))
  }
  blocks.push({ kind: 'list', ordered, items })
  return index - 1
}

const INLINE = /(`[^`]+`)|(\*\*[^*]+\*\*)|(\*[^*]+\*|_[^_]+_)|(\[[^\]]+\]\([^)]+\))/

/** Quebra uma linha nos trechos marcados, deixando o resto como texto. */
export function parseInline(text: string): Inline[] {
  const spans: Inline[] = []
  let rest = text
  while (rest.length > 0) {
    const match = INLINE.exec(rest)
    if (!match || match.index === undefined) break
    if (match.index > 0) spans.push({ kind: 'text', text: rest.slice(0, match.index) })
    spans.push(toSpan(match[0]))
    rest = rest.slice(match.index + match[0].length)
  }
  if (rest.length > 0) spans.push({ kind: 'text', text: rest })
  return mergeText(spans)
}

/**
 * Junta textos vizinhos. Um trecho recusado (um link que não é http, um
 * parêntese fechando cedo dentro da URL) sai partido em dois pedaços de texto,
 * e o que o mestre escreveu tem de voltar inteiro.
 */
function mergeText(spans: Inline[]): Inline[] {
  return spans.reduce<Inline[]>((acc, span) => {
    const last = acc.at(-1)
    if (span.kind === 'text' && last?.kind === 'text') {
      acc[acc.length - 1] = { kind: 'text', text: last.text + span.text }
      return acc
    }
    acc.push(span)
    return acc
  }, [])
}

function toSpan(token: string): Inline {
  if (token.startsWith('`')) return { kind: 'code', text: token.slice(1, -1) }
  if (token.startsWith('**')) return { kind: 'strong', text: token.slice(2, -2) }
  if (token.startsWith('[')) return toLink(token)
  return { kind: 'em', text: token.slice(1, -1) }
}

/**
 * Só http(s). Um `javascript:` num link vira TEXTO, não um link morto: quem
 * escreveu vê o que escreveu, e nada navegável sai daqui.
 */
function toLink(token: string): Inline {
  const [, text, href] = /^\[([^\]]+)\]\(([^)]+)\)$/.exec(token) ?? []
  if (!text || !href) return { kind: 'text', text: token }
  return /^https?:\/\//i.test(href) ? { kind: 'link', text, href } : { kind: 'text', text: token }
}
