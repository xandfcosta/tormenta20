/**
 * Armadilhas — PDF Cap 7 Aventura, livro p317-318.
 *
 * Sistema (p317):
 *  - **Detecção**: perícia **Investigação** (Int) via uso "Procurar".
 *    PDF NÃO usa Percepção para encontrar armadilhas.
 *  - **Desarme**: perícia **Ladinagem** (Des, **Apenas Treinada**).
 *    Não-treinado em Ladinagem NÃO pode desarmar armadilhas.
 *    Tempo: 1d4 rodadas (ou ação completa com -5). Falha por 5+ ativa.
 *  - **ND da armadilha** = funciona como ND de criatura. Use a mesma
 *    tabela XP-por-ND para a recompensa ao detectar/desarmar/sobreviver.
 *  - **NÃO há tiers categóricos**. Cada armadilha é individual com ND
 *    próprio. Único "tipo" formal: `magica: boolean` (nomes em itálico
 *    no PDF — anuláveis por Dissipar Magia).
 *  - `triggerType` neste módulo é **inferência narrativa** baseada na
 *    descrição PDF — NÃO é taxonomia formal do livro.
 */

export type TrapTriggerType =
  | 'pressao'
  | 'tripwire'
  | 'proximidade'
  | 'manipulacao'
  | 'magica'

export type TrapSaveType = 'fortitude' | 'reflexos' | 'vontade' | 'none'
export type TrapSaveEffect = 'anula' | 'metade' | 'parcial'

export type Trap = {
  id: string
  name: string
  /** ND da armadilha; segue tabela XP-por-ND de criaturas. 0.25 = ND 1/4. */
  nd: number
  triggerType: TrapTriggerType
  /** PDF p317: Investigação. Sempre Investigação para encontrar armadilhas. */
  detectSkill: 'investigacao'
  detectCd: number
  /** Ladinagem, Apenas Treinada (PDF p111). */
  disarmCd: number
  /** PT-BR; null quando o efeito é apenas condição (sem dano direto). */
  damage: string | null
  save: {
    type: TrapSaveType
    cd: number
    effect: TrapSaveEffect | null
  }
  effect: string
  /** True quando a armadilha é mágica (itálico no PDF; Dissipar Magia funciona). */
  magica: boolean
  bookPage: number
}

/** PDF p111 — sem treino em Ladinagem, sem desarme. */
export const TRAP_DISARM_REQUIRES_TRAINED_LADINAGEM = true

/** PDF p120 — desarme leva 1d4 rodadas (ou ação completa com -5). */
export const TRAP_DISARM_BASE_TIME = '1d4 rodadas'

/** PDF p120 — falha por 5+ ativa a armadilha. */
export const TRAP_DISARM_CRITICAL_FAILURE_MARGIN = 5

