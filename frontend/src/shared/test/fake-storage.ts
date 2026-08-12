/**
 * Named fake for `localStorage`, so a store test never patches the global.
 * Pass it into the `createXStore(storage)` seam.
 *
 * @example const store = createUiStore(new FakeStorage())
 */
export class FakeStorage implements Storage {
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
