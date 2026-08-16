import { FakeStorage } from '@/shared/test/fake-storage'
import { beforeEach, describe, expect, it } from 'vitest'
import { createUiStore, readStoredTheme } from './ui-store'


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
      state: { theme: 'dark', sfx: false },
    })
  })

  it('som começa desligado e persiste junto com o tema', () => {
    const storage = new FakeStorage()
    const ui = createUiStore(storage)
    expect(ui.sfx()).toBe(false)

    ui.toggleSfx()

    expect(ui.sfx()).toBe(true)
    expect(JSON.parse(storage.getItem('t20-ui') ?? '{}')).toEqual({
      state: { theme: 'light', sfx: true },
    })
  })

  it('hidrata o som do storage', () => {
    const storage = new FakeStorage()
    storage.setItem('t20-ui', '{"state":{"theme":"light","sfx":true}}')
    expect(createUiStore(storage).sfx()).toBe(true)
  })
})