export const TRAPS: readonly Trap[] = Object.freeze([
  {
    id: 'agulha-envenenada',
    name: 'Agulha Envenenada',
    nd: 0.25,
    triggerType: 'manipulacao',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 20,
    damage: '1 ponto de perfuração + 1d12 PV de veneno',
    save: { type: 'reflexos', cd: 20, effect: 'anula' },
    effect:
      'A vítima sofre 1 ponto de perfuração e perde 1d12 PV por veneno.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'arame-farpado',
    name: 'Arame Farpado',
    nd: 0.25,
    triggerType: 'proximidade',
    detectSkill: 'investigacao',
    detectCd: 10,
    disarmCd: 20,
    damage: '1d6+2 corte',
    save: { type: 'none', cd: 0, effect: null },
    effect:
      'Conta como terreno difícil e causa 1d6+2 pontos de dano de corte em quem atravessá-lo.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'fosso-camuflado',
    name: 'Fosso Camuflado',
    nd: 0.25,
    triggerType: 'pressao',
    detectSkill: 'investigacao',
    detectCd: 20,
    disarmCd: 20,
    damage: '2d6 impacto',
    save: { type: 'reflexos', cd: 20, effect: 'anula' },
    effect:
      'Queda de 3m causa 2d6 de impacto. Atletismo CD 20 para escalar de volta.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'rede',
    name: 'Rede',
    nd: 0.25,
    triggerType: 'tripwire',
    detectSkill: 'investigacao',
    detectCd: 20,
    disarmCd: 20,
    damage: null,
    save: { type: 'reflexos', cd: 20, effect: 'anula' },
    effect:
      'Criatura fica agarrada. Ação completa + Acrobacia CD 20 para escapar.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'virote',
    name: 'Virote',
    nd: 0.25,
    triggerType: 'tripwire',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 20,
    damage: '1d10+2 perfuração',
    save: { type: 'reflexos', cd: 20, effect: 'anula' },
    effect: 'Dispara um virote que causa 1d10+2 de perfuração.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'fosso-profundo',
    name: 'Fosso Profundo',
    nd: 0.5,
    triggerType: 'pressao',
    detectSkill: 'investigacao',
    detectCd: 20,
    disarmCd: 20,
    damage: '4d6 impacto',
    save: { type: 'reflexos', cd: 20, effect: 'anula' },
    effect:
      'Queda de 6m causa 4d6 de impacto. Atletismo CD 20 para escalar de volta.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'lamina-na-parede',
    name: 'Lâmina na Parede',
    nd: 0.5,
    triggerType: 'tripwire',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 20,
    damage: '2d6+5 corte',
    save: { type: 'reflexos', cd: 20, effect: 'anula' },
    effect: 'Uma lâmina sai da parede causando 2d6+5 de corte.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'bloco-de-pedra',
    name: 'Bloco de Pedra',
    nd: 1,
    triggerType: 'pressao',
    detectSkill: 'investigacao',
    detectCd: 20,
    disarmCd: 20,
    damage: '6d6 impacto',
    save: { type: 'reflexos', cd: 20, effect: 'anula' },
    effect: 'Um bloco de pedra cai causando 6d6 de impacto.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'pendulo-de-teto',
    name: 'Pêndulo de Teto',
    nd: 1,
    triggerType: 'tripwire',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 20,
    damage: '1d12+10 corte',
    save: { type: 'reflexos', cd: 25, effect: 'anula' },
    effect: 'Um pêndulo desce do teto causando 1d12+10 de corte.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'fosso-com-estacas',
    name: 'Fosso com Estacas',
    nd: 2,
    triggerType: 'pressao',
    detectSkill: 'investigacao',
    detectCd: 20,
    disarmCd: 20,
    damage: '6d6 impacto + 2d4+5 perfuração',
    save: { type: 'reflexos', cd: 20, effect: 'anula' },
    effect:
      'Queda de 9m: 6d6 de impacto mais estacas causam 2d4+5 de perfuração. Atletismo CD 20 para escalar de volta.',
    magica: false,
    bookPage: 317,
  },
  {
    id: 'runa-de-protecao',
    name: 'Runa de Proteção',
    nd: 2,
    triggerType: 'magica',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 25,
    damage: '6d6 (fogo / ácido / eletricidade / frio / luz / trevas)',
    save: { type: 'reflexos', cd: 20, effect: 'metade' },
    effect:
      '6d6 de fogo (ou ácido, eletricidade, frio, luz ou trevas) em criaturas a até 3m. Quem ativou não tem direito ao teste.',
    magica: true,
    bookPage: 317,
  },
  {
    id: 'simbolo-do-medo',
    name: 'Símbolo do Medo',
    nd: 2,
    triggerType: 'magica',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 25,
    damage: null,
    save: { type: 'vontade', cd: 20, effect: 'anula' },
    effect: 'Criaturas em alcance curto ficam abaladas até o fim da cena.',
    magica: true,
    bookPage: 317,
  },
  {
    id: 'estatua-executora',
    name: 'Estátua Executora',
    nd: 3,
    triggerType: 'proximidade',
    detectSkill: 'investigacao',
    detectCd: 20,
    disarmCd: 20,
    damage: '1d12+10 corte + 1d12+10 corte (dois golpes)',
    save: { type: 'reflexos', cd: 25, effect: 'parcial' },
    effect:
      'Dois golpes de 1d12+10 de corte. Dois testes de Reflexos CD 25 (cada teste evita um dos danos).',
    magica: false,
    bookPage: 318,
  },
  {
    id: 'gas-venenoso',
    name: 'Gás Venenoso',
    nd: 3,
    triggerType: 'pressao',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 25,
    damage: '1d12 PV de veneno por rodada durante 2d4 rodadas',
    save: { type: 'fortitude', cd: 20, effect: 'metade' },
    effect: 'Vítima perde 1d12 PV por veneno por rodada durante 2d4 rodadas.',
    magica: false,
    bookPage: 318,
  },
  {
    id: 'simbolo-do-sono',
    name: 'Símbolo do Sono',
    nd: 3,
    triggerType: 'magica',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 25,
    damage: null,
    save: { type: 'vontade', cd: 20, effect: 'anula' },
    effect:
      'Criaturas em alcance curto com 8 níveis ou menos caem inconscientes (como a magia Sono).',
    magica: true,
    bookPage: 318,
  },
  {
    id: 'parede-instavel',
    name: 'Parede Instável',
    nd: 4,
    triggerType: 'proximidade',
    detectSkill: 'investigacao',
    detectCd: 20,
    disarmCd: 20,
    damage: '8d6 impacto',
    save: { type: 'reflexos', cd: 25, effect: 'metade' },
    effect: '8d6 de impacto num quadrado de 3m de lado.',
    magica: false,
    bookPage: 318,
  },
  {
    id: 'simbolo-da-dor',
    name: 'Símbolo da Dor',
    nd: 4,
    triggerType: 'magica',
    detectSkill: 'investigacao',
    detectCd: 30,
    disarmCd: 30,
    damage: null,
    save: { type: 'fortitude', cd: 25, effect: 'anula' },
    effect:
      'Criaturas em alcance curto sofrem dor terrível: -5 em todos os testes até o fim da cena.',
    magica: true,
    bookPage: 318,
  },
  {
    id: 'bruma-da-insanidade',
    name: 'Bruma da Insanidade',
    nd: 5,
    triggerType: 'pressao',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 25,
    damage: null,
    save: { type: 'fortitude', cd: 20, effect: 'anula' },
    effect:
      'Criaturas em um cubo de 6m de lado ficam confusas até o fim da cena.',
    magica: false,
    bookPage: 318,
  },
  {
    id: 'simbolo-do-atordoamento',
    name: 'Símbolo do Atordoamento',
    nd: 5,
    triggerType: 'magica',
    detectSkill: 'investigacao',
    detectCd: 30,
    disarmCd: 30,
    damage: null,
    save: { type: 'fortitude', cd: 25, effect: 'anula' },
    effect:
      'Criaturas em alcance curto ficam atordoadas por 1d6 rodadas.',
    magica: true,
    bookPage: 318,
  },
  {
    id: 'desabamento-do-teto',
    name: 'Desabamento do Teto',
    nd: 6,
    triggerType: 'pressao',
    detectSkill: 'investigacao',
    detectCd: 25,
    disarmCd: 25,
    damage: '15d6 impacto',
    save: { type: 'reflexos', cd: 30, effect: 'metade' },
    effect:
      '15d6 de impacto em criaturas num quadrado de 6m de lado.',
    magica: false,
    bookPage: 318,
  },
  {
    id: 'simbolo-da-insanidade',
    name: 'Símbolo da Insanidade',
    nd: 6,
    triggerType: 'magica',
    detectSkill: 'investigacao',
    detectCd: 30,
    disarmCd: 30,
    damage: null,
    save: { type: 'vontade', cd: 25, effect: 'anula' },
    effect:
      'Criaturas em alcance curto ficam confusas permanentemente.',
    magica: true,
    bookPage: 318,
  },
  {
    id: 'abismo-da-morte',
    name: 'Abismo da Morte',
    nd: 8,
    triggerType: 'pressao',
    detectSkill: 'investigacao',
    detectCd: 30,
    disarmCd: 30,
    damage: '20d6 impacto + 2d8+10 perfuração',
    save: { type: 'reflexos', cd: 30, effect: 'anula' },
    effect:
      'Quadrado de 6m de lado se abre para queda de 30m sobre estacas: 20d6 de impacto + 2d8+10 de perfuração. Atletismo CD 25 para escalar de volta.',
    magica: false,
    bookPage: 318,
  },
  {
    id: 'simbolo-da-morte',
    name: 'Símbolo da Morte',
    nd: 8,
    triggerType: 'magica',
    detectSkill: 'investigacao',
    detectCd: 30,
    disarmCd: 30,
    damage: '10d6 trevas (em sucesso parcial)',
    save: { type: 'fortitude', cd: 30, effect: 'parcial' },
    effect:
      'Criaturas em alcance curto são reduzidas a -1 PV. Fortitude reduz para 10d6 de trevas em vez disso.',
    magica: true,
    bookPage: 318,
  },
])

