import { describe, expect, it } from 'vitest'
import type { Character } from '@/shared/api/api'
import fixtures from './__fixtures__/character-input-parity.json'
import { computedSheetFor } from './computed-sheet'

const chars = fixtures as { slug: string; char: Character }[]
const base = chars[0]!.char

const withConditions = (ids: string[]): Character => ({
  ...base,
  activeConditions: JSON.stringify(ids),
})

const expertise = (char: Character, name: string) =>
  computedSheetFor(char).expertises.find((e) => e.name === name)?.total

/**
 * ALE-28: applied p394 conditions must move the derived sheet numbers, flowing
 * through the same v2 engine as items. These run the TS oracle branch (MODE ===
 * 'test'), which the Go WASM engine reproduces byte-equal via the parity oracle.
 */
describe('condition mechanical effects (ALE-28)', () => {
  it('Vulnerável baixa a Defesa em 2', () => {
    const before = computedSheetFor(base).defense.total
    const after = computedSheetFor(withConditions(['vulneravel'])).defense.total
    expect(after).toBe(before - 2)
  })

  it('Abalado baixa todas as perícias — inclusive Fortitude — em 2', () => {
    const before = expertise(base, 'Fortitude')!
    const after = expertise(withConditions(['abalado']), 'Fortitude')!
    expect(after).toBe(before - 2)
  })

  it('Desprevenido: Defesa −5 e Reflexos −5', () => {
    const defBefore = computedSheetFor(base).defense.total
    const refBefore = expertise(base, 'Reflexos')!
    const sheet = computedSheetFor(withConditions(['desprevenido']))
    expect(sheet.defense.total).toBe(defBefore - 5)
    expect(sheet.expertises.find((e) => e.name === 'Reflexos')!.total).toBe(
      refBefore - 5,
    )
  })

  it('mais severo: Vulnerável + Desprevenido na Defesa = −5, não −7 (p394)', () => {
    const before = computedSheetFor(base).defense.total
    const after = computedSheetFor(
      withConditions(['vulneravel', 'desprevenido']),
    ).defense.total
    expect(after).toBe(before - 5)
  })

  it('Caído penaliza Luta mas não Pontaria', () => {
    const lutaBefore = expertise(base, 'Luta')!
    const pontariaBefore = expertise(base, 'Pontaria')!
    const after = withConditions(['caido'])
    expect(expertise(after, 'Luta')).toBe(lutaBefore - 5)
    expect(expertise(after, 'Pontaria')).toBe(pontariaBefore)
  })

  it('condição só-lembrete (Lento) não altera número nenhum', () => {
    const before = computedSheetFor(base)
    const after = computedSheetFor(withConditions(['lento']))
    expect(after).toEqual(before)
  })
})
