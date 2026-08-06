import type { CharacterInput, ComputedSheet } from '@tormenta20/t20-data'

/**
 * Lazy loader + typed wrapper for the engine-go rules engine compiled to WASM.
 * Runs the SAME Go rules as the backend `computeCharacterSheet` — single source
 * of truth, no TS derive duplication (project_front_decouple_catalog Fase 3:
 * compute is free, WASM ~0.28ms/sheet + a one-time 0.95MB load, and works
 * offline). Artifacts are built by `scripts/build-engine-wasm.sh` into
 * `public/engine/` and served by Vite at `/engine/*`.
 *
 * The Go runtime glue (`wasm_exec.js`) defines `globalThis.Go`; we inject it as
 * a <script> (CSP-friendly, no eval) then instantiate the module once. Call
 * `ensureEngine()` from the route loader (parallel to `ensureCatalogs`) so the
 * engine is warm before the first derive; `computeSheet` is sync afterwards.
 */
type GoRuntime = {
  importObject: WebAssembly.Imports
  run(instance: WebAssembly.Instance): void
}
type GlobalWithGo = typeof globalThis & {
  Go?: new () => GoRuntime
  computeCharacterSheet?: (inputJson: string) => string
}

const WASM_URL = '/engine/t20.wasm'
const GLUE_URL = '/engine/wasm_exec.js'
let enginePromise: Promise<void> | null = null

function loadGlue(): Promise<void> {
  const g = globalThis as GlobalWithGo
  if (g.Go) return Promise.resolve()
  return new Promise<void>((resolve, reject) => {
    const s = document.createElement('script')
    s.src = GLUE_URL
    s.onload = () => resolve()
    s.onerror = () => reject(new Error(`engine-wasm: failed to load ${GLUE_URL}`))
    document.head.appendChild(s)
  })
}

async function initEngine(): Promise<void> {
  await loadGlue()
  const g = globalThis as GlobalWithGo
  if (!g.Go) throw new Error('engine-wasm: wasm_exec.js did not define globalThis.Go')
  const go = new g.Go()
  const { instance } = await WebAssembly.instantiateStreaming(
    fetch(WASM_URL),
    go.importObject,
  )
  go.run(instance) // main() blocks on select{}; registers computeCharacterSheet
}

/** Load + instantiate the WASM engine once (cached). Warm it from the route
 *  loader before rendering, like `ensureCatalogs`. */
export function ensureEngine(): Promise<void> {
  enginePromise ??= initEngine()
  return enginePromise
}

/** True once the engine global is registered — for a render-time gate. */
export function isEngineReady(): boolean {
  return typeof (globalThis as GlobalWithGo).computeCharacterSheet === 'function'
}

/**
 * Compute a character's sheet through the Go engine. Requires `ensureEngine()`
 * to have resolved. Mirrors backend `computeCharacterSheet` byte-for-byte
 * (verified by the bench oracle).
 */
export function computeSheet(input: CharacterInput): ComputedSheet {
  const fn = (globalThis as GlobalWithGo).computeCharacterSheet
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(fn(JSON.stringify(input))) as ComputedSheet & {
    error?: string
  }
  if (out.error) throw new Error(`engine-wasm: ${out.error}`)
  return out
}
