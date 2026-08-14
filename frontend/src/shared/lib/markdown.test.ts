import { describe, expect, it } from 'vitest'
import { parseInline, parseMarkdown } from './markdown'

describe('parseMarkdown', () => {
  it('separa parágrafos por linha em branco e junta as linhas de um', () => {
    expect(parseMarkdown('o ogro fugiu\npela ponte\n\ne voltou')).toEqual([
      { kind: 'paragraph', spans: [{ kind: 'text', text: 'o ogro fugiu pela ponte' }] },
      { kind: 'paragraph', spans: [{ kind: 'text', text: 'e voltou' }] },
    ])
  })

  it('lê títulos de um a três', () => {
    const blocks = parseMarkdown('# Cena 1\n### Detalhe')
    expect(blocks.map((b) => b.kind === 'heading' && b.level)).toEqual([1, 3])
  })

  // Itens seguidos são UMA lista: um bloco por linha perderia a marcação.
  it('junta as linhas seguidas numa lista só', () => {
    const [block] = parseMarkdown('- ogro\n- goblin\n- zumbi')
    expect(block).toEqual({
      kind: 'list',
      ordered: false,
      items: [
        [{ kind: 'text', text: 'ogro' }],
        [{ kind: 'text', text: 'goblin' }],
        [{ kind: 'text', text: 'zumbi' }],
      ],
    })
  })

  it('distingue lista numerada', () => {
    const [block] = parseMarkdown('1. primeiro\n2. segundo')
    expect(block.kind === 'list' && block.ordered).toBe(true)
  })

  it('lê citação e linha divisória', () => {
    expect(parseMarkdown('> o mestre disse\n\n---').map((b) => b.kind)).toEqual(['quote', 'rule'])
  })

  // O que o subconjunto não conhece tem de SOBREVIVER como texto: comer o que
  // o mestre escreveu no meio da sessão é o pior desfecho possível.
  it('deixa o que não reconhece passar como texto', () => {
    expect(parseMarkdown('| tabela | não |')).toEqual([
      { kind: 'paragraph', spans: [{ kind: 'text', text: '| tabela | não |' }] },
    ])
  })
})

describe('parseInline', () => {
  it('lê negrito, itálico e código', () => {
    expect(parseInline('o **ogro** ficou *bravo* e usou `1d8+3`')).toEqual([
      { kind: 'text', text: 'o ' },
      { kind: 'strong', text: 'ogro' },
      { kind: 'text', text: ' ficou ' },
      { kind: 'em', text: 'bravo' },
      { kind: 'text', text: ' e usou ' },
      { kind: 'code', text: '1d8+3' },
    ])
  })

  it('lê link http', () => {
    expect(parseInline('ver [o mapa](https://exemplo.com/mapa)')).toEqual([
      { kind: 'text', text: 'ver ' },
      { kind: 'link', text: 'o mapa', href: 'https://exemplo.com/mapa' },
    ])
  })

  // A árvore nunca vira HTML, então injeção já é impossível — mas um link
  // navegável para `javascript:` não pode existir nem assim.
  it('um link que não é http vira texto', () => {
    expect(parseInline('[clique](javascript:alert(1))')).toEqual([
      { kind: 'text', text: '[clique](javascript:alert(1))' },
    ])
  })
})
