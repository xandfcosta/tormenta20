/**
 * Reward / wealth / difficulty tables — PDF Cap 8 (Recompensas p326-332)
 * + Cap 3 (Equipamento, Estadia p157), Cap 5 (Dificuldades p220), p106
 * (Condições de Descanso).
 *
 * Out of scope this module: the d% loot subtables (Tabela 8-1/8-2/8-3/
 * 8-4/8-5). Those drive random treasure generation and belong to a
 * dedicated loot-generator module.
 *
 * In scope:
 *  - Tabela 3-1 (Dinheiro Inicial, p140) — wealth by character level.
 *  - Tabela 8-6 (Tesouro Médio por Cena, p332) — GM treasure pacing.
 *  - Tabela 5-1 (Dificuldades, p220) — CD tiers with jump 30→40.
 *  - Tabela 3-6 (Serviços, p157) — estadia / condução / curandeiro /
 *    magia paga / mensageiro.
 *  - p106 — Condições de descanso (ruim/normal/confortável/luxuosa).
 *  - Tabela 8-7 (Preço de Encantos, p334) — repete o que já está em
 *    `weapon-enchants.ts` mas com o CD-aumento associado.
 *
 * Notes from PDF:
 *  - NO ND→XP table; T20 usa fórmula `ND × 1000` (`xp.ts`).
 *  - NO "starting wealth by level" formula above L1 — valores são
 *    hand-tuned. Encoded como array literal.
 *  - NO monthly cost of living. Custo de vida é por noite (Estadia).
 *  - Tabela 5-1 tem salto 30 → 40 sem CD 35.
 */
import type { SpellCircle } from './spells'

// ─── Tabela 5-1: Dificuldades (p220) ─────────────────────────────────
export type DifficultyTier =
  | 'facil'
  | 'media'
  | 'dificil'
  | 'desafiadora'
  | 'formidavel'
  | 'heroica'
  | 'quase-impossivel'

export type DifficultyRow = {
  tier: DifficultyTier
  label: string
  cd: number
  example: string
}

export const DIFFICULTY_TABLE: readonly DifficultyRow[] = Object.freeze([
  { tier: 'facil', label: 'Fácil', cd: 5, example: 'Subir uma encosta íngreme (Atletismo)' },
  { tier: 'media', label: 'Média', cd: 10, example: 'Ouvir um guarda se aproximando (Percepção)' },
  { tier: 'dificil', label: 'Difícil', cd: 15, example: 'Estancar um sangramento (Cura)' },
  { tier: 'desafiadora', label: 'Desafiadora', cd: 20, example: 'Nadar contra correnteza (Atletismo)' },
  { tier: 'formidavel', label: 'Formidável', cd: 25, example: 'Sabotar armadilha complexa (Ladinagem)' },
  { tier: 'heroica', label: 'Heroica', cd: 30, example: 'Decifrar pergaminho antigo em idioma morto (Conhecimento)' },
  { tier: 'quase-impossivel', label: 'Quase Impossível', cd: 40, example: 'Fabricar obra-prima (item com 4 melhorias) (Ofício)' },
])

export function difficultyForCd(cd: number): DifficultyRow {
  let best = DIFFICULTY_TABLE[0]!
  for (const row of DIFFICULTY_TABLE) {
    if (cd >= row.cd) best = row
  }
  return best
}

// ─── Tabela 3-1: Dinheiro Inicial (p140) ─────────────────────────────
/** Nível 1: 4d6 T$ (média 14). Demais níveis: valor fixo. */
export type StartingWealthRow = {
  level: number
  startingTibar: number
  isDiced: boolean
}

export const STARTING_WEALTH_TABLE: readonly StartingWealthRow[] = Object.freeze([
  { level: 1, startingTibar: 14, isDiced: true },
  { level: 2, startingTibar: 300, isDiced: false },
  { level: 3, startingTibar: 600, isDiced: false },
  { level: 4, startingTibar: 1000, isDiced: false },
  { level: 5, startingTibar: 2000, isDiced: false },
  { level: 6, startingTibar: 3000, isDiced: false },
  { level: 7, startingTibar: 5000, isDiced: false },
  { level: 8, startingTibar: 7000, isDiced: false },
  { level: 9, startingTibar: 10000, isDiced: false },
  { level: 10, startingTibar: 13000, isDiced: false },
  { level: 11, startingTibar: 19000, isDiced: false },
  { level: 12, startingTibar: 27000, isDiced: false },
  { level: 13, startingTibar: 36000, isDiced: false },
  { level: 14, startingTibar: 49000, isDiced: false },
  { level: 15, startingTibar: 66000, isDiced: false },
  { level: 16, startingTibar: 88000, isDiced: false },
  { level: 17, startingTibar: 110000, isDiced: false },
  { level: 18, startingTibar: 150000, isDiced: false },
  { level: 19, startingTibar: 200000, isDiced: false },
  { level: 20, startingTibar: 260000, isDiced: false },
])

export function startingWealthForLevel(level: number): number {
  const clamped = Math.max(1, Math.min(20, level))
  return STARTING_WEALTH_TABLE.find((r) => r.level === clamped)!.startingTibar
}

// ─── Tabela 8-6: Tesouro Médio por Cena (p332) ───────────────────────
export type AverageTreasureRow = {
  level: number
  treasureTibar: number
}

