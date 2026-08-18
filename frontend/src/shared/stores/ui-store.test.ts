import { FakeStorage } from '@/shared/test/fake-storage'
import { beforeEach, describe, expect, it } from 'vitest'
import { createUiStore, readStoredTheme, readStoredVolume } from './ui-store'


describe('readStoredTheme', () => {
  it('cai em light sem nada salvo', () => {
    expect(readStoredTheme(null)).toBe('light')
  })

  it('lê o formato do zustand do app React', () => {
    expect(readStoredTheme('{"state":{"theme":"dark"}}')).toBe('dark')
  })

  it('não explode com JSON corrompido', () => {
    expect(readStoredTheme('{isto não é json')).toBe('light')
  })

  it('ignora valor desconhecido', () => {
    expect(readStoredTheme('{"state":{"theme":"neon"}}')).toBe('light')
  })
})

describe('createUiStore', () => {
  beforeEach(() => document.documentElement.classList.remove('dark'))

  it('hidrata do storage', () => {
    const storage = new FakeStorage()
    storage.setItem('t20-ui', '{"state":{"theme":"dark"}}')
    expect(createUiStore(storage).theme()).toBe('dark')
  })

  it('toggle alterna e aplica a classe dark no <html>', () => {
    const ui = createUiStore(new FakeStorage())
    ui.toggleTheme()
    expect(ui.theme()).toBe('dark')
    expect(document.documentElement).toHaveClass('dark')
    ui.toggleTheme()
    expect(document.documentElement).not.toHaveClass('dark')
  })

  it('persiste no formato que o index.html lê antes do mount', () => {
    const storage = new FakeStorage()
    createUiStore(storage).setTheme('dark')
    expect(JSON.parse(storage.getItem('t20-ui') ?? '{}')).toEqual({
      state: { theme: 'dark', sfx: false, volume: 100 },
    })
  })

  it('som começa desligado e persiste junto com o tema', () => {
    const storage = new FakeStorage()
    const ui = createUiStore(storage)
    expect(ui.sfx()).toBe(false)

    ui.toggleSfx()

    expect(ui.sfx()).toBe(true)
    expect(JSON.parse(storage.getItem('t20-ui') ?? '{}')).toEqual({
      state: { theme: 'light', sfx: true, volume: 100 },
    })
  })

  it('hidrata o som do storage', () => {
    const storage = new FakeStorage()
    storage.setItem('t20-ui', '{"state":{"theme":"light","sfx":true}}')
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
  it('guarda o volume junto do tema e do som', () => {
    const storage = new FakeStorage()
    const ui = createUiStore(storage)
    ui.setVolume(35)
    expect(ui.volume()).toBe(35)
    expect(createUiStore(storage).volume()).toBe(35)
  })
})
