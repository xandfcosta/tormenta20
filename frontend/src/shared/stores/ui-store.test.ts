import { FakeStorage } from '@/shared/test/fake-storage'
import { describe, expect, it } from 'vitest'
import { createUiStore, readStoredVolume } from './ui-store'


describe('createUiStore', () => {
  it('som começa desligado e persiste', () => {
    const storage = new FakeStorage()
    const ui = createUiStore(storage)
    expect(ui.sfx()).toBe(false)

    ui.toggleSfx()

    expect(ui.sfx()).toBe(true)
    expect(JSON.parse(storage.getItem('t20-ui') ?? '{}')).toEqual({
      state: { sfx: true, volume: 100 },
    })
  })

  it('hidrata o som do storage', () => {
    const storage = new FakeStorage()
    storage.setItem('t20-ui', '{"state":{"sfx":true}}')
    expect(createUiStore(storage).sfx()).toBe(true)
  })
})

/**
 * Storage corrompido não pode emudecer a mesa nem estourar o ganho: qualquer
 * coisa que não seja um número de 0 a 100 vira volume cheio (ALE-180).
 */
describe('readStoredVolume', () => {
  it('cai em 100 sem nada salvo', () => {
    expect(readStoredVolume(null)).toBe(100)
  })

  it('lê o valor guardado', () => {
    expect(readStoredVolume('{"state":{"volume":40}}')).toBe(40)
  })

  it('prende o que passa dos limites', () => {
    expect(readStoredVolume('{"state":{"volume":250}}')).toBe(100)
    expect(readStoredVolume('{"state":{"volume":-30}}')).toBe(0)
  })

  it('ignora o que não é número', () => {
    expect(readStoredVolume('{"state":{"volume":"alto"}}')).toBe(100)
  })
})

describe('createUiStore — volume', () => {
  it('guarda o volume junto do som', () => {
    const storage = new FakeStorage()
    const ui = createUiStore(storage)
    ui.setVolume(35)
    expect(ui.volume()).toBe(35)
    expect(createUiStore(storage).volume()).toBe(35)
  })
})
