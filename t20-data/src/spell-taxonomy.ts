/**
 * Spell taxonomy — tipos de efeito + tipos de dano + metadata das
 * 8 escolas de magia.
 *
 * PDF refs:
 *  - Escolas: Cap 4 p172-173 (Características de magias)
 *  - Tipos de efeito: Cap 5 p227-228 (interação com imunidades)
 *  - Tipos de dano: Cap 5 p230 (subdivisão de Dano)
 *
 * NOTA: T20 não usa "descritor" — divide em "tipos de efeito" e
 * "tipos de dano". Escolas contam como tipos de efeito (p172).
 * Não há escolas opostas/proibidas nem save fixo por escola.
 */

// ─── Tipos de efeito (p227-228) ─────────────────────────────────────
export const EFFECT_TYPES = [
  'arcano',
  'atordoamento',
  'cansaco',
  'climatico',
  'cura',
  'dano',
  'divino',
  'luz',
  'magico',
  'medo',
  'mental',
  'metabolismo',
  'metamorfose',
  'movimento',
  'perda-de-vida',
  'sentidos',
  'trevas',
  'veneno',
] as const
export type EffectType = (typeof EFFECT_TYPES)[number]

export type EffectCategory =
  | 'origem'
  | 'condicao'
  | 'ambiental'
  | 'beneficio'
  | 'energia'
  | 'mental'
  | 'fisiologico'
  | 'fisico'
  | 'dano-especial'
  | 'dano-generico'

export type EffectMeta = {
  type: EffectType
  category: EffectCategory
  /** Criaturas naturalmente imunes (INT nula, construtos, mortos-vivos, etc). */
  immuneCreatures: readonly string[]
  description: string
}

export const EFFECT_META: Readonly<Record<EffectType, EffectMeta>> =
  Object.freeze({
    arcano: {
      type: 'arcano',
      category: 'origem',
      immuneCreatures: [],
      description: 'Gerado pelas energias místicas de Arton. Sempre mágico.',
    },
    atordoamento: {
      type: 'atordoamento',
      category: 'condicao',
      immuneCreatures: [],
      description: 'Afeta a capacidade de agir do alvo.',
    },
    cansaco: {
      type: 'cansaco',
      category: 'condicao',
      immuneCreatures: ['construto', 'morto-vivo'],
      description: 'Diminui capacidades físicas.',
    },
    climatico: {
      type: 'climatico',
      category: 'ambiental',
      immuneCreatures: [],
      description: 'Gerado pelas forças da natureza.',
    },
    cura: {
      type: 'cura',
      category: 'beneficio',
      immuneCreatures: [],
      description: 'Cura pontos de vida do alvo.',
    },
    dano: {
      type: 'dano',
      category: 'dano-generico',
      immuneCreatures: [],
      description: 'Reduz PV. Subdividido em tipos de dano (p230).',
    },
    divino: {
      type: 'divino',
      category: 'origem',
      immuneCreatures: [],
      description: 'Gerado por energia de um deus. Sempre mágico.',
    },
    luz: {
      type: 'luz',
      category: 'energia',
      immuneCreatures: [],
      description: 'Dano/cura de luz, iluminação e energia positiva.',
    },
    magico: {
      type: 'magico',
      category: 'origem',
      immuneCreatures: [],
      description: 'Energizado por forças arcanas ou divinas. Subdivide em escolas.',
    },
    medo: {
      type: 'medo',
      category: 'mental',
      immuneCreatures: ['inteligencia-nula'],
      description: 'Medo capaz de prejudicar.',
    },
    mental: {
      type: 'mental',
      category: 'mental',
      immuneCreatures: ['inteligencia-nula'],
      description: 'Afeta a mente.',
    },
    metabolismo: {
      type: 'metabolismo',
      category: 'fisiologico',
      immuneCreatures: ['construto', 'morto-vivo'],
      description: 'Doenças, sangramento, fome.',
    },
    metamorfose: {
      type: 'metamorfose',
      category: 'fisico',
      immuneCreatures: [],
      description: 'Altera forma/composição corporal. Inclui petrificação.',
    },
    movimento: {
      type: 'movimento',
      category: 'fisico',
      immuneCreatures: [],
      description: 'Afeta ou remove capacidade de se movimentar.',
    },
    'perda-de-vida': {
      type: 'perda-de-vida',
      category: 'dano-especial',
      immuneCreatures: [],
      description: 'Reduz PV; ignora redução de dano.',
    },
    sentidos: {
      type: 'sentidos',
      category: 'fisico',
      immuneCreatures: [],
      description: 'Afeta sentidos físicos (cego, surdo).',
    },
    trevas: {
      type: 'trevas',
      category: 'energia',
      immuneCreatures: [],
      description: 'Necromancia, escuridão, energia negativa.',
    },
    veneno: {
      type: 'veneno',
      category: 'fisiologico',
      immuneCreatures: ['construto', 'morto-vivo'],
      description: 'Efeitos de venenos.',
    },
  })

// ─── Tipos de dano (p230) ───────────────────────────────────────────
export const DAMAGE_TYPES = [
  'acido',
  'corte',
  'eletricidade',
  'essencia',
  'fogo',
  'frio',
  'impacto',
  'luz',
  'perfuracao',
  'psiquico',
  'trevas',
] as const
export type SpellDamageType = (typeof DAMAGE_TYPES)[number]

