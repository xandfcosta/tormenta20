/**
 * Vitest setup file — runs before every test module loads. Installs a
 * deterministic MemoryStorage as `globalThis.localStorage` so any code
 * (zustand/persist, lazy module init) that captures the storage handle
 * at import time sees a working object instead of the broken partial
 * jsdom/happy-dom shim in this Vitest version.
 */
import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

/* Vitest doesn't auto-cleanup React trees between tests — DOM leaks
 * across tests and `getByText` starts matching leftovers from an
 * earlier render. Explicit cleanup restores per-test isolation. */
afterEach(() => {
  cleanup()
})
class MemoryStorage {
  private store = new Map<string, string>()
  get length() {
    return this.store.size
  }
  clear() {
    this.store.clear()
  }
  getItem(key: string) {
    return this.store.get(key) ?? null
  }
  setItem(key: string, value: string) {
    this.store.set(key, value)
  }
  removeItem(key: string) {
    this.store.delete(key)
  }
  key(i: number) {
    return Array.from(this.store.keys())[i] ?? null
  }
}

const storage = new MemoryStorage()
Object.defineProperty(globalThis, 'localStorage', {
  value: storage,
  writable: true,
  configurable: true,
})
Object.defineProperty(globalThis, 'sessionStorage', {
  value: new MemoryStorage(),
  writable: true,
  configurable: true,
})

export { storage as testLocalStorage }
