/**
 * Lista de Condições (PDF book p394-395). Mechanical effects pinned from
 * the rulebook. T20 condition system rules:
 *
 *   - "Condições com os mesmos efeitos não se acumulam; aplique apenas
 *     o mais severo."
 *   - "A menos que especificado, condições terminam no fim da cena."
 *   - Some conditions carry a *tipo de efeito* (Medo, Mental, …) — used
 *     by other rules (immunities, recovery sources, etc.).
 *   - A handful of conditions "upgrade" if applied again: a fresh
 *     Abalado on an already-Abalado target becomes Apavorado, etc.
 *
 * Upgrade chains:
 *   Fraco → Debilitado → Inconsciente
 *   Frustrado → Esmorecido
 *   Fatigado → Exausto → Inconsciente
 *   Abalado → Apavorado
 */
export const CONDITION_TAGS = [
  'medo',
  'mental',
  'movimento',
  'metabolismo',
  'sentidos',
  'cansaco',
  'veneno',
  'metamorfose',
] as const

export type ConditionTag = (typeof CONDITION_TAGS)[number]

export const CONDITION_IDS = [
  'abalado',
  'agarrado',
  'alquebrado',
  'apavorado',
  'atordoado',
  'caido',
  'cego',
  'confuso',
  'debilitado',
  'desprevenido',
  'doente',
  'em-chamas',
  'enfeitiçado',
  'enjoado',
  'enredado',
  'envenenado',
  'esmorecido',
  'exausto',
  'fascinado',
  'fatigado',
  'fraco',
  'frustrado',
  'imovel',
  'inconsciente',
  'indefeso',
  'lento',
  'ofuscado',
  'paralisado',
  'pasmo',
  'petrificado',
  'sangrando',
  'sobrecarregado',
  'surdo',
  'surpreendido',
  'vulneravel',
] as const

export type ConditionId = (typeof CONDITION_IDS)[number]

export type Condition = {
  id: ConditionId
  name: string
  description: string
  tags: ConditionTag[]
  /** When applying this condition to a target who already has it, the
   *  condition is *replaced* by `upgradesTo` (PDF p394 sidebar). */
  upgradesTo?: ConditionId
}

