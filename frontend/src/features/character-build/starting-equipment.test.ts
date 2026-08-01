import { describe, expect, it } from 'vitest'
import {
  lightArmorOptions,
  originStartingItems,
  startingItemsPayload,
  startingLoadout,
  weaponOptions,
} from './starting-equipment'

describe('startingLoadout — kit p140 + Tabela 3-1', () => {
  it('Guerreiro: simples+marcial, brunea, escudo', () => {
    const { kit } = startingLoadout('Guerreiro', 1)
    expect(kit.weapons).toBe('simples+marcial')
    expect(kit.armor).toBe('brunea')
    expect(kit.shieldLeve).toBe(true)
  })

  it('Arcanista: sem armadura, só simples', () => {
    const { kit } = startingLoadout('Arcanista', 1)
    expect(kit.armor).toBe('nenhuma')
    expect(kit.weapons).toBe('simples')
    expect(kit.shieldLeve).toBe(false)
  })

  it('dinheiro: Nv 1 → null (4d6); Nv 6 → 3000', () => {
    expect(startingLoadout('Guerreiro', 1).tableMoney).toBeNull()
    expect(startingLoadout('Guerreiro', 6).tableMoney).toBe(3_000)
  })
})

describe('weapon/armor pools from the catalog', () => {
  it('armas por categoria', () => {
    expect(weaponOptions('weapon-simple').some((w) => w.id === 'adaga')).toBe(true)
    expect(
      weaponOptions('weapon-martial').some((w) => w.id === 'espada-longa'),
    ).toBe(true)
  })

  it('3 armaduras leves (p140: couro, couro batido, gibão de peles)', () => {
    expect(lightArmorOptions().map((a) => a.id)).toEqual([
      'armadura-couro',
      'couro-batido',
      'gibao-peles',
    ])
  })
})

describe('originStartingItems — itens da origem por nome', () => {
  it('Acólito recebe símbolo sagrado + traje', () => {
    expect(originStartingItems('Acólito')).toEqual([
      'Símbolo sagrado',
      'Traje de sacerdote',
    ])
  })

  it('origem desconhecida → vazio', () => {
    expect(originStartingItems('Não Existe')).toEqual([])
  })
})

describe('startingItemsPayload — payload de criação', () => {
  const kit = startingLoadout('Guerreiro', 1).kit

  it('base + escolhas equipadas + itens da origem', () => {
    const items = startingItemsPayload(
      {
        weaponSimple: 'espada-curta',
        weaponMartial: 'espada-longa',
        armor: 'brunea',
        shield: true,
      },
      kit,
      'Acólito',
    )
    const names = items.map((i) => i.name)
    expect(names).toContain('Mochila')
    expect(names).toContain('Espada curta')
    expect(names).toContain('Espada longa')
    expect(names).toContain('Brunea')
    expect(names).toContain('Escudo leve')
    expect(names).toContain('Símbolo sagrado')
    const brunea = items.find((i) => i.catalogId === 'brunea')
    expect(brunea?.equipped).toBe('vested')
    const sword = items.find((i) => i.catalogId === 'espada-longa')
    expect(sword?.equipped).toBe('wielded')
  })

  it('escolhas vazias são puladas (sub-preenchimento permitido)', () => {
    const items = startingItemsPayload(
      { weaponSimple: '', weaponMartial: '', armor: '', shield: false },
      kit,
      '',
    )
    expect(items).toHaveLength(3) // só o kit base
  })

  it('Arcanista nunca recebe armadura mesmo com pick residual', () => {
    const arcKit = startingLoadout('Arcanista', 1).kit
    const items = startingItemsPayload(
      { weaponSimple: '', weaponMartial: '', armor: 'brunea', shield: true },
      arcKit,
      '',
    )
    expect(items.some((i) => i.catalogId === 'brunea')).toBe(false)
    expect(items.some((i) => i.catalogId === 'escudo-leve')).toBe(false)
  })
})
