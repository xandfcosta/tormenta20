import { describe, expect, it } from 'vitest'
import { getCatalogItem, type ItemFlag } from '@tormenta20/t20-data'
import {
  describeModifierTarget,
  itemOverlayCatalogs,
  itemOverlayNames,
  loadLimitLabel,
  overlayNotesSummary,
} from './item-describe'

// Regression (bug B): flag slugs used to render raw ("Efeito: fatigue-on-sleep").
describe('describeModifierTarget — flag targets', () => {
  const cases: [ItemFlag, string][] = [
    ['lethal-unarmed', 'Ataques desarmados causam dano letal'],
    ['cannot-apply-dex-to-defense', 'Não soma Destreza na Defesa'],
    ['fatigue-on-sleep', 'Fadiga ao dormir'],
    ['reach-extends', 'Alcance ampliado'],
    ['armadura-pesada', 'Conta como armadura pesada'],
  ]

  it.each(cases)('labels %s in PT-BR', (flag, label) => {
    expect(describeModifierTarget({ k: 'flag', name: flag })).toBe(label)
  })

  it('never leaks the raw slug', () => {
    for (const [flag] of cases) {
      const label = describeModifierTarget({ k: 'flag', name: flag })
      expect(label).not.toContain(flag)
      expect(label).not.toContain('Efeito:')
    }
  })
})

describe('loadLimitLabel', () => {
  // Regression (bug F): the caption must show resolved values, never the
  // math placeholder '10 + 2×|FOR|'.
  it('renders the resolved limit and a signed For value', () => {
    expect(loadLimitLabel(18, 4)).toBe('limite 18 · 10 + 2×For +4')
  })

  it('keeps the sign on a negative For', () => {
    expect(loadLimitLabel(8, -2)).toBe('limite 8 · 10 + 2×For -2')
  })
})

describe('overlayNotesSummary', () => {
  // Regressão: Equilibrada tem 4 modifiers de manobra com a MESMA note —
  // o join cru mostrava "+2 em manobras" quatro vezes no picker.
  it('deduplica notes repetidas', () => {
    const equilibrada = getCatalogItem('melhoria-equilibrada')!
    expect(overlayNotesSummary(equilibrada.modifiers)).toBe('+2 em manobras')
  })

  it('mantém notes distintas separadas por vírgula', () => {
    expect(
      overlayNotesSummary([
        { target: { k: 'defense' }, amount: 1, bonusType: 'untyped', note: 'a' },
        { target: { k: 'defense' }, amount: 2, bonusType: 'untyped', note: 'b' },
      ]),
    ).toBe('a, b')
  })
})

describe('itemOverlayNames', () => {
  it('resolve melhorias + material para nomes', () => {
    expect(
      itemOverlayNames({
        improvements: JSON.stringify(['melhoria-reforcada', 'melhoria-vigilante']),
        material: 'material-aco-rubi',
      }),
    ).toEqual(['Reforçada', 'Vigilante', 'Aço-rubi'])
  })

  it('ignora ids desconhecidos e blob inválido', () => {
    expect(
      itemOverlayNames({ improvements: 'not json', material: null }),
    ).toEqual([])
    expect(
      itemOverlayNames({
        improvements: JSON.stringify(['nao-existe']),
        material: null,
      }),
    ).toEqual([])
  })
})

describe('itemOverlayCatalogs', () => {
  it('resolve entradas completas do catálogo (com modifiers)', () => {
    const overlays = itemOverlayCatalogs({
      improvements: JSON.stringify(['melhoria-vigilante']),
      material: null,
    })
    expect(overlays).toHaveLength(1)
    expect(overlays[0].name).toBe('Vigilante')
    expect(overlays[0].modifiers[0].target).toEqual({ k: 'defense' })
  })
})
