/**
 * Unified activation registry — Fase 1 do redesign Poderes/Efeitos.
 *
 * Junta os 16 módulos `*-power-mechanics.ts` (ação + PM + usos por poder)
 * numa única tabela consultável por id completo, e deriva a TAXONOMIA que a
 * UI de Poderes renderiza:
 *
 *  - 'passive'            → sem botão (modifiers já fluem pela ficha)
 *  - 'triggered-passive'  → dispara quando a postura exigida liga
 *  - 'stance'             → liga/desliga com custo (FLAG_ACTIVATIONS)
 *  - 'instant'            → botão Usar N PM (com limite de usos quando houver)
 *
 * Chaves seguem a convenção de ids das habilidades:
 *  `class.<classe>.<slug>` · `race.<raça>.<slug>` · `origin.<origem>.<slug>`
 *  · `deus.<deus>.<slug>`.
 * `findActivationByName` cobre superfícies que só conhecem o nome.
 */
import { slug } from './abilities/classes/_helpers'
import type { AttributeKey } from './attributes'
import type { Modifier } from './items/types'
import { BARBARO_ELECTIVES } from './barbaro-power-mechanics'
import { BARDO_ELECTIVES } from './bardo-power-mechanics'
import { BUCANEIRO_ELECTIVES } from './bucaneiro-power-mechanics'
import { CACADOR_ELECTIVES } from './cacador-power-mechanics'
import { CAVALEIRO_ELECTIVES } from './cavaleiro-power-mechanics'
import { CLERIGO_ELECTIVES } from './clerigo-power-mechanics'
import { DIVINE_POWERS } from './divine-power-mechanics'
import { DRUIDA_ELECTIVES } from './druida-power-mechanics'
import { FLAG_ACTIVATIONS } from './flag-activations'
import { GUERREIRO_ELECTIVES } from './guerreiro-power-mechanics'
import { INVENTOR_ELECTIVES } from './inventor-power-mechanics'
import { LADINO_ELECTIVES } from './ladino-power-mechanics'
import { LUTADOR_ELECTIVES } from './lutador-power-mechanics'
import { NOBRE_ELECTIVES } from './nobre-power-mechanics'
import { ORIGEM_POWERS } from './origem-power-mechanics'
import { PALADINO_ELECTIVES } from './paladino-power-mechanics'
import { RACIAL_POWERS } from './racial-power-mechanics'

export type ActivationKind =
  | 'passive'
  | 'triggered-passive'
  | 'stance'
  | 'instant'

export type ActivationAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type ActivationUses = null | 'cena' | 'rodada' | 'dia' | number

export type ActivationScaling = {
  basePm: number
  stepPm: number
  /** O que cada passo compra ("+1 no bônus de Fúria"). */
  stepLabel: string
  /**
   * Nível NA CLASSE que destrava o primeiro passo, e de quantos em quantos
   * níveis vem o próximo.
   *
   * DADO, não função: o catálogo é servido por HTTP e uma função não sobrevive
   * ao JSON — o front recebia `scaling` sem o método e derrubava a cena ao
   * abrir qualquer postura que escala. Leia com `maxStepsForLevel`.
   */
  firstStepLevel: number
  stepEveryLevels: number
}

/**
 * Passos extras que o nível NA CLASSE concede (p40: multiclasse conta só os
 * níveis de Bárbaro para a Fúria).
 *
 * @example maxStepsForLevel({ firstStepLevel: 5, stepEveryLevels: 5, ... }, 10) // 2
 */
export function maxStepsForLevel(
  scaling: ActivationScaling,
  classLevel: number,
): number {
  if (classLevel < scaling.firstStepLevel) return 0
  return 1 + Math.floor((classLevel - scaling.firstStepLevel) / scaling.stepEveryLevels)
}

/**
 * Efeito PERSISTENTE que usar o poder concede (Fase 4) — vira um ActiveEffect
 * no backend (`POST /characters/:id/active-effects` com `{powerId}`).
 *
 *  - 'temp-hp'      → pool de PV temporários; o SERVIDOR calcula o total como
 *                     nível + atributo FINAL (`attribute`) via computeSheet.
 *  - 'active-effect'→ modifiers persistidos verbatim, escopo cena/dia.
 */
export type ActivationGrant =
  | { kind: 'temp-hp'; scope: 'scene'; attribute: AttributeKey }
  | { kind: 'active-effect'; scope: 'scene' | 'day'; modifiers: Modifier[] }

export type ActivationSpec = {
  id: string
  name: string
  kind: ActivationKind
  action: ActivationAction
  pmCost: number | 'variavel'
  uses: ActivationUses
  /** Flag de postura exigida pra ativar/disparar ('furia', 'inspiracao'). */
  requiresFlag?: string
  /** Presente só em posturas/escaláveis (hand-authored). */
  scaling?: ActivationScaling
  /** Efeito duradouro concedido ao usar/disparar (hand-authored). */
  grant?: ActivationGrant
  bookPage: number
}