const byId = new Map(TRAPS.map((t) => [t.id, t]))

export function trapById(id: string): Trap | undefined {
  return byId.get(id)
}

export function trapsByTrigger(trigger: TrapTriggerType): readonly Trap[] {
  return TRAPS.filter((t) => t.triggerType === trigger)
}

export function trapsByNd(nd: number): readonly Trap[] {
  return TRAPS.filter((t) => t.nd === nd)
}

export function magicalTraps(): readonly Trap[] {
  return TRAPS.filter((t) => t.magica)
}

/**
 * Resolve um teste de Sabotar (Ladinagem) contra a armadilha. PDF p120:
 * falha por 5+ aciona a armadilha. Retorna 'sucesso' | 'falha' |
 * 'falha-critica'.
 *
 * Não modela o requisito de treino — caller deve validar
 * `TRAP_DISARM_REQUIRES_TRAINED_LADINAGEM` antes de chamar.
 */
export function resolveDisarm(
  trap: Trap,
  ladinagemTotal: number,
): 'sucesso' | 'falha' | 'falha-critica' {
  if (ladinagemTotal >= trap.disarmCd) return 'sucesso'
  if (ladinagemTotal <= trap.disarmCd - TRAP_DISARM_CRITICAL_FAILURE_MARGIN) {
    return 'falha-critica'
  }
  return 'falha'
}
