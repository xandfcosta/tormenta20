import { beforeEach, describe, expect, it } from 'vitest'
import { createUiStore, readStoredTheme } from './ui-store'

/** Named fake for localStorage — no global patching. */
class FakeStorage implements Storage {
  private readonly entries = new Map<string, string>()
  get length() {
    return this.entries.size
  }
  clear() {
    this.entries.clear()
  }
  getItem(key: string) {
    return this.entries.get(key) ?? null
  }
  key(index: number) {
    return [...this.entries.keys()][index] ?? null
  }
  removeItem(key: string) {
    this.entries.delete(key)
  }
  setItem(key: string, value: string) {
    this.entries.set(key, value)
  }
}

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
    expect(JSON.parse(storage.getItem('t20-ui') ?? '{}')).toEqual({ state: { theme: 'dark' } })
  })
})
