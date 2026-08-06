/**
 * Vitest setup file — runs before every test module loads. Installs a
 * deterministic MemoryStorage as `globalThis.localStorage` so any code
 * (zustand/persist, lazy module init) that captures the storage handle
 * at import time sees a working object instead of the broken partial
 * jsdom/happy-dom shim in this Vitest version.
 */
import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import {
  CATALOG_ITEMS,
  CLASS_POWERS_CATALOG,
  DEUSES,
  GENERAL_POWERS_CATALOG,
  GRANTED_POWERS,
  ORIGINS_CATALOG,
  RACES_CATALOG,
  SPELL_CATALOG,
} from '@tormenta20/t20-data'
import { afterEach } from 'vitest'
import { primeAbilities } from './shared/lib/abilities-cache'
import { primeItemCatalog } from './shared/lib/catalog-cache'
import { primeSpellCatalog } from './shared/lib/spell-cache'

/* The catalogs are fetched + primed by the root loader at runtime (B.2/B.3 —
 * keeps the data out of the bundle). Tests have no loader, so prime the caches
 * once from the real t20-data catalogs — restores the pre-migration behavior
 * where getCatalogItem / getRace / ownedClassPowers worked synchronously. */
primeItemCatalog(CATALOG_ITEMS)
primeAbilities({
  races: RACES_CATALOG,
  origins: ORIGINS_CATALOG,
  classPowers: CLASS_POWERS_CATALOG,
  generalPowers: GENERAL_POWERS_CATALOG,
  deuses: DEUSES,
  grantedPowers: GRANTED_POWERS,
})
primeSpellCatalog(SPELL_CATALOG)

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
