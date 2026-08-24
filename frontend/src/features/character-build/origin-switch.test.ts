import { describe, expect, it } from 'vitest'
import { originSwitchPatch } from './origin-switch'
import { wizardDefaults } from './wizard-steps'

describe('originSwitchPatch', () => {
  it('sets the new origin and clears the previous picks', () => {
    const patch = originSwitchPatch(
      {
        ...wizardDefaults,
        origin: 'Acólito',
        originChoices: ['acolito-cura-leve'],
        originItemPicks: { 'Símbolo sagrado': 'simbolo-sagrado' },
      },
      'Batedor',
    )
    expect(patch.origin).toBe('Batedor')
    expect(patch.originChoices).toEqual([])
    expect(patch.originItemPicks).toEqual({})
  })

  /**
   * O dinheiro que a origem ANTERIOR já rolou sai com ela — dinheiro não pode
   * vazar de uma origem que o personagem não tem mais.
   *
   * Reescrito na ALE-187, e a reescrita achou um defeito: o esperado era
   * `12 - origemRolledMoneySum('Batedor', picks)`, calculado pela função IRMÃ.
   * Isso já bastaria para condenar — implementação comparada consigo mesma —,
   * mas o pior estava na FIXTURE: o Batedor não concede dinheiro nenhum (os
   * itens dele são barraca, equipamento de viagem e uma arma), e o rótulo
   * `'T$ 2d6'` não existe em origem alguma. A soma devolvia ZERO, o teste
   * afirmava `12 - 0 === 12` e passava sem nunca exercitar a subtração que ele
   * diz proteger.
   *
   * Quem rola dinheiro é o MARUJO, com o rótulo exato do catálogo
   * (`origens.json`). Com ele o caso finalmente morde: 12 na bolsa, 7 rolados,
   * sobram 5 — número TRANSCRITO, não recalculado.
   */
  it('gives back the T$ the previous origin had already rolled', () => {
    const patch = originSwitchPatch(
      {
        ...wizardDefaults,
        origin: 'Marujo',
        originItemPicks: { 'T$ 2d6 (último salário)': '7' },
        tibar: 12,
      },
      'Artesão',
    )

    expect(patch.tibar).toBe(5)
  })

  it('never drives tibar negative', () => {
    const patch = originSwitchPatch(
      { ...wizardDefaults, origin: 'Batedor', originItemPicks: { 'T$ 2d6': '9' }, tibar: 0 },
      'Artesão',
    )
    expect(patch.tibar).toBe(0)
  })

  it('drops the power picked for a benefit of the origin being left', () => {
    const patch = originSwitchPatch(
      {
        ...wizardDefaults,
        origin: 'Soldado',
        originChoices: ['origin-soldado-poder-poder-de-combate-escolha'],
        powerChoices: { 'origin-soldado-poder-poder-de-combate-escolha': ['ataque-poderoso'], 'racial-x': ['dentes-afiados'] },
      },
      'Artesão',
    )
    expect(patch.powerChoices['origin-soldado-poder-poder-de-combate-escolha']).toBeUndefined()
    // Picks that belong to anything else (race, class) are not this step's to erase.
    expect(patch.powerChoices['racial-x']).toEqual(['dentes-afiados'])
  })

  it('is a no-op on tibar when there was no origin yet', () => {
    const patch = originSwitchPatch({ ...wizardDefaults, tibar: 5 }, 'Acólito')
    expect(patch.tibar).toBe(5)
    expect(patch.origin).toBe('Acólito')
  })
})
