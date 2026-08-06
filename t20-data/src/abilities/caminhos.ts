/**
 * Caminho/subpath option tables — pure static data, NO Deus catalog. Split out
 * of `./deuses` so the frontend can import `caminhoSlotFor`/`CAMINHOS` without
 * anchoring the DEUSES catalog (project_front_decouple_catalog B.3). `./deuses`
 * re-exports these.
 */

/** Caminhos per class — subpath choices unlocked by class abilities. */
export type CaminhoOption = { id: string; name: string }

export const CAMINHOS: Record<string, CaminhoOption[]> = {
  Arcanista: [
    { id: 'bruxo', name: 'Bruxo' },
    { id: 'feiticeiro', name: 'Feiticeiro' },
    { id: 'mago', name: 'Mago' },
  ],
  Paladino: [
    { id: 'egide-sagrada', name: 'Égide Sagrada' },
    { id: 'montaria-sagrada', name: 'Montaria Sagrada' },
  ],
  Cavaleiro: [
    { id: 'bastiao', name: 'Bastião' },
    { id: 'montaria', name: 'Montaria' },
  ],
}

/**
 * Per-class caminho/subpath slot. Returns options + the class level at which
 * the caminho choice unlocks (the class's "Caminho" auto-power), or null when
 * the class has no caminho slot.
 */
export function caminhoSlotFor(
  className: string,
): { options: CaminhoOption[]; minLevel: number } | null {
  const options = CAMINHOS[className]
  if (!options) return null
  const minLevel = className === 'Arcanista' ? 1 : 5
  return { options, minLevel }
}
