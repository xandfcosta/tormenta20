/**
 * Parceiros — regras de aliados/companheiros/montarias (PDF Cap 6 p260-262).
 *
 * Um parceiro tem sempre um **tipo** + **patamar** (iniciante/veterano/mestre).
 * Não age no turno próprio: fornece bônus ao PJ. Não pode ser alvo de ataques
 * hostis (variante Vulneráveis em p262).
 *
 * Este módulo cobre:
 *  - Limite de parceiros por nível (p260 tabela)
 *  - Tier de Companheiro Animal por nível de druida/caçador (p50, p61-62)
 *  - Tier de Montaria por nível de cavaleiro (p54-55)
 *  - Dado da variante "Parceiros Vulneráveis" (p262)
 *  - Catálogo dos 12 tipos + boxes de menção no PDF
 *
 * NÃO cobre (fora de escopo — depende de resolver de combate):
 *  - Bônus mecânico por (tipo × patamar): descrito em prosa complexa p260-261.
 *  - Stat blocks de montarias específicas (Cavalo, Grifo, etc — p262).
 *  - Regras de combate montado (p261-262 Cavalgar).
 */

export type ParceiroTier = 'iniciante' | 'veterano' | 'mestre'

export const PARCEIRO_TIERS: readonly ParceiroTier[] = Object.freeze([
  'iniciante',
  'veterano',
  'mestre',
])

export type ParceiroType =
  | 'adepto'
  | 'ajudante'
  | 'assassino'
  | 'atirador'
  | 'combatente'
  | 'destruidor'
  | 'fortao'
  | 'guardiao'
  | 'magivocador'
  | 'medico'
  | 'perseguidor'
  | 'vigilante'

export const PARCEIRO_TYPES: readonly ParceiroType[] = Object.freeze([
  'adepto',
  'ajudante',
  'assassino',
  'atirador',
  'combatente',
  'destruidor',
  'fortao',
  'guardiao',
  'magivocador',
  'medico',
  'perseguidor',
  'vigilante',
])

/** Rótulo em PT-BR para UI. */
export const PARCEIRO_TYPE_LABELS: Readonly<Record<ParceiroType, string>> =
  Object.freeze({
    adepto: 'Adepto',
    ajudante: 'Ajudante',
    assassino: 'Assassino',
    atirador: 'Atirador',
    combatente: 'Combatente',
    destruidor: 'Destruidor',
    fortao: 'Fortão',
    guardiao: 'Guardião',
    magivocador: 'Magivocador',
    medico: 'Médico',
    perseguidor: 'Perseguidor',
    vigilante: 'Vigilante',
  })

/**
 * Limite de parceiros ativos por nível do PJ (PDF p260):
 *  - 1º-4º: 1 parceiro
 *  - 5º-16º: 2 parceiros
 *  - 17º-20º: 3 parceiros
 */
export function parceiroLimit(pcLevel: number): 1 | 2 | 3 {
  if (pcLevel < 1) {
    throw new Error(`parceiroLimit: pcLevel must be ≥ 1, got ${pcLevel}`)
  }
  if (pcLevel <= 4) return 1
  if (pcLevel <= 16) return 2
  return 3
}

/**
 * Tier do Companheiro Animal de druida/caçador (PDF p50, p61-62):
 *  - 1º-6º: iniciante
 *  - 7º-14º: veterano
 *  - 15º+: mestre
 */
export function companheiroAnimalTier(classLevel: number): ParceiroTier {
  if (classLevel < 1) {
    throw new Error(
      `companheiroAnimalTier: classLevel must be ≥ 1, got ${classLevel}`,
    )
  }
  if (classLevel < 7) return 'iniciante'
  if (classLevel < 15) return 'veterano'
  return 'mestre'
}

/**
 * Tier da Montaria do Cavaleiro (Caminho da Montaria, PDF p54-55):
 *  - <11º: veterano (baseline da habilidade)
 *  - 11º+: mestre
 * NOTA: montarias compradas fora de habilidade de classe são sempre
 * iniciante (p262). Este helper é para o Caminho.
 */
export function cavaleiroMontariaTier(cavaleiroLevel: number): ParceiroTier {
  if (cavaleiroLevel < 1) {
    throw new Error(
      `cavaleiroMontariaTier: cavaleiroLevel must be ≥ 1, got ${cavaleiroLevel}`,
    )
  }
  if (cavaleiroLevel < 11) return 'veterano'
  return 'mestre'
}

/** Dado da variante "Parceiros Vulneráveis" por patamar (PDF p262 sidebar). */
export const VULNERABLE_PARTNER_DIE: Readonly<Record<ParceiroTier, 'd4' | 'd6' | 'd8'>> =
  Object.freeze({
    iniciante: 'd4',
    veterano: 'd6',
    mestre: 'd8',
  })

export function vulnerablePartnerDie(tier: ParceiroTier): 'd4' | 'd6' | 'd8' {
  return VULNERABLE_PARTNER_DIE[tier]
}