export const AVERAGE_TREASURE_PER_SCENE: readonly AverageTreasureRow[] = Object.freeze([
  { level: 1, treasureTibar: 300 },
  { level: 2, treasureTibar: 300 },
  { level: 3, treasureTibar: 400 },
  { level: 4, treasureTibar: 1000 },
  { level: 5, treasureTibar: 1000 },
  { level: 6, treasureTibar: 2000 },
  { level: 7, treasureTibar: 2000 },
  { level: 8, treasureTibar: 3000 },
  { level: 9, treasureTibar: 3000 },
  { level: 10, treasureTibar: 6000 },
  { level: 11, treasureTibar: 8000 },
  { level: 12, treasureTibar: 9000 },
  { level: 13, treasureTibar: 13000 },
  { level: 14, treasureTibar: 17000 },
  { level: 15, treasureTibar: 22000 },
  { level: 16, treasureTibar: 22000 },
  { level: 17, treasureTibar: 40000 },
  { level: 18, treasureTibar: 50000 },
  { level: 19, treasureTibar: 60000 },
  { level: 20, treasureTibar: 72000 },
])

export function averageTreasurePerScene(level: number): number {
  const clamped = Math.max(1, Math.min(20, level))
  return AVERAGE_TREASURE_PER_SCENE.find((r) => r.level === clamped)!.treasureTibar
}

// ─── Tabela 3-6: Serviços comuns (p157) ──────────────────────────────
export type ServiceCategory =
  | 'estadia'
  | 'conducao'
  | 'curandeiro'
  | 'magia'
  | 'mensageiro'

export type ServiceUnit = 'por-noite' | 'por-km' | 'por-uso' | 'por-dia'

export type ServicePriceRow = {
  category: ServiceCategory
  service: string
  priceTibar: number
  unit: ServiceUnit
}

export const COMMON_SERVICES_TABLE: readonly ServicePriceRow[] = Object.freeze([
  { category: 'estadia', service: 'Estadia comum', priceTibar: 0.5, unit: 'por-noite' },
  { category: 'estadia', service: 'Estadia confortável', priceTibar: 4, unit: 'por-noite' },
  { category: 'estadia', service: 'Estadia luxuosa', priceTibar: 20, unit: 'por-noite' },
  { category: 'conducao', service: 'Condução terrestre', priceTibar: 0.5, unit: 'por-km' },
  { category: 'conducao', service: 'Condução marítima', priceTibar: 0.1, unit: 'por-km' },
  { category: 'conducao', service: 'Condução aérea', priceTibar: 10, unit: 'por-km' },
  { category: 'curandeiro', service: 'Curandeiro', priceTibar: 5, unit: 'por-uso' },
  { category: 'magia', service: 'Magia de 1º círculo', priceTibar: 10, unit: 'por-uso' },
  { category: 'magia', service: 'Magia de 2º círculo', priceTibar: 90, unit: 'por-uso' },
  { category: 'magia', service: 'Magia de 3º círculo', priceTibar: 360, unit: 'por-uso' },
  { category: 'mensageiro', service: 'Mensageiro', priceTibar: 0.5, unit: 'por-km' },
])

/**
 * Custo de magia paga por círculo (1-3 listados no PDF; 4-5 não estão
 * na Tabela 3-6).
 */
export function paidMagicCostTibar(circle: SpellCircle): number | null {
  if (circle === 1) return 10
  if (circle === 2) return 90
  if (circle === 3) return 360
  return null
}

// ─── Condições de Descanso (p106) ────────────────────────────────────
export type RestQuality = 'ruim' | 'normal' | 'confortavel' | 'luxuosa'

export type RestRecoveryRow = {
  quality: RestQuality
  /** Multiplica pelo nível do personagem para PV/PM recuperados por noite. */
  recoveryMultiplier: number
  /** T$ por noite (Tabela 3-6) se for estadia paga; null = dormir ao relento. */
  pairedEstadiaTibar: number | null
}

export const REST_RECOVERY_TABLE: readonly RestRecoveryRow[] = Object.freeze([
  { quality: 'ruim', recoveryMultiplier: 0.5, pairedEstadiaTibar: null },
  { quality: 'normal', recoveryMultiplier: 1.0, pairedEstadiaTibar: 0.5 },
  { quality: 'confortavel', recoveryMultiplier: 2.0, pairedEstadiaTibar: 4 },
  { quality: 'luxuosa', recoveryMultiplier: 3.0, pairedEstadiaTibar: 20 },
])

export function restRecoveryPerNight(
  quality: RestQuality,
  characterLevel: number,
): number {
  const row = REST_RECOVERY_TABLE.find((r) => r.quality === quality)!
  return Math.floor(row.recoveryMultiplier * characterLevel)
}

// ─── Tabela 8-7: Preço de Encantos (p334) ────────────────────────────
export type EnchantmentPriceRow = {
  enchantmentCount: 1 | 2 | 3
  priceIncreaseTibar: number
  cdIncrease: number
}

export const ENCHANTMENT_PRICE_TABLE: readonly EnchantmentPriceRow[] = Object.freeze([
  { enchantmentCount: 1, priceIncreaseTibar: 18000, cdIncrease: 10 },
  { enchantmentCount: 2, priceIncreaseTibar: 36000, cdIncrease: 15 },
  { enchantmentCount: 3, priceIncreaseTibar: 72000, cdIncrease: 20 },
])

/**
 * CD de Ofício para fabricar item mágico por tier (PDF p334, prose).
 *  - Menores: CD 30
 *  - Médios:  CD 40
 *  - Maiores: CD 50
 */
export const MAGIC_ITEM_CRAFT_CD: Record<'menor' | 'medio' | 'maior', number> =
  Object.freeze({
    menor: 30,
    medio: 40,
    maior: 50,
  })

/** PM sacrificado para fabricar item mágico por tier (PDF p334). */
export const MAGIC_ITEM_CRAFT_PM_SACRIFICE: Record<
  'menor' | 'medio' | 'maior',
  number
> = Object.freeze({
  menor: 1,
  medio: 2,
  maior: 3,
})