export const CONDITIONS: Record<ConditionId, Condition> = {
  abalado: {
    id: 'abalado',
    name: 'Abalado',
    description: '-2 em testes de perícia.',
    tags: ['medo'],
    upgradesTo: 'apavorado',
  },
  agarrado: {
    id: 'agarrado',
    name: 'Agarrado',
    description:
      'Desprevenido e imóvel; -2 em ataques; só pode usar armas leves; ataques à distância 50% acertam alvo errado.',
    tags: ['movimento'],
  },
  alquebrado: {
    id: 'alquebrado',
    name: 'Alquebrado',
    description: 'Custo em mana das habilidades +1.',
    tags: ['mental'],
  },
  apavorado: {
    id: 'apavorado',
    name: 'Apavorado',
    description:
      '-5 em testes de perícia; não pode se aproximar da fonte do medo.',
    tags: ['medo'],
  },
  atordoado: {
    id: 'atordoado',
    name: 'Atordoado',
    description: 'Desprevenido e não pode fazer ações.',
    tags: ['mental'],
  },
  caido: {
    id: 'caido',
    name: 'Caído',
    description:
      '-5 Defesa vs corpo-a-corpo, +5 vs distância (cumulativo); -5 ataques corpo-a-corpo; deslocamento 1,5m.',
    tags: [],
  },
  cego: {
    id: 'cego',
    name: 'Cego',
    description:
      'Desprevenido e lento; sem testes de Percepção visuais; -5 perícias baseadas em Força/Destreza; alvos têm camuflagem total.',
    tags: ['sentidos'],
  },
  confuso: {
    id: 'confuso',
    name: 'Confuso',
    description:
      'Rola 1d6 por turno: 1) move aleatório; 2-3) balbucia; 4-5) ataca criatura mais próxima; 6) condição termina.',
    tags: ['mental'],
  },
  debilitado: {
    id: 'debilitado',
    name: 'Debilitado',
    description:
      '-5 testes de Força/Destreza/Constituição e perícias baseadas.',
    tags: [],
    upgradesTo: 'inconsciente',
  },
  desprevenido: {
    id: 'desprevenido',
    name: 'Desprevenido',
    description:
      '-5 na Defesa e em Reflexos; desprevenido contra inimigos que não pode perceber.',
    tags: [],
  },
  doente: {
    id: 'doente',
    name: 'Doente',
    description: 'Sob efeito de uma doença.',
    tags: ['metabolismo'],
  },
  'em-chamas': {
    id: 'em-chamas',
    name: 'Em Chamas',
    description:
      'Pegando fogo; 1d6 dano de fogo no início do turno; ação padrão para apagar (ou imersão em água).',
    tags: [],
  },
  enfeitiçado: {
    id: 'enfeitiçado',
    name: 'Enfeitiçado',
    description:
      'Vê a fonte favoravelmente; a fonte recebe +10 em Diplomacia.',
    tags: ['mental'],
  },
  enjoado: {
    id: 'enjoado',
    name: 'Enjoado',
    description:
      'Apenas uma ação padrão OU de movimento por rodada (não ambas); pode investir até o deslocamento (sem dobrar).',
    tags: ['metabolismo'],
  },
  enredado: {
    id: 'enredado',
    name: 'Enredado',
    description: 'Lento, vulnerável e -2 em ataques.',
    tags: ['movimento'],
  },
  envenenado: {
    id: 'envenenado',
    name: 'Envenenado',
    description:
      'Varia pelo veneno; geralmente perda de vida recorrente cumulativa.',
    tags: ['veneno'],
  },
  esmorecido: {
    id: 'esmorecido',
    name: 'Esmorecido',
    description:
      '-5 Inteligência/Sabedoria/Carisma e perícias baseadas.',
    tags: ['mental'],
  },
  exausto: {
    id: 'exausto',
    name: 'Exausto',
    description: 'Debilitado, lento e vulnerável.',
    tags: ['cansaco'],
    upgradesTo: 'inconsciente',
  },
  fascinado: {
    id: 'fascinado',
    name: 'Fascinado',
    description:
      '-5 Percepção; sem ações exceto observar; anulada por ações hostis contra o personagem.',
    tags: ['mental'],
  },
  fatigado: {
    id: 'fatigado',
    name: 'Fatigado',
    description: 'Fraco e vulnerável.',
    tags: ['cansaco'],
    upgradesTo: 'exausto',
  },
  fraco: {
    id: 'fraco',
    name: 'Fraco',
    description:
      '-2 em testes de Força/Destreza/Constituição e perícias baseadas.',
    tags: [],
    upgradesTo: 'debilitado',
  },
  frustrado: {
    id: 'frustrado',
    name: 'Frustrado',
    description:
      '-2 em Inteligência/Sabedoria/Carisma e perícias baseadas.',
    tags: ['mental'],
    upgradesTo: 'esmorecido',
  },
  imovel: {
    id: 'imovel',
    name: 'Imóvel',
    description: 'Todas formas de deslocamento reduzidas a 0m.',
    tags: ['movimento'],
  },
  inconsciente: {
    id: 'inconsciente',
    name: 'Inconsciente',
    description:
      'Indefeso; sem ações (incluindo reações); pode rolar Constituição para estabilizar sangramento; ação padrão de outro personagem para acordá-lo.',
    tags: [],
  },
  indefeso: {
    id: 'indefeso',
    name: 'Indefeso',
    description:
      'Desprevenido; -10 na Defesa; falha automaticamente em Reflexos; pode sofrer golpe de misericórdia.',
    tags: [],
  },
  lento: {
    id: 'lento',
    name: 'Lento',
    description:
      'Deslocamentos pela metade (arredondado para baixo no primeiro incremento de 1,5m); não pode correr ou investir.',
    tags: ['movimento'],
  },
  ofuscado: {
    id: 'ofuscado',
    name: 'Ofuscado',
    description: '-2 em ataque e Percepção.',
    tags: ['sentidos'],
  },
  paralisado: {
    id: 'paralisado',
    name: 'Paralisado',
    description: 'Imóvel e indefeso; só ações mentais.',
    tags: ['movimento'],
  },
  pasmo: {
    id: 'pasmo',
    name: 'Pasmo',
    description: 'Não pode fazer ações.',
    tags: ['mental'],
  },
  petrificado: {
    id: 'petrificado',
    name: 'Petrificado',
    description: 'Inconsciente e redução de dano 8.',
    tags: ['metamorfose'],
  },
  sangrando: {
    id: 'sangrando',
    name: 'Sangrando',
    description:
      'No início do turno, Constituição CD 15. Falha = perde 1d6 PV e continua; sucesso remove a condição.',
    tags: ['metabolismo'],
  },
  sobrecarregado: {
    id: 'sobrecarregado',
    name: 'Sobrecarregado',
    description: 'Penalidade de armadura -5; deslocamento -3m.',
    tags: ['movimento'],
  },
  surdo: {
    id: 'surdo',
    name: 'Surdo',
    description:
      'Sem testes de Percepção auditivos; -5 em Iniciativa; condição ruim para lançar magias.',
    tags: ['sentidos'],
  },
  surpreendido: {
    id: 'surpreendido',
    name: 'Surpreendido',
    description: 'Desprevenido e não pode fazer ações.',
    tags: [],
  },
  vulneravel: {
    id: 'vulneravel',
    name: 'Vulnerável',
    description: '-2 na Defesa.',
    tags: [],
  },
}

/**
 * Apply a condition id to an active-condition set, honoring PDF upgrade
 * rules ("se ficar X novamente, vire Y"). Returns a *new* Set so the
 * caller can use it for diffs. When the condition has an upgrade and is
 * already present, the source condition is **replaced** by the upgrade.
 */
export function applyCondition(
  active: ReadonlySet<ConditionId>,
  id: ConditionId,
): Set<ConditionId> {
  const next = new Set(active)
  if (!active.has(id)) {
    next.add(id)
    return next
  }
  const entry = CONDITIONS[id]
  if (entry.upgradesTo) {
    next.delete(id)
    next.add(entry.upgradesTo)
  }
  return next
}

export function removeCondition(
  active: ReadonlySet<ConditionId>,
  id: ConditionId,
): Set<ConditionId> {
  const next = new Set(active)
  next.delete(id)
  return next
}

/**
 * Linear chain of upgrades starting at `id`. Useful for the UI to show
 * "Fraco → Debilitado → Inconsciente". Walks `upgradesTo` and stops at
 * the first id that has no upgrade.
 */
export function conditionUpgradeChain(id: ConditionId): ConditionId[] {
  const chain: ConditionId[] = [id]
  const seen = new Set<ConditionId>([id])
  let current: ConditionId | undefined = CONDITIONS[id]?.upgradesTo
  while (current && !seen.has(current)) {
    chain.push(current)
    seen.add(current)
    current = CONDITIONS[current]?.upgradesTo
  }
  return chain
}
