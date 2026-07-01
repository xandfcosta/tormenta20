/**
 * Spell save + resistance + concentration extras.
 *
 * Complementa `spells.ts` (que já tem `spellSaveDc`, `concentrationCd`
 * normal/ruim/terrivel, `SpellResistance` enum).
 *
 * PDF refs:
 *  - Concentração (dano/armor): Cap 4 p170-171
 *  - Resistência aplicada: Cap 5 p227 (verbatim de cada tipo)
 *  - Contramágica: Cap 4 p173
 *  - Alvo múltiplo: Cap 4 p173
 */
import type { SpellResistance } from './spells'

// ─── Concentration CD variants (p170-171) ───────────────────────────
/**
 * PDF p170: se conjurador sofre dano durante lançamento, CD do
 * Vontade = dano recebido. Retorna 0 se dano ≤ 0 (sem teste).
 */
export function damageConcentrationCd(damageTaken: number): number {
  return damageTaken > 0 ? damageTaken : 0
}

/**
 * PDF p171: lançar magia arcana com armadura exige teste Misticismo
 * CD 20 + custo total em PM da magia + penalidade da armadura.
 */
export function armorArcaneConcentrationCd(
  spellTotalPm: number,
  armorPenalty: number,
): number {
  if (spellTotalPm < 0) {
    throw new Error(
      `armorArcaneConcentrationCd: spellTotalPm must be ≥ 0, got ${spellTotalPm}`,
    )
  }
  if (armorPenalty < 0) {
    throw new Error(
      `armorArcaneConcentrationCd: armorPenalty must be ≥ 0, got ${armorPenalty}`,
    )
  }
  return 20 + spellTotalPm + armorPenalty
}

// ─── Save resolution (p227) ─────────────────────────────────────────
/**
 * Resultado semântico da aplicação de resistência.
 *  - `full`  — efeito completo aplicado ao alvo
 *  - `half`  — efeito numérico à metade (parcial em "metade")
 *  - `none`  — efeito não afeta o alvo
 *  - `reduced` — efeito reduzido (per-spell text define quanto — "parcial")
 *  - `revealed` — alvo percebe a ilusão mas ela continua funcionando
 */
export type SaveOutcome = 'full' | 'half' | 'none' | 'reduced' | 'revealed'

/**
 * Aplica resultado do teste de resistência ao alvo.
 *  - anula + pass → none;  anula + fail → full
 *  - metade + pass → half; metade + fail → full
 *  - parcial + pass → reduced; parcial + fail → full
 *  - desacredita + pass → revealed; desacredita + fail → full
 */
export function applySaveResult(
  resistance: SpellResistance,
  savePassed: boolean,
): SaveOutcome {
  if (!savePassed) return 'full'
  switch (resistance) {
    case 'anula':
      return 'none'
    case 'metade':
      return 'half'
    case 'parcial':
      return 'reduced'
    case 'desacredita':
      return 'revealed'
  }
}

/**
 * Aplica resistência a um valor numérico de dano.
 *  - `anula` + pass → 0
 *  - `metade` + pass → floor(damage / 2)
 *  - `parcial` + pass → damage (per-spell text define; sem redução default)
 *  - `desacredita` + pass → 0 (ilusão não causa dano real ao alvo)
 *  - fail (qualquer tipo) → damage integral
 */
export function damageAfterSave(
  damage: number,
  resistance: SpellResistance,
  savePassed: boolean,
): number {
  if (damage < 0) {
    throw new Error(`damageAfterSave: damage must be ≥ 0, got ${damage}`)
  }
  if (!savePassed) return damage
  switch (resistance) {
    case 'anula':
      return 0
    case 'metade':
      return Math.floor(damage / 2)
    case 'parcial':
      return damage
    case 'desacredita':
      return 0
  }
}

/**
 * Determina se o alvo tem direito a teste extra de resistência pra
 * desacreditar ilusão. Regra p227: interação (examinar de perto,
 * tocar) permite reroll. Apenas observar de longe não permite.
 */
export function desacreditaExtraSaveAllowed(
  meaningfulInteraction: boolean,
): boolean {
  return meaningfulInteraction === true
}

// ─── Counterspell (p173) ────────────────────────────────────────────
export type CounterspellRule =
  | { kind: 'same-spell'; spellId: string }
  | {
      kind: 'dispel'
      /** Misticismo do defensor (quem lança contramágica). */
      dispelerMisticismo: number
      /** max(Misticismo, Vontade) do conjurador original. */
      casterOpposedMod: number
    }

export type CounterspellResult = {
  succeeds: boolean
  /** Rolls consumidos (dispel só). */
  detail: string
}

/**
 * Resolve tentativa de contramágica.
 *
 * Rule p173:
 *  - `same-spell`: lançar mesma magia via ação preparar → cancela.
 *  - `dispel`: Dissipar Magia via teste Misticismo oposto por
 *    max(Misticismo, Vontade) do conjurador original.
 *
 * Testes usam d20 externos — este helper apenas resolve regra dado
 * `dispelerRoll` + `casterRoll` (bônus já somados).
 */
export function resolveCounterspell(
  rule: CounterspellRule,
  dispelerRoll?: number,
  casterRoll?: number,
): CounterspellResult {
  if (rule.kind === 'same-spell') {
    return {
      succeeds: true,
      detail: `same-spell counter (${rule.spellId})`,
    }
  }
  if (dispelerRoll === undefined || casterRoll === undefined) {
    throw new Error(
      'resolveCounterspell: dispel kind requires dispelerRoll + casterRoll',
    )
  }
  const dispelTotal = dispelerRoll + rule.dispelerMisticismo
  const casterTotal = casterRoll + rule.casterOpposedMod
  return {
    succeeds: dispelTotal > casterTotal,
    detail: `dispel Misticismo ${dispelTotal} vs opposed ${casterTotal}`,
  }
}
