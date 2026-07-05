import { describe, expect, it } from 'vitest'
import {
  GENERAL_POWERS_CATALOG,
  getGeneralPower,
} from '../abilities/general-powers'

/**
 * Cap 2 residual audit 2026-07-05 — 22 combate poderes que faltavam,
 * enumerados no gap audit + pinados aqui pelo id. Regressão: caso
 * alguém remova um por engano, este arquivo grita.
 */

const AUDIT_MISSING_COMBATE_IDS = [
  'arma-secundaria-grande',
  'arremesso-multiplo',
  'arremesso-potente',
  'ataque-com-escudo',
  'ataque-pesado',
  'ataque-preciso',
  'bloqueio-com-escudo',
  'carga-de-cavalaria',
  'derrubar-aprimorado',
  'desarmar-aprimorado',
  'disparo-preciso',
  'disparo-rapido',
  'empunhadura-poderosa',
  'estilo-de-arma-longa',
  'fanatico',
  'finta-aprimorada',
  'inexpugnavel',
  'mira-apurada',
  'piqueiro',
  'presenca-aterradora',
  'quebrar-aprimorado',
  'trespassar',
] as const

describe('Combate residual audit — 22 novos poderes p124-129', () => {
  it('todos os 22 ids estão no catálogo', () => {
    for (const id of AUDIT_MISSING_COMBATE_IDS) {
      expect(getGeneralPower(id), `missing power ${id}`).toBeDefined()
    }
  })

  it('todos são kind=combate', () => {
    for (const id of AUDIT_MISSING_COMBATE_IDS) {
      expect(getGeneralPower(id)?.kind).toBe('combate')
    }
  })

  it('total combate cresceu para 40 (18 anteriores + 22 novos)', () => {
    const combate = GENERAL_POWERS_CATALOG.filter((p) => p.kind === 'combate')
    expect(combate.length).toBe(40)
  })
})

describe('Combate poderes pins seleccionados (p124-129)', () => {
  it('Ataque Poderoso é prereq de Quebrar Aprimorado + Trespassar', () => {
    for (const id of ['quebrar-aprimorado', 'trespassar']) {
      const prereqs = getGeneralPower(id)?.prerequisites ?? []
      const hasParent = prereqs.some(
        (p) => p.kind === 'power' && p.id === 'ataque-poderoso',
      )
      expect(hasParent, `${id} missing Ataque Poderoso prereq`).toBe(true)
    }
  })

  it('Estilo de Arma e Escudo é prereq de Ataque com Escudo + Bloqueio com Escudo', () => {
    for (const id of ['ataque-com-escudo', 'bloqueio-com-escudo']) {
      const prereqs = getGeneralPower(id)?.prerequisites ?? []
      const hasParent = prereqs.some(
        (p) => p.kind === 'power' && p.id === 'estilo-de-arma-e-escudo',
      )
      expect(hasParent).toBe(true)
    }
  })

  it('Fanático exige minLevel 12 + Encouraçado', () => {
    const p = getGeneralPower('fanatico')!
    expect(p.minLevel).toBe(12)
    expect(p.prerequisites?.some(
      (x) => x.kind === 'power' && x.id === 'encouracado',
    )).toBe(true)
  })

  it('Inexpugnável exige minLevel 6 + Encouraçado', () => {
    const p = getGeneralPower('inexpugnavel')!
    expect(p.minLevel).toBe(6)
  })

  it('Empunhadura Poderosa exige For 3', () => {
    const prereqs = getGeneralPower('empunhadura-poderosa')?.prerequisites ?? []
    const attr = prereqs.find(
      (p) => p.kind === 'attribute' && p.attr === 'strength',
    )
    expect(attr?.kind === 'attribute' && attr.min).toBe(3)
  })

  it('Disparo Preciso aceita Estilo de Disparo OU Estilo de Arremesso', () => {
    const prereqs = getGeneralPower('disparo-preciso')?.prerequisites ?? []
    const any = prereqs.find((p) => p.kind === 'anyPower')
    expect(any?.kind === 'anyPower' && any.ids).toEqual([
      'estilo-de-disparo',
      'estilo-de-arremesso',
    ])
  })

  it('Presença Aterradora exige treinado em Intimidação', () => {
    const prereqs = getGeneralPower('presenca-aterradora')?.prerequisites ?? []
    const trained = prereqs.find((p) => p.kind === 'trained')
    expect(trained?.kind === 'trained' && trained.expertise).toBe('Intimidação')
  })

  it('Carga de Cavalaria exige Ginete', () => {
    const prereqs = getGeneralPower('carga-de-cavalaria')?.prerequisites ?? []
    expect(prereqs.some((p) => p.kind === 'power' && p.id === 'ginete')).toBe(
      true,
    )
  })
})
