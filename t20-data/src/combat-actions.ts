import type { ActionKind } from './combat'

/**
 * Catálogo de Ações de Combate — PDF Cap 5 (Combate), p233-236.
 *
 * Complements `combat.ts` (resolution + iniciativa + action-economy
 * validator) and `maneuvers.ts` (the 5 core manobras). This catalog
 * enumerates the *named* actions a character can declare on their
 * turn, mapped to the existing `ActionKind` union.
 *
 * T20-specific notes (vs D&D 3.5):
 *
 *  - There is **no generic "Ataque de Oportunidade"** rule. The only
 *    AoO-like triggers come from class features (e.g. Marca de Combate
 *    do guerreiro) and a handful of talents — they are encoded with
 *    the granting feature, not as a baseline action.
 *  - "Lutar Defensivamente" / "Defesa Total" / "Esquiva" são poderes /
 *    perícias / atributos passivos, NÃO ações de combate nomeadas.
 *  - "Beber Poção" / "Estabilizar" são instâncias de `usar-habilidade-
 *    ou-item` / perícia Cura, não entradas separadas.
 */
export type CombatAction = {
  id: string
  name: string
  kind: ActionKind
  effect: string
  bookPage: number
}

export const COMBAT_ACTIONS: readonly CombatAction[] = Object.freeze([
  // ─── Ações Padrão (p233-234) ────────────────────────────────────
  {
    id: 'agredir',
    name: 'Agredir',
    kind: 'padrao',
    effect:
      'Você faz um ataque com uma arma corpo a corpo ou à distância. Com arma corpo a corpo, pode atacar qualquer inimigo dentro de seu alcance natural (1,5m para criaturas Pequenas e Médias, ou adjacente no mapa). Com arma à distância, pode atacar qualquer inimigo que consiga ver e esteja no alcance da arma (até o dobro do alcance impõe -5). Pode substituir um ataque corpo a corpo por uma manobra. Atirar em combate corpo a corpo impõe -5 no teste de ataque.',
    bookPage: 233,
  },
  {
    id: 'atropelar',
    name: 'Atropelar',
    kind: 'padrao',
    effect:
      'Você avança pelo espaço ocupado por uma criatura durante um movimento (exceção à regra normal). A criatura pode dar passagem (sem teste) ou resistir; nesse caso, faça um teste de manobra oposto. Se vencer, deixa o alvo caído e continua; se perder, o alvo segue de pé e detém seu avanço. Durante uma investida, atropelar vira ação livre.',
    bookPage: 233,
  },
  {
    id: 'fintar',
    name: 'Fintar',
    kind: 'padrao',
    effect:
      'Faça um teste de Enganação oposto ao teste de Reflexos de uma criatura em alcance curto. Se passar, ela fica desprevenida contra seu próximo ataque, mas apenas até o fim de seu próximo turno.',
    bookPage: 234,
  },
  {
    id: 'lancar-uma-magia',
    name: 'Lançar uma Magia',
    kind: 'padrao',
    effect:
      'A maioria das magias exige uma ação padrão para ser executada. Magias com execução maior usam a entrada Lançar uma Magia (execução maior) entre as ações completas.',
    bookPage: 234,
  },
  {
    id: 'preparar',
    name: 'Preparar',
    kind: 'padrao',
    effect:
      'Você prepara uma ação (padrão, de movimento ou livre) para realizar mais tarde, após seu turno mas antes de seu turno na próxima rodada. Declare a ação e a circunstância gatilho. Pode executá-la como reação ao gatilho a qualquer momento antes do seu próximo turno; se chegar seu próximo turno sem disparar, perde a ação preparada (mas pode prepará-la de novo). Sua Iniciativa pelo resto do combate fica imediatamente acima daquela em que disparou a ação.',
    bookPage: 234,
  },
  {
    id: 'usar-habilidade-ou-item',
    name: 'Usar Habilidade ou Item',
    kind: 'padrao',
    effect:
      'Algumas habilidades e itens (como poções) exigem uma ação padrão para serem usados.',
    bookPage: 234,
  },

  // ─── Ações de Movimento (p234) ──────────────────────────────────
  {
    id: 'levantar-se',
    name: 'Levantar-se',
    kind: 'movimento',
    effect:
      'Levantar-se do chão (ou de uma cama, cadeira...) exige uma ação de movimento.',
    bookPage: 234,
  },
  {
    id: 'manipular-item',
    name: 'Manipular Item',
    kind: 'movimento',
    effect:
      'Pegar um objeto em uma mochila, abrir ou fechar uma porta, atirar uma corda para alguém — manipulações comuns de itens exigem uma ação de movimento.',
    bookPage: 234,
  },
  {
    id: 'mirar',
    name: 'Mirar',
    kind: 'movimento',
    effect:
      'Você mira em um alvo que possa ver, dentro do alcance da arma. Anula a penalidade de -5 em testes de Pontaria contra aquele alvo neste turno caso ele esteja engajado em combate corpo a corpo.',
    bookPage: 234,
  },
  {
    id: 'movimentar-se',
    name: 'Movimentar-se',
    kind: 'movimento',
    effect:
      'Você percorre uma distância igual a seu deslocamento (tipicamente 9m para raças de tamanho Médio). Outros modos — nadar, escalar, cavalgar — também usam esta ação.',
    bookPage: 234,
  },
  {
    id: 'sacar-ou-guardar-item',
    name: 'Sacar ou Guardar Item',
    kind: 'movimento',
    effect:
      'Sacar ou guardar um item exige uma ação de movimento. Se puder usar mais de uma arma (Ambidestria, etc.), pode sacar todas elas.',
    bookPage: 234,
  },

  // ─── Ações Completas (p235) ─────────────────────────────────────
  {
    id: 'corrida',
    name: 'Corrida',
    kind: 'completa',
    effect:
      'Você corre mais rapidamente que seu deslocamento normal (perícia Atletismo).',
    bookPage: 235,
  },
  {
    id: 'golpe-de-misericordia',
    name: 'Golpe de Misericórdia',
    kind: 'completa',
    effect:
      'Você desfere um golpe letal em um oponente adjacente e indefeso. É acerto crítico automático. Além do dano, a vítima tem chance de morrer instantaneamente: 25% (1 em 1d4) para personagens e NPCs importantes; 75% (1-3 em 1d4) para NPCs secundários.',
    bookPage: 235,
  },
  {
    id: 'investida',
    name: 'Investida',
    kind: 'completa',
    effect:
      'Você avança até o dobro de seu deslocamento (no mínimo 3m) em linha reta e, no fim do movimento, faz um ataque corpo a corpo. +2 no teste de ataque, -2 na Defesa até seu próximo turno (guarda aberta). Não é possível em terreno difícil. Durante a investida, atropelar vira ação livre (não pode atropelar e atacar o mesmo alvo).',
    bookPage: 235,
  },
  {
    id: 'lancar-uma-magia-completa',
    name: 'Lançar uma Magia (execução maior)',
    kind: 'completa',
    effect:
      'Ao lançar magias com execução maior do que uma ação completa, você gasta uma ação completa a cada rodada até concluir a execução.',
    bookPage: 235,
  },

  // ─── Ações Livres (p235-236) ────────────────────────────────────
  {
    id: 'atrasar',
    name: 'Atrasar',
    kind: 'livre',
    effect:
      'Atrasando sua ação, você age mais tarde na ordem de Iniciativa, em relação à Iniciativa que rolou — equivale a reduzir sua Iniciativa voluntariamente pelo resto do combate. Pode declarar a nova Iniciativa ou esperar um momento e então agir. Limite: pode atrasar até -10 menos seu valor de Iniciativa; nesse ponto, precisa agir ou abrir mão da ação na rodada.',
    bookPage: 235,
  },
  {
    id: 'falar',
    name: 'Falar',
    kind: 'livre',
    effect:
      'Em geral, falar é ação livre. Lançar magias com componente verbal e habilidades de classe que dependem da voz não são ações livres. O mestre pode limitar o que se fala por rodada (vinte palavras é o limite padrão).',
    bookPage: 236,
  },
  {
    id: 'jogar-se-no-chao',
    name: 'Jogar-se no Chão',
    kind: 'livre',
    effect:
      'Jogar-se no chão é ação livre. Recebe os benefícios e penalidades normais da condição caído, mas em geral não sofre dano por se jogar.',
    bookPage: 236,
  },
  {
    id: 'largar-um-item',
    name: 'Largar um Item',
    kind: 'livre',
    effect:
      'Deixar cair um item que esteja segurando é ação livre. Deixar cair (ou jogar) um item com a intenção de acertar algo é ação padrão. Deixar cair (ou jogar) um item para que outra pessoa agarre é ação de movimento.',
    bookPage: 236,
  },

  // ─── Ações de Reação (p233 — Tipos de Ações) ────────────────────
  {
    id: 'reacao-preparada',
    name: 'Reação',
    kind: 'reacao',
    effect:
      'Uma reação acontece em resposta a outra coisa. Como ações livres, tomam tão pouco tempo que se pode realizar qualquer quantidade delas. Diferença: ação livre é escolha consciente no seu turno; reação é resposta automática, pode ocorrer fora do seu turno. Pode reagir mesmo se não puder realizar ações (ex.: atordoado). Exemplos: teste de Percepção para detectar um troll, teste de Reflexos para escapar de uma explosão.',
    bookPage: 233,
  },
])

const byId = new Map(COMBAT_ACTIONS.map((a) => [a.id, a]))

export function combatActionById(id: string): CombatAction | undefined {
  return byId.get(id)
}

export function combatActionsByKind(
  kind: ActionKind,
): readonly CombatAction[] {
  return COMBAT_ACTIONS.filter((a) => a.kind === kind)
}
