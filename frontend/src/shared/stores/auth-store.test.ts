import { describe, expect, it } from 'vitest'
import { createAuthStore } from './auth-store'

const ALICE = { id: 1, email: 'mestre@t20.local', name: 'Mestre', isAdmin: false }

describe('createAuthStore', () => {
  it('começa deslogado', () => {
    const auth = createAuthStore()
    expect(auth.user()).toBeNull()
    expect(auth.isAuthenticated()).toBe(false)
  })

  it('aceita um usuário inicial (hidratação da rota)', () => {
    const auth = createAuthStore(ALICE)
    expect(auth.user()).toEqual(ALICE)
    expect(auth.isAuthenticated()).toBe(true)
  })

  it('setUser publica o usuário e marca como autenticado', () => {
    const auth = createAuthStore()
    auth.setUser(ALICE)
    expect(auth.user()).toEqual(ALICE)
    expect(auth.isAuthenticated()).toBe(true)
  })

  it('setUser(null) desloga', () => {
    const auth = createAuthStore(ALICE)
    auth.setUser(null)
    expect(auth.user()).toBeNull()
    expect(auth.isAuthenticated()).toBe(false)
  })

  it('cada store é independente (sem singleton global como no zustand)', () => {
    const one = createAuthStore()
    const other = createAuthStore()
    one.setUser(ALICE)
    expect(other.user()).toBeNull()
  })
})
