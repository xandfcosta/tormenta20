import { type Accessor, createSignal } from 'solid-js'

/**
 * Persisted under the SAME key/shape the React app's zustand store used
 * (`t20-ui` → `{ state: { … } }`). O campo `theme` que morava aqui SAIU na
 * ALE-173: o app tem uma identidade só — grimório escuro — desde que a porta
 * do jogo virou cena, e o seletor claro/escuro nunca teve controle que o
 * chamasse. Um valor antigo no storage é simplesmente ignorado.
 */
const STORAGE_KEY = 't20-ui'

/** Cheio por padrão: os cues foram afinados um a um nesse ganho, e o slider só
 *  atenua a partir daí (ALE-180). */
const FULL_VOLUME = 100

export type UiStore = {
  /** UI sound cues (hover/select/scene transition). Off by default so the app
   *  never surprises with sound — the player opts in from the Hub. */
  sfx: Accessor<boolean>
  setSfx: (on: boolean) => void
  toggleSfx: () => void
  /** 0–100. Alerta que só liga e desliga é alerta que se desliga: quem acha o
   *  sino do "Sua vez" alto abaixa em vez de calar a mesa inteira (ALE-180). */
  volume: Accessor<number>
  setVolume: (percent: number) => void
}

type PersistedUi = { state?: { sfx?: unknown; volume?: unknown } }

function parseStored(raw: string | null): PersistedUi['state'] {
  if (!raw) return undefined
  try {
    return (JSON.parse(raw) as PersistedUi).state
  } catch {
    return undefined
  }
}

export function readStoredSfx(raw: string | null): boolean {
  return parseStored(raw)?.sfx === true
}

/** Qualquer coisa que não seja um número de 0 a 100 vira volume cheio: um
 *  storage corrompido não pode emudecer a mesa nem estourar o ganho. */
export function readStoredVolume(raw: string | null): number {
  return clampVolume(parseStored(raw)?.volume)
}

function clampVolume(value: unknown): number {
  if (typeof value !== 'number' || Number.isNaN(value)) return FULL_VOLUME
  return Math.min(100, Math.max(0, Math.round(value)))
}

/**
 * Preferências de interface: som e volume.
 *
 * @example const ui = createUiStore(); ui.toggleSfx()
 */
export function createUiStore(storage: Storage | undefined = globalThis.localStorage): UiStore {
  const stored = storage?.getItem(STORAGE_KEY) ?? null
  const [sfx, setSfxSignal] = createSignal(readStoredSfx(stored))
  const [volume, setVolumeSignal] = createSignal(readStoredVolume(stored))

  const persist = () => {
    const state = { sfx: sfx(), volume: volume() }
    storage?.setItem(STORAGE_KEY, JSON.stringify({ state }))
  }

  const setSfx = (on: boolean) => {
    setSfxSignal(on)
    persist()
  }

  const setVolume = (percent: number) => {
    setVolumeSignal(clampVolume(percent))
    persist()
  }

  return {
    sfx,
    setSfx,
    toggleSfx: () => setSfx(!sfx()),
    volume,
    setVolume,
  }
}
