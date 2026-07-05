import { describe, expect, it } from 'vitest'
import {
  GENERAL_POWERS_CATALOG,
  getGeneralPower,
} from '../abilities/general-powers'

/**
 * Cap 2 residual audit 2026-07-05 — 6 poderes de destino p129-131 que
 * faltavam. Regressão pinada por id.
 */

const AUDIT_MISSING_DESTINO_IDS = [
  'ao-sabor-do-destino',
  'costas-largas',
  'inventario-organizado',
  'investigador',
  'parceiro-poder',
  'veneficio',
] as const

describe('Destino residual audit — 6 novos poderes p129-131', () => {
  it('todos os 6 ids estão no catálogo', () => {
    for (const id of AUDIT_MISSING_DESTINO_IDS) {
      expect(getGeneralPower(id), `missing power ${id}`).toBeDefined()
    }
  })

  it('todos são kind=destino', () => {
    for (const id of AUDIT_MISSING_DESTINO_IDS) {
      expect(getGeneralPower(id)?.kind).toBe('destino')
    }
  })

  it('total destino cresceu para 20 (14 anteriores + 6 novos)', () => {
    const destino = GENERAL_POWERS_CATALOG.filter((p) => p.kind === 'destino')
    expect(destino.length).toBe(20)
  })
})

describe('Destino poderes pinning (p129-131)', () => {
  it('Ao Sabor do Destino exige minLevel 6', () => {
    expect(getGeneralPower('ao-sabor-do-destino')?.minLevel).toBe(6)
  })

  it('Costas Largas exige Con 1 + For 1', () => {
    const prereqs = getGeneralPower('costas-largas')?.prerequisites ?? []
    const con = prereqs.find(
      (p) => p.kind === 'attribute' && p.attr === 'constitution',
    )
    const forr = prereqs.find(
      (p) => p.kind === 'attribute' && p.attr === 'strength',
    )
    expect(con?.kind === 'attribute' && con.min).toBe(1)
    expect(forr?.kind === 'attribute' && forr.min).toBe(1)
  })

  it('Inventário Organizado + Investigador exigem Int 1', () => {
    for (const id of ['inventario-organizado', 'investigador']) {
      const prereqs = getGeneralPower(id)?.prerequisites ?? []
      const attr = prereqs.find(
        (p) => p.kind === 'attribute' && p.attr === 'intelligence',
      )
      expect(attr?.kind === 'attribute' && attr.min, `${id}`).toBe(1)
    }
  })

  it('Parceiro (id parceiro-poder) exige L5 + note Adestramento/Diplomacia', () => {
    const p = getGeneralPower('parceiro-poder')!
    expect(p.minLevel).toBe(5)
    const note = p.prerequisites?.find((x) => x.kind === 'note')
    expect(note?.kind === 'note' && note.description).toMatch(
      /Adestramento|Diplomacia/,
    )
  })

  it('Venefício note Ofício (alquimista)', () => {
    const prereqs = getGeneralPower('veneficio')?.prerequisites ?? []
    const note = prereqs.find((p) => p.kind === 'note')
    expect(note?.kind === 'note' && note.description).toMatch(/alquimista/)
  })
})
