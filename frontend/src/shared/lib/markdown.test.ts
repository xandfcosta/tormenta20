import { describe, expect, it } from 'vitest'
import { parseInline, parseMarkdown, toggleTaskLine } from './markdown'

describe('parseMarkdown', () => {
  // A quebra que o mestre digitou é a quebra que ele vê: o markdown padrão
  // junta linhas seguidas num parágrafo só, e trinta linhas de anotação viravam
  // uma frase corrida.
  it('preserva a quebra de linha dentro do parágrafo', () => {
    expect(parseMarkdown('o ogro fugiu\npela ponte\n\ne voltou')).toEqual([
      {
        kind: 'paragraph',
        lines: [[{ kind: 'text', text: 'o ogro fugiu' }], [{ kind: 'text', text: 'pela ponte' }]],
      },
      { kind: 'paragraph', lines: [[{ kind: 'text', text: 'e voltou' }]] },
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
        { spans: [{ kind: 'text', text: 'ogro' }] },
        { spans: [{ kind: 'text', text: 'goblin' }] },
        { spans: [{ kind: 'text', text: 'zumbi' }] },
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
      { kind: 'paragraph', lines: [[{ kind: 'text', text: '| tabela | não |' }]] },
    ])
  })

  // A tarefa carrega a LINHA da origem: é por ela que marcar o checkbox
  // reescreve a nota, em vez de guardar o estado ao lado dela.
  it('lê tarefa marcada e desmarcada com a linha de origem', () => {
    const [block] = parseMarkdown('- [ ] dar XP\n- [x] anotar tesouro')
    expect(block).toEqual({
      kind: 'list',
      ordered: false,
      items: [
        { spans: [{ kind: 'text', text: 'dar XP' }], task: { checked: false, line: 0 } },
        { spans: [{ kind: 'text', text: 'anotar tesouro' }], task: { checked: true, line: 1 } },
      ],
    })
  })
})

describe('toggleTaskLine', () => {
  it('marca e desmarca a linha pedida, sem tocar nas outras', () => {
    const source = '# Cena\n- [ ] dar XP\n- [x] tesouro'
    expect(toggleTaskLine(source, 1, true)).toBe('# Cena\n- [x] dar XP\n- [x] tesouro')
    expect(toggleTaskLine(source, 2, false)).toBe('# Cena\n- [ ] dar XP\n- [ ] tesouro')
  })

  it('devolve o texto intacto quando a linha não é uma tarefa', () => {
    expect(toggleTaskLine('só texto', 0, true)).toBe('só texto')
    expect(toggleTaskLine('- [ ] dar XP', 9, true)).toBe('- [ ] dar XP')
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