type MechanicsRow = {
  name: string
  action: ActivationAction
  pmCost: number | 'variavel'
  uses: ActivationUses
  bookPage: number
  id?: string
  requiresFuria?: boolean
  requiresInspiracao?: boolean
}

function requiresFlagOf(row: MechanicsRow): string | undefined {
  if (row.requiresFuria) return 'furia'
  if (row.requiresInspiracao) return 'inspiracao'
  return undefined
}

function kindOf(row: MechanicsRow): ActivationKind {
  const flag = requiresFlagOf(row)
  if (row.action === 'passivo') return flag ? 'triggered-passive' : 'passive'
  return 'instant'
}

function specFrom(id: string, row: MechanicsRow): ActivationSpec {
  const requiresFlag = requiresFlagOf(row)
  return {
    id,
    name: row.name,
    kind: kindOf(row),
    action: row.action,
    pmCost: row.pmCost,
    uses: row.uses,
    ...(requiresFlag ? { requiresFlag } : {}),
    bookPage: row.bookPage,
  }
}

const CLASS_SOURCES: [string, readonly MechanicsRow[]][] = [
  ['Bárbaro', BARBARO_ELECTIVES],
  ['Bardo', BARDO_ELECTIVES],
  ['Bucaneiro', BUCANEIRO_ELECTIVES],
  ['Caçador', CACADOR_ELECTIVES],
  ['Cavaleiro', CAVALEIRO_ELECTIVES],
  ['Clérigo', CLERIGO_ELECTIVES],
  ['Druida', DRUIDA_ELECTIVES],
  ['Guerreiro', GUERREIRO_ELECTIVES],
  ['Inventor', INVENTOR_ELECTIVES],
  ['Ladino', LADINO_ELECTIVES],
  ['Lutador', LUTADOR_ELECTIVES],
  ['Nobre', NOBRE_ELECTIVES],
  ['Paladino', PALADINO_ELECTIVES],
]

/**
 * Posturas hand-authored — poderes AUTOMÁTICOS de classe (fora dos módulos de
 * eletivos) que ligam/desligam via FLAG_ACTIVATIONS. Fúria (p40): base 2 PM;
 * "a cada 5 níveis, pode gastar +1 PM para aumentar o bônus em +1".
 */
const STANCE_SPECS: ActivationSpec[] = [
  {
    id: 'class.barbaro.furia',
    name: FLAG_ACTIVATIONS.furia.name,
    kind: 'stance',
    action: 'livre',
    pmCost: FLAG_ACTIVATIONS.furia.pmCost,
    uses: null,
    scaling: {
      basePm: FLAG_ACTIVATIONS.furia.pmCost,
      stepPm: 1,
      stepLabel: '+1 no bônus de Fúria',
      firstStepLevel: 5,
      stepEveryLevels: 5,
    },
    bookPage: FLAG_ACTIVATIONS.furia.bookPage,
  },
  // Inspiração (p44): ação padrão + 2 PM, cena inteira; "a cada quatro
  // níveis, pode gastar +2 PM para aumentar o bônus em +1" (tiers nos níveis
  // 5/9/13/17 — Tabela 1-7).
  {
    id: 'class.bardo.inspiracao',
    name: FLAG_ACTIVATIONS.inspiracao.name,
    kind: 'stance',
    action: 'padrao',
    pmCost: FLAG_ACTIVATIONS.inspiracao.pmCost,
    uses: null,
    scaling: {
      basePm: FLAG_ACTIVATIONS.inspiracao.pmCost,
      stepPm: 2,
      stepLabel: '+1 no bônus de Inspiração',
      firstStepLevel: 5,
      stepEveryLevels: 4,
    },
    bookPage: FLAG_ACTIVATIONS.inspiracao.bookPage,
  },
]

/**
 * Grants hand-authored — só poderes cujo texto do livro é um efeito numérico
 * simples e DURADOURO (cena/dia). Efeitos parciais/ambíguos (ex.: Sangue de
 * Ferro, cuja RD 5 não tem ModifierTarget) ficam de fora até serem
 * expressáveis por completo.
 */
