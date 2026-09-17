
/**
 * Persisted under the SAME key/shape the React app's zustand store used
 * (`t20-ui` → `{ state: { … } }`). O campo `theme` que morava aqui SAIU na
 * ALE-173: o app tem uma identidade só — grimório escuro — desde que a porta
 * do jogo virou cena, e o seletor claro/escuro nunca teve controle que o
 * chamasse. Um valor antigo no storage é simplesmente ignorado.
 */
export const STORAGE_KEY = 't20-ui'

/** Grava a preferência na MESMA chave e forma que a SPA usa. Exportado porque o
 *  módulo do Datastar (`piloto/scene.ts`, ALE-231) escreve nela: som e volume
 *  são preferência DESTE aparelho, e as duas portas do app têm de ler a mesma. */
export function persistUi(state: { sfx: boolean; volume: number }, storage = globalThis.localStorage): void {
  storage?.setItem(STORAGE_KEY, JSON.stringify({ state }))
}


/** Cheio por padrão: os cues foram afinados um a um nesse ganho, e o slider só
 *  atenua a partir daí (ALE-180). */
const FULL_VOLUME = 100

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

