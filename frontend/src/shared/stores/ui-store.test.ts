import { beforeEach, describe, expect, it } from 'vitest'
import { THEME_STORAGE_KEY, useUiStore } from './ui-store'

beforeEach(() => {
  localStorage.clear()
  useUiStore.setState({ theme: 'light' })
})

describe('useUiStore', () => {
  it('initial theme is light', () => {
    expect(useUiStore.getState().theme).toBe('light')
  })

  it('toggleTheme cycles light ↔ dark', () => {
    useUiStore.getState().toggleTheme()
    expect(useUiStore.getState().theme).toBe('dark')
    useUiStore.getState().toggleTheme()
    expect(useUiStore.getState().theme).toBe('light')
  })

  it('setTheme sets the theme explicitly', () => {
    useUiStore.getState().setTheme('dark')
    expect(useUiStore.getState().theme).toBe('dark')
  })

  it('persists to the configured storage key', () => {
    useUiStore.getState().setTheme('dark')
    const raw = localStorage.getItem(THEME_STORAGE_KEY)
    expect(raw).toBeTruthy()
    const parsed = JSON.parse(raw!)
    expect(parsed.state.theme).toBe('dark')
  })
})