const ACTIVATION_GRANTS: Record<string, ActivationGrant> = {
  // Bárbaro p41: "quando entra em fúria, recebe PV temporários iguais ao seu
  // nível + Força". Servidor computa nível + Força FINAL.
  'class.barbaro.alma-de-bronze': {
    kind: 'temp-hp',
    scope: 'scene',
    attribute: 'strength',
  },
  // Cavaleiro p53 (Armadura da Honra): "No início de cada cena, você recebe
  // pontos de vida temporários iguais a seu nível + Carisma".
  'class.cavaleiro.armadura-da-honra': {
    kind: 'temp-hp',
    scope: 'scene',
    attribute: 'charisma',
  },
  // Origem Herói Camponês (Coração Heroico): "2 PM como ação livre para
  // receber PV temporários iguais ao seu nível + sua Carisma até o fim da cena".
  'origin.heroi-camponês.coracao-heroico': {
    kind: 'temp-hp',
    scope: 'scene',
    attribute: 'charisma',
  },
  // Bucaneiro p47 (En Garde): "até fim da cena, em arma corpo-a-corpo leve ou
  // ágil, recebe +2 margem de ameaça com essas armas e +2 Defesa".
  'class.bucaneiro.en-garde': {
    kind: 'active-effect',
    scope: 'scene',
    modifiers: [
      { target: { k: 'defense' }, amount: 2, bonusType: 'untyped', note: 'En Garde' },
      {
        target: { k: 'critRange' },
        amount: 2,
        bonusType: 'untyped',
        note: 'En Garde',
        condition: { c: 'context', note: 'com armas corpo a corpo leves ou ágeis' },
      },
    ],
  },
  // Dahllan p21 (Armadura de Allihanna): "+2 na Defesa até o fim da cena".
  'race.dahllan.armadura-de-allihanna': {
    kind: 'active-effect',
    scope: 'scene',
    modifiers: [
      {
        target: { k: 'defense' },
        amount: 2,
        bonusType: 'untyped',
        note: 'Armadura de Allihanna',
      },
    ],
  },
  // Khalmyr p133 (Dom da Verdade): "+5 em testes de Intuição, e em testes de
  // Percepção contra Enganação e Furtividade, até o fim da cena".
  'deus.khalmyr.dom-da-verdade': {
    kind: 'active-effect',
    scope: 'scene',
    modifiers: [
      {
        target: { k: 'expertise', name: 'Intuição' },
        amount: 5,
        bonusType: 'untyped',
        note: 'Dom da Verdade',
      },
      {
        target: { k: 'expertise', name: 'Percepção' },
        amount: 5,
        bonusType: 'untyped',
        note: 'Dom da Verdade',
        condition: { c: 'context', note: 'contra Enganação e Furtividade' },
      },
    ],
  },
}

/** Anexa os grants hand-authored, falhando alto em id que saiu do registro. */
function attachGrants(out: Map<string, ActivationSpec>): void {
  for (const [id, grant] of Object.entries(ACTIVATION_GRANTS)) {
    const spec = out.get(id)
    if (!spec) {
      throw new Error(
        `power-activation: grant for unknown id "${id}" — expected an id present in the activation registry`,
      )
    }
    out.set(id, { ...spec, grant })
  }
}

function buildRegistry(): Map<string, ActivationSpec> {
  const out = new Map<string, ActivationSpec>()
  const put = (spec: ActivationSpec) => {
    if (out.has(spec.id)) {
      throw new Error(
        `power-activation: duplicate id "${spec.id}" — expected unique keys per source`,
      )
    }
    out.set(spec.id, spec)
  }
  for (const [className, rows] of CLASS_SOURCES) {
    for (const row of rows) {
      put(specFrom(`class.${slug(className)}.${row.id ?? slug(row.name)}`, row))
    }
  }
  for (const row of RACIAL_POWERS as readonly (MechanicsRow & { racaId: string })[]) {
    put(specFrom(`race.${row.racaId}.${slug(row.name)}`, row))
  }
  for (const row of ORIGEM_POWERS as readonly (MechanicsRow & { origemId: string })[]) {
    put(specFrom(`origin.${row.origemId}.${slug(row.name)}`, row))
  }
  for (const row of DIVINE_POWERS as readonly (MechanicsRow & { deusId: string })[]) {
    put(specFrom(`deus.${row.deusId}.${slug(row.name)}`, row))
  }
  for (const spec of STANCE_SPECS) put(spec)
  attachGrants(out)
  return out
}

const REGISTRY = buildRegistry()

/** Todas as specs, pra varreduras (contagens, quick-bar defaults). */
export const ACTIVATION_SPECS: readonly ActivationSpec[] = Object.freeze([
  ...REGISTRY.values(),
])

/**
 * Lookup por id completo.
 *
 * @example getActivation('class.barbaro.golpe-poderoso')?.pmCost // 1
 */
export function getActivation(id: string): ActivationSpec | undefined {
  return REGISTRY.get(id)
}

/**
 * Fallback por nome (case/acento-insensível) pra superfícies que ainda não
 * carregam o id completo. Ambíguo entre fontes? Prefere poder de classe.
 *
 * @example findActivationByName('Golpe Poderoso')?.kind // 'instant'
 */
export function findActivationByName(name: string): ActivationSpec | undefined {
  const wanted = slug(name)
  let fallback: ActivationSpec | undefined
  for (const spec of REGISTRY.values()) {
    if (slug(spec.name) !== wanted) continue
    if (spec.id.startsWith('class.')) return spec
    fallback ??= spec
  }
  return fallback
}
