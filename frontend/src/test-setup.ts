/**
 * Vitest setup file — runs before every test module loads. Installs a
 * deterministic MemoryStorage as `globalThis.localStorage` so any code
 * (zustand/persist, lazy module init) that captures the storage handle
 * at import time sees a working object instead of the broken partial
 * jsdom/happy-dom shim in this Vitest version.
 */
import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { CATALOG_ITEMS } from '@tormenta20/t20-data'
import { afterEach } from 'vitest'
import { primeItemCatalog } from './shared/lib/catalog-cache'

/* The item catalog is fetched + primed by the root loader at runtime (B.2 —
 * keeps the data out of the bundle). Tests have no loader, so prime the cache
 * once from the real t20-data catalog — restores the pre-migration behavior
 * where getCatalogItem worked synchronously at import. */
primeItemCatalog(CATALOG_ITEMS)

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