export type DamageClass = 'fisico' | 'energia'

export type DamageMeta = {
  type: SpellDamageType
  class: DamageClass
  /** Elemento associado (ar/água/terra/fogo) ou null. */
  element: 'ar' | 'agua' | 'terra' | 'fogo' | null
  description: string
}

export const DAMAGE_META: Readonly<Record<SpellDamageType, DamageMeta>> =
  Object.freeze({
    acido: {
      type: 'acido',
      class: 'energia',
      element: 'terra',
      description: 'Ligado ao elemento terra.',
    },
    corte: {
      type: 'corte',
      class: 'fisico',
      element: null,
      description: 'Armas afiadas.',
    },
    eletricidade: {
      type: 'eletricidade',
      class: 'energia',
      element: 'ar',
      description: 'Ligada ao elemento ar.',
    },
    essencia: {
      type: 'essencia',
      class: 'energia',
      element: null,
      description: 'Energia mágica pura (ex.: Seta Infalível de Talude).',
    },
    fogo: {
      type: 'fogo',
      class: 'energia',
      element: 'fogo',
      description: 'Elemento fogo.',
    },
    frio: {
      type: 'frio',
      class: 'energia',
      element: 'agua',
      description: 'Elemento água.',
    },
    impacto: {
      type: 'impacto',
      class: 'fisico',
      element: null,
      description: 'Contusão, sônico, quedas, explosões.',
    },
    luz: {
      type: 'luz',
      class: 'energia',
      element: null,
      description: 'Divindades bondosas.',
    },
    perfuracao: {
      type: 'perfuracao',
      class: 'fisico',
      element: null,
      description: 'Pontudas, mordidas.',
    },
    psiquico: {
      type: 'psiquico',
      class: 'energia',
      element: null,
      description: 'Ataques mentais.',
    },
    trevas: {
      type: 'trevas',
      class: 'energia',
      element: null,
      description: 'Necromancia, divindades malignas.',
    },
  })

// ─── Escolas de magia metadata (p172-173) ───────────────────────────
import type { SpellSchool } from './spells'

export type SpellSchoolMeta = {
  school: SpellSchool
  abbrev: string
  focus: string
  /** Effect types que TODA magia da escola aplica automaticamente. */
  intrinsicEffects: readonly EffectType[]
}

/**
 * Escolas + effects intrínsecos (p172):
 *  - Encantamento: todas são efeitos mentais.
 *  - Ilusão: todas são efeitos mentais.
 *  - Necromancia: todas são efeitos de trevas.
 * Demais escolas: sem efeito intrínseco (varia por magia).
 */
export const SCHOOL_META: Readonly<Record<SpellSchool, SpellSchoolMeta>> =
  Object.freeze({
    abjuracao: {
      school: 'abjuracao',
      abbrev: 'Abjur',
      focus: 'Proteção; anula magias ou expulsa criaturas convocadas.',
      intrinsicEffects: [],
    },
    adivinhacao: {
      school: 'adivinhacao',
      abbrev: 'Adiv',
      focus: 'Detecção; vasculha passado e futuro.',
      intrinsicEffects: [],
    },
    convocacao: {
      school: 'convocacao',
      abbrev: 'Conv',
      focus: 'Transporta matéria pelo Éter Entre Mundos.',
      intrinsicEffects: [],
    },
    encantamento: {
      school: 'encantamento',
      abbrev: 'Encan',
      focus: 'Afeta a mente.',
      intrinsicEffects: ['mental'],
    },
    evocacao: {
      school: 'evocacao',
      abbrev: 'Evoc',
      focus: 'Manipula ou gera energia pura (ácido/eletricidade/fogo/frio/luz/essência).',
      intrinsicEffects: [],
    },
    ilusao: {
      school: 'ilusao',
      abbrev: 'Ilusão',
      focus: 'Faz outros perceberem algo inexistente ou ignorarem algo real.',
      intrinsicEffects: ['mental'],
    },
    necromancia: {
      school: 'necromancia',
      abbrev: 'Necro',
      focus: 'Canaliza energia negativa; escuridão, dreno vital, mortos-vivos.',
      intrinsicEffects: ['trevas'],
    },
    transmutacao: {
      school: 'transmutacao',
      abbrev: 'Trans',
      focus: 'Altera propriedades físicas de criatura ou objeto.',
      intrinsicEffects: [],
    },
  })

// ─── Helpers ────────────────────────────────────────────────────────
export function effectMeta(type: EffectType): EffectMeta {
  return EFFECT_META[type]
}

export function damageMeta(type: SpellDamageType): DamageMeta {
  return DAMAGE_META[type]
}

export function schoolMeta(school: SpellSchool): SpellSchoolMeta {
  return SCHOOL_META[school]
}

/**
 * Effect types automaticamente aplicados a uma magia por conta de
 * sua escola. Ex.: qualquer magia de Encantamento é `mental`.
 */
export function intrinsicEffectsForSchool(
  school: SpellSchool,
): readonly EffectType[] {
  return SCHOOL_META[school].intrinsicEffects
}
