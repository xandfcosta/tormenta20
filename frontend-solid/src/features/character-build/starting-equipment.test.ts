import { describe, expect, it } from 'vitest'
import {
  bagagemGroups,
  lightArmorOptions,
  purchasesPayload,
  purchasesTotal,
  shopCatalog,
  originStartingItems,
  startingItemsPayload,
  startingLoadout,
  startingSlots,
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

describe('loja — purchasesTotal / purchasesPayload / shopCatalog', () => {
  it('soma preços × quantidade com centavos exatos', () => {
    // adaga T$ 2; tocha? use catalog-known: adaga ×2 = 4
    expect(purchasesTotal({ adaga: 2 })).toBe(4)
  })

  it('centavos não acumulam erro de float', () => {
    const cheap = shopCatalog('all').find((i) => i.price === 0.1)
    if (!cheap) return // catálogo sem item de T$ 0,1 — nada a testar
    expect(purchasesTotal({ [cheap.id]: 3 })).toBe(0.3)
  })

  it('ids desconhecidos e qty 0 são ignorados', () => {
    expect(purchasesTotal({ 'nao-existe': 5, adaga: 0 })).toBe(0)
    expect(purchasesPayload({ 'nao-existe': 5, adaga: 0 })).toEqual([])
  })

  it('payload carrega catalogId + quantidade, sem equipar', () => {
    expect(purchasesPayload({ adaga: 2 })).toEqual([
      { catalogId: 'adaga', name: 'Adaga', quantity: 2, slots: 1 },
    ])
  })

  it('shopCatalog exclui overlays (improvement/material)', () => {
    expect(
      shopCatalog('all').some(
        (i) => i.category === 'improvement' || i.category === 'material',
      ),
    ).toBe(false)
  })

  it('categoria Armas só traz armas', () => {
    expect(
      shopCatalog('weapons').every((i) => i.category.startsWith('weapon-')),
    ).toBe(true)
  })
})

describe('startingSlots — espaços de inventário (p141)', () => {
  const kit = startingLoadout('Guerreiro', 1).kit
  const draft = {
    weaponSimple: 'espada-curta',
    weaponMartial: 'espada-longa',
    armor: 'brunea',
    shield: true,
  }

  it('capacidade = 10 + 2×|FOR|', () => {
    expect(startingSlots(draft, kit, '', {}, {}, 3).capacity).toBe(16)
    expect(startingSlots(draft, kit, '', {}, {}, -1).capacity).toBe(12)
  })

  it('usados somam kit + compras (slots × quantidade)', () => {
    const none = startingSlots(
      { weaponSimple: '', weaponMartial: '', armor: '', shield: false },
      kit,
      '',
      {},
      {},
      0,
    )
    const withBuys = startingSlots(
      { weaponSimple: '', weaponMartial: '', armor: '', shield: false },
      kit,
      '',
      {},
      { adaga: 4 },
      0,
    )
    expect(withBuys.used).toBe(none.used + 4)
  })
})

describe('bagagemGroups — Sua bagagem derivada', () => {
  const kit = startingLoadout('Guerreiro', 1).kit

  it('agrupa por fonte com fantasmas para escolhas pendentes', () => {
    const groups = bagagemGroups(
      { weaponSimple: '', weaponMartial: 'espada-longa', armor: '', shield: true },
      kit,
      'Acólito',
      {},
      { adaga: 2 },
    )
    expect(groups.map((g) => g.title)).toEqual(['Kit', 'Classe', 'Origem', 'Comprado'])
    const classe = groups.find((g) => g.title === 'Classe')
    expect(classe?.lines[0]).toMatchObject({ kind: 'ghost', label: 'arma simples' })
    expect(
      classe?.lines.some((l) => l.kind === 'item' && l.name === 'Espada longa'),
    ).toBe(true)
    expect(
      classe?.lines.some((l) => l.kind === 'ghost' && l.label === 'armadura'),
    ).toBe(true)
    const comprado = groups.find((g) => g.title === 'Comprado')
    expect(comprado?.lines[0]).toMatchObject({ kind: 'item', name: 'Adaga', qty: 2 })
  })

  it('grupos vazios somem (sem origem, sem compras)', () => {
    const groups = bagagemGroups(
      { weaponSimple: 'adaga', weaponMartial: 'espada-longa', armor: 'brunea', shield: false },
      kit,
      '',
      {},
      {},
    )
    expect(groups.map((g) => g.title)).toEqual(['Kit', 'Classe'])
  })

  it('grant de origem sem pick vira fantasma', () => {
    const groups = bagagemGroups(
      { weaponSimple: '', weaponMartial: '', armor: '', shield: false },
      kit,
      'Refugiado',
      {},
      {},
    )
    const origem = groups.find((g) => g.title === 'Origem')
    expect(origem?.lines.some((l) => l.kind === 'ghost')).toBe(true)
  })
})
