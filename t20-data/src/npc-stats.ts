/**
 * NPCs sem ficha completa — atalhos p259 (Cap 6, "Regras para NPCs").
 *
 * Duas ferramentas:
 *
 *   1. Categorias de Atitude (p259). NPC guarda uma categoria de atitude
 *      por PC. Determina o quanto NPC está disposto a fazer + modificador
 *      em testes de persuasão do PC (Diplomacia / Enganação / Intimidação).
 *
 *      Prestativo   +5    ajuda mesmo em perigo
 *      Amistoso      0    ajuda, mas não se arrisca
 *      Indiferente   0    padrão
 *      Inamistoso   −5    pode enganar / criar intriga
 *      Hostil    autofail vai prejudicar
 *
 *   2. Tabela 6-1: Estatísticas de NPCs (p259). Em vez de ficha completa,
 *      classifique o NPC em três patamares e leia bônus por perícia
 *      "forte" ou "fraca" (definidas pelo papel narrativo).
 *
 *      Iniciante   forte +5,  fraca +0    (guarda de cidade, mascate)
 *      Veterano    forte +10, fraca +3    (capitão da guarda, mercador próspero)
 *      Campeão     forte +15, fraca +6    (guarda-costas real, mestre de guilda)
 *
 * Ambas as regras aplicam-se a NPCs que não são combatentes principais;
 * inimigos com ND próprio devem usar `bestiary.ts` ou uma ficha completa.
 */

export const NPC_ATTITUDES = [
  'prestativo',
  'amistoso',
  'indiferente',
  'inamistoso',
  'hostil',
] as const

export type NpcAttitude = (typeof NPC_ATTITUDES)[number]

export type NpcAttitudeRow = {
  attitude: NpcAttitude
  label: string
  persuasionModifier: number | 'auto-fail'
  description: string
}

export const NPC_ATTITUDE_TABLE: readonly NpcAttitudeRow[] = Object.freeze([
  {
    attitude: 'prestativo',
    label: 'Prestativo',
    persuasionModifier: 5,
    description: 'Adora o personagem e pode ajudá-lo, mesmo correndo perigo.',
  },
  {
    attitude: 'amistoso',
    label: 'Amistoso',
    persuasionModifier: 0,
    description: 'Gosta do personagem e pode ajudá-lo, mas dificilmente se arriscará.',
  },
  {
    attitude: 'indiferente',
    label: 'Indiferente',
    persuasionModifier: 0,
    description: 'Não gosta nem desgosta. Tratamento socialmente esperado. Padrão.',
  },
  {
    attitude: 'inamistoso',
    label: 'Inamistoso',
    persuasionModifier: -5,
    description: 'Desgosta do personagem sem se arriscar para prejudicá-lo — engana, cria intriga.',
  },
  {
    attitude: 'hostil',
    label: 'Hostil',
    persuasionModifier: 'auto-fail',
    description: 'Odeia o personagem e tenta prejudicá-lo, mesmo correndo perigo.',
  },
])

/**
 * Modificador em testes de persuasão (Diplomacia / Enganação / Intimidação)
 * do PC contra NPC com dada atitude. `null` = falha automática (hostil).
 */
export function persuasionModifierForAttitude(
  attitude: NpcAttitude,
): number | null {
  const row = NPC_ATTITUDE_TABLE.find((r) => r.attitude === attitude)
  if (!row) throw new Error(`Unknown NpcAttitude: ${attitude}`)
  return row.persuasionModifier === 'auto-fail' ? null : row.persuasionModifier
}

/** Whether a persuasion check against this attitude auto-fails (hostil). */
export function isPersuasionAutoFail(attitude: NpcAttitude): boolean {
  return attitude === 'hostil'
}

// ─── Tabela 6-1: Estatísticas de NPCs (p259) ─────────────────────────

export const NPC_TIERS = ['iniciante', 'veterano', 'campeao'] as const

export type NpcTier = (typeof NPC_TIERS)[number]

export type NpcTierRow = {
  tier: NpcTier
  label: string
  strongSkill: number
  weakSkill: number
  examples: readonly string[]
}

export const NPC_TIER_TABLE: readonly NpcTierRow[] = Object.freeze([
  {
    tier: 'iniciante',
    label: 'Iniciante',
    strongSkill: 5,
    weakSkill: 0,
    examples: ['guarda de cidade', 'mascate'],
  },
  {
    tier: 'veterano',
    label: 'Veterano',
    strongSkill: 10,
    weakSkill: 3,
    examples: ['capitão da guarda', 'mercador próspero'],
  },
  {
    tier: 'campeao',
    label: 'Campeão',
    strongSkill: 15,
    weakSkill: 6,
    examples: ['guarda-costas real', 'mestre de guilda'],
  },
])

export type NpcSkillStrength = 'strong' | 'weak'

/**
 * Bônus de perícia para um NPC de dado patamar. Usa a coluna "forte" ou
 * "fraca" conforme o papel do NPC no mundo — ex.: mercador tem Diplomacia,
 * Ofício e Vontade como fortes; demais perícias são fracas.
 */
export function npcSkillBonus(
  tier: NpcTier,
  strength: NpcSkillStrength,
): number {
  const row = NPC_TIER_TABLE.find((r) => r.tier === tier)
  if (!row) throw new Error(`Unknown NpcTier: ${tier}`)
  return strength === 'strong' ? row.strongSkill : row.weakSkill
}
