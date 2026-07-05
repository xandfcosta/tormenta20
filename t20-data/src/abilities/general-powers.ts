import type { Modifier } from '../items/types'
import type { Prerequisite } from './types'

/**
 * Power "kinds" mirror the PDF's four general-power pools — Combate, Destino,
 * Magia, Tormenta — plus per-class pools (`barbaro`, `bardo`, etc.). When a
 * class grants a power slot at a given level, the slot's `kinds` list tells
 * the picker which pools the player may draw from.
 */
export type PowerKind =
  | 'combate'
  | 'destino'
  | 'magia'
  | 'tormenta'
  | 'arcanista'
  | 'barbaro'
  | 'bardo'
  | 'bucaneiro'
  | 'cacador'
  | 'cavaleiro'
  | 'clerigo'
  | 'druida'
  | 'guerreiro'
  | 'inventor'
  | 'ladino'
  | 'lutador'
  | 'nobre'
  | 'paladino'

/**
 * General powers (PDF Cap 4 — power pools shared across classes). Stored
 * separately from class-specific electives because multiple classes draw from
 * the same pool. Same shape as ClassPower minus the `className` field.
 */
export type GeneralPower = {
  id: string
  kind: PowerKind
  name: string
  description: string
  minLevel?: number
  prerequisites?: Prerequisite[]
  modifiers?: Modifier[]
}

/**
 * General Powers (PDF Cap 2 — "Poderes Gerais", book p124-131).
 *
 * Per PDF p33: players may substitute a class power slot for a general
 * power. Pools encoded here: Combate (p124-128), Destino (p129-131),
 * Magia (p131). Tormenta is encoded separately in `../tormenta.ts`.
 *
 * Class-specific lists live under `./classes/`. The general-power
 * catalog is kept separate so the same general power isn't duplicated
 * 14 times.
 *
 * Power kinds in this catalog map 1:1 to T20 categorias. Prereqs use
 * the typed `Prerequisite` union from `./types`; `note` is reserved
 * for free-form prereqs (e.g., "Lançar magias") not yet
 * machine-checked.
 */
export const GENERAL_POWERS_CATALOG: GeneralPower[] = [
  // ─── Combate ──────────────────────────────────────────────────────
  {
    id: 'acuidade-com-arma',
    kind: 'combate',
    name: 'Acuidade com Arma',
    description:
      'Com arma corpo a corpo leve ou de arremesso, usa Des em vez de For nos testes de ataque e nas rolagens de dano.',
    prerequisites: [{ kind: 'attribute', attr: 'dexterity', min: 1 }],
  },
  {
    id: 'ataque-poderoso',
    kind: 'combate',
    name: 'Ataque Poderoso',
    description: 'Em ataques corpo a corpo: -2 no teste de ataque para +5 no dano.',
    prerequisites: [{ kind: 'attribute', attr: 'strength', min: 1 }],
  },
  {
    id: 'combate-defensivo',
    kind: 'combate',
    name: 'Combate Defensivo',
    description: 'Ao agredir, -2 nos ataques para +5 na Defesa até o próximo turno.',
    prerequisites: [{ kind: 'attribute', attr: 'intelligence', min: 1 }],
  },
  {
    id: 'esquiva',
    kind: 'combate',
    name: 'Esquiva',
    description: '+2 na Defesa e em Reflexos.',
    prerequisites: [{ kind: 'attribute', attr: 'dexterity', min: 1 }],
  },
  {
    id: 'reflexos-de-combate',
    kind: 'combate',
    name: 'Reflexos de Combate',
    description: 'Ganha 1 ação de movimento extra no primeiro turno de cada combate.',
    prerequisites: [{ kind: 'attribute', attr: 'dexterity', min: 1 }],
  },
  {
    id: 'encouracado',
    kind: 'combate',
    name: 'Encouraçado',
    description:
      'Usando armadura pesada, +2 na Defesa (cumulativo +2 por poder com Encouraçado como prereq).',
    prerequisites: [
      { kind: 'note', description: 'Proficiência com armaduras pesadas' },
    ],
  },
  {
    id: 'estilo-de-arma-e-escudo',
    kind: 'combate',
    name: 'Estilo de Arma e Escudo',
    description: 'O bônus na Defesa concedido pelo escudo aumenta em +2.',
    prerequisites: [
      { kind: 'trained', expertise: 'Luta' },
      { kind: 'note', description: 'Proficiência com escudos' },
    ],
  },
  {
    id: 'estilo-de-duas-armas',
    kind: 'combate',
    name: 'Estilo de Duas Armas',
    description:
      'Empunhando duas armas (uma leve) e usando agredir: dois ataques com -2 em todos.',
    prerequisites: [
      { kind: 'attribute', attr: 'dexterity', min: 2 },
      { kind: 'trained', expertise: 'Luta' },
    ],
  },
  {
    id: 'estilo-de-duas-maos',
    kind: 'combate',
    name: 'Estilo de Duas Mãos',
    description:
      'Com arma corpo a corpo em duas mãos, +5 nas rolagens de dano (não funciona com armas leves).',
    prerequisites: [
      { kind: 'attribute', attr: 'strength', min: 2 },
      { kind: 'trained', expertise: 'Luta' },
    ],
  },
  {
    id: 'estilo-de-disparo',
    kind: 'combate',
    name: 'Estilo de Disparo',
    description: 'Empunhando arma de disparo, soma Des nas rolagens de dano.',
    prerequisites: [{ kind: 'trained', expertise: 'Pontaria' }],
  },
  {
    id: 'estilo-de-arremesso',
    kind: 'combate',
    name: 'Estilo de Arremesso',
    description: 'Saca armas de arremesso como ação livre; +2 nos danos.',
    prerequisites: [{ kind: 'trained', expertise: 'Pontaria' }],
  },
  {
    id: 'estilo-de-uma-arma',
    kind: 'combate',
    name: 'Estilo de Uma Arma',
    description:
      'Empunhando arma corpo a corpo em uma mão e nada na outra: +2 na Defesa e nos ataques (exceto desarmados).',
    prerequisites: [{ kind: 'trained', expertise: 'Luta' }],
  },
  {
    id: 'estilo-desarmado',
    kind: 'combate',
    name: 'Estilo Desarmado',
    description:
      'Ataques desarmados causam 1d6 e podem causar dano letal ou não letal sem penalidade.',
    prerequisites: [{ kind: 'trained', expertise: 'Luta' }],
  },
  {
    id: 'foco-em-arma',
    kind: 'combate',
    name: 'Foco em Arma',
    description: '+2 nos ataques com a arma escolhida. Pode ser pego várias vezes.',
    prerequisites: [
      { kind: 'note', description: 'Proficiência com a arma escolhida' },
    ],
  },
  {
    id: 'ginete',
    kind: 'combate',
    name: 'Ginete',
    description:
      'Sucesso automático em Cavalgar para não cair ao sofrer dano; sem penalidades ao atacar/lançar magias montado.',
    prerequisites: [{ kind: 'trained', expertise: 'Cavalgar' }],
  },
  {
    id: 'proficiencia',
    kind: 'combate',
    name: 'Proficiência',
    description:
      'Ganha proficiência em armas marciais, armas de fogo, armaduras pesadas ou escudos (à escolha).',
  },
  {
    id: 'saque-rapido',
    kind: 'combate',
    name: 'Saque Rápido',
    description:
      '+2 em Iniciativa; saca/guarda itens como ação livre; recarga diminui uma categoria.',
    prerequisites: [{ kind: 'trained', expertise: 'Iniciativa' }],
  },
  {
    id: 'vitalidade',
    kind: 'combate',
    name: 'Vitalidade',
    description: '+1 PV por nível e +2 em Fortitude.',
    prerequisites: [{ kind: 'attribute', attr: 'constitution', min: 1 }],
  },
  // Combate residual audit 2026-07-05 — 22 poderes p124-125,128-129 que
  // ainda não constavam no catálogo. Inseridos em ordem alfabética.
  {
    id: 'arma-secundaria-grande',
    kind: 'combate',
    name: 'Arma Secundária Grande',
    description:
      'Pode empunhar duas armas de uma mão com o poder Estilo de Duas Armas.',
    prerequisites: [{ kind: 'power', id: 'estilo-de-duas-armas' }],
  },
  {
    id: 'arremesso-multiplo',
    kind: 'combate',
    name: 'Arremesso Múltiplo',
    description:
      'Uma vez por rodada, ao atacar com arma de arremesso, gasta 1 PM para ataque adicional contra o mesmo alvo arremessando outra arma.',
    prerequisites: [
      { kind: 'attribute', attr: 'dexterity', min: 1 },
      { kind: 'power', id: 'estilo-de-arremesso' },
    ],
  },
  {
    id: 'arremesso-potente',
    kind: 'combate',
    name: 'Arremesso Potente',
    description:
      'Com arma de arremesso, usa For em vez de Des nos testes de ataque. Se possui Ataque Poderoso, pode usá-lo com armas de arremesso.',
    prerequisites: [
      { kind: 'attribute', attr: 'strength', min: 1 },
      { kind: 'power', id: 'estilo-de-arremesso' },
    ],
  },
  {
    id: 'ataque-com-escudo',
    kind: 'combate',
    name: 'Ataque com Escudo',
    description:
      'Uma vez por rodada, ao usar ação agredir com escudo, gasta 1 PM para ataque corpo a corpo extra com o escudo. Não perde o bônus do escudo na Defesa.',
    prerequisites: [{ kind: 'power', id: 'estilo-de-arma-e-escudo' }],
  },
  {
    id: 'ataque-pesado',
    kind: 'combate',
    name: 'Ataque Pesado',
    description:
      'Ao acertar ataque corpo a corpo com arma de duas mãos, gasta 1 PM para aplicar derrubar ou empurrar contra o alvo (usa resultado do ataque como teste de manobra).',
    prerequisites: [{ kind: 'power', id: 'estilo-de-duas-maos' }],
  },
  {
    id: 'ataque-preciso',
    kind: 'combate',
    name: 'Ataque Preciso',
    description:
      'Empunhando arma corpo a corpo em uma mão e nada na outra: +2 na margem de ameaça e +1 no multiplicador de crítico com ela.',
    prerequisites: [{ kind: 'power', id: 'estilo-de-uma-arma' }],
  },
  {
    id: 'bloqueio-com-escudo',
    kind: 'combate',
    name: 'Bloqueio com Escudo',
    description:
      'Ao sofrer dano, gasta 1 PM para receber redução de dano igual ao bônus de Defesa do escudo. Exige escudo equipado.',
    prerequisites: [{ kind: 'power', id: 'estilo-de-arma-e-escudo' }],
  },
  {
    id: 'carga-de-cavalaria',
    kind: 'combate',
    name: 'Carga de Cavalaria',
    description:
      'Ao fazer investida montado: +2d8 dano e pode continuar movendo depois do ataque (linha reta; máximo 2× deslocamento).',
    prerequisites: [{ kind: 'power', id: 'ginete' }],
  },
  {
    id: 'derrubar-aprimorado',
    kind: 'combate',
    name: 'Derrubar Aprimorado',
    description:
      '+2 em testes de ataque para derrubar. Ao derrubar com essa manobra, gasta 1 PM para ataque extra contra o alvo.',
    prerequisites: [{ kind: 'power', id: 'combate-defensivo' }],
  },
  {
    id: 'desarmar-aprimorado',
    kind: 'combate',
    name: 'Desarmar Aprimorado',
    description:
      '+2 em testes de ataque para desarmar. Ao desarmar, gasta 1 PM para arremessar a arma para longe (direção 1d8, distância 1d6×1,5m).',
    prerequisites: [{ kind: 'power', id: 'combate-defensivo' }],
  },
  {
    id: 'disparo-preciso',
    kind: 'combate',
    name: 'Disparo Preciso',
    description:
      'Ataques à distância contra oponentes envolvidos em corpo a corpo não sofrem penalidade de -5.',
    prerequisites: [
      {
        kind: 'anyPower',
        ids: ['estilo-de-disparo', 'estilo-de-arremesso'],
      },
    ],
  },
  {
    id: 'disparo-rapido',
    kind: 'combate',
    name: 'Disparo Rápido',
    description:
      'Empunhando arma de disparo que recarrega como ação livre + agredir como ação completa: um ataque adicional. Sofre -2 em todos os testes de ataque até o próximo turno.',
    prerequisites: [
      { kind: 'attribute', attr: 'dexterity', min: 1 },
      { kind: 'power', id: 'estilo-de-disparo' },
    ],
  },
  {
    id: 'empunhadura-poderosa',
    kind: 'combate',
    name: 'Empunhadura Poderosa',
    description:
      'Ao usar arma de uma categoria de tamanho maior que o próprio, penalidade cai de -5 para -2.',
    prerequisites: [{ kind: 'attribute', attr: 'strength', min: 3 }],
  },
  {
    id: 'estilo-de-arma-longa',
    kind: 'combate',
    name: 'Estilo de Arma Longa',
    description:
      '+2 em testes de ataque com armas alongadas e pode atacar alvos adjacentes com essas armas.',
    prerequisites: [
      { kind: 'attribute', attr: 'strength', min: 1 },
      { kind: 'trained', expertise: 'Luta' },
    ],
  },
  {
    id: 'fanatico',
    kind: 'combate',
    name: 'Fanático',
    description:
      'Deslocamento não é reduzido por usar armaduras pesadas.',
    minLevel: 12,
    prerequisites: [{ kind: 'power', id: 'encouracado' }],
  },
  {
    id: 'finta-aprimorada',
    kind: 'combate',
    name: 'Finta Aprimorada',
    description:
      '+2 em testes de Enganação para fintar e pode fintar como ação de movimento.',
    prerequisites: [{ kind: 'trained', expertise: 'Enganação' }],
  },
  {
    id: 'inexpugnavel',
    kind: 'combate',
    name: 'Inexpugnável',
    description:
      'Usando armadura pesada: +2 em todos os testes de resistência.',
    minLevel: 6,
    prerequisites: [{ kind: 'power', id: 'encouracado' }],
  },
  {
    id: 'mira-apurada',
    kind: 'combate',
    name: 'Mira Apurada',
    description:
      'Ao usar a ação mirar: +2 em testes de ataque e na margem de ameaça com ataques à distância até o fim do turno.',
    prerequisites: [
      { kind: 'attribute', attr: 'wisdom', min: 1 },
      { kind: 'power', id: 'disparo-preciso' },
    ],
  },
  {
    id: 'piqueiro',
    kind: 'combate',
    name: 'Piqueiro',
    description:
      'Uma vez por rodada, se empunhando arma alongada e um inimigo entrar voluntariamente no alcance: gasta 1 PM para ataque corpo a corpo. Se o oponente entrou em investida, ataque causa dois dados de dano extra.',
    prerequisites: [{ kind: 'power', id: 'estilo-de-arma-longa' }],
  },
  {
    id: 'presenca-aterradora',
    kind: 'combate',
    name: 'Presença Aterradora',
    description:
      'Ação padrão + 1 PM para assustar todas as criaturas escolhidas em alcance curto (regras de Intimidação).',
    prerequisites: [{ kind: 'trained', expertise: 'Intimidação' }],
  },
  {
    id: 'quebrar-aprimorado',
    kind: 'combate',
    name: 'Quebrar Aprimorado',
    description:
      '+2 em testes de ataque para quebrar. Ao reduzir PV de arma a 0 ou menos, gasta 1 PM para ataque extra contra o usuário (mesmos valores; dados rolam de novo).',
    prerequisites: [{ kind: 'power', id: 'ataque-poderoso' }],
  },
  {
    id: 'trespassar',
    kind: 'combate',
    name: 'Trespassar',
    description:
      'Ao reduzir PV do alvo a 0 ou menos em ataque corpo a corpo, gasta 1 PM para ataque adicional contra outra criatura no alcance.',
    prerequisites: [{ kind: 'power', id: 'ataque-poderoso' }],
  },

  // ─── Destino ──────────────────────────────────────────────────────
  {
    id: 'acrobatico',
    kind: 'destino',
    name: 'Acrobático',
    description:
      'Usa Des em vez de For em Atletismo; terreno difícil não reduz deslocamento nem impede investidas.',
    prerequisites: [{ kind: 'attribute', attr: 'dexterity', min: 2 }],
  },
  {
    id: 'aparencia-inofensiva',
    kind: 'destino',
    name: 'Aparência Inofensiva',
    description:
      'Primeira criatura inteligente a atacá-lo faz Vontade (CD Car); na falha, perde a ação. 1×/cena.',
    prerequisites: [{ kind: 'attribute', attr: 'charisma', min: 1 }],
  },
  {
    id: 'atletico',
    kind: 'destino',
    name: 'Atlético',
    description: '+2 em Atletismo e +3m no deslocamento.',
    prerequisites: [{ kind: 'attribute', attr: 'strength', min: 2 }],
  },
  {
    id: 'atraente',
    kind: 'destino',
    name: 'Atraente',
    description:
      '+2 em perícias baseadas em Carisma contra criaturas que possam se sentir atraídas.',
    prerequisites: [{ kind: 'attribute', attr: 'charisma', min: 1 }],
  },
  {
    id: 'comandar',
    kind: 'destino',
    name: 'Comandar',
    description:
      'Ação de movimento + 1 PM: grita ordens para aliados em alcance médio; +1 em perícias até o fim da cena.',
    prerequisites: [{ kind: 'attribute', attr: 'charisma', min: 1 }],
  },
  {
    id: 'foco-em-pericia',
    kind: 'destino',
    name: 'Foco em Perícia',
    description:
      'Em teste da perícia escolhida, gasta 1 PM para rolar 2d20 e usar o melhor. Pode ser pego várias vezes (não Luta/Pontaria).',
    prerequisites: [
      { kind: 'note', description: 'Treinado na perícia escolhida' },
    ],
  },
  {
    id: 'lobo-solitario',
    kind: 'destino',
    name: 'Lobo Solitário',
    description:
      '+1 em perícia e Defesa quando sem aliados em alcance curto; sem penalidade ao usar Cura em si mesmo.',
  },
  {
    id: 'medicina',
    kind: 'destino',
    name: 'Medicina',
    description:
      'Ação completa + teste de Cura (CD 15): paciente recupera 1d6 PV +1d6 por 5 acima da CD. 1×/dia por criatura.',
    prerequisites: [
      { kind: 'attribute', attr: 'wisdom', min: 1 },
      { kind: 'trained', expertise: 'Cura' },
    ],
  },
  {
    id: 'sentidos-agucados',
    kind: 'destino',
    name: 'Sentidos Aguçados',
    description:
      '+2 em Percepção; não fica desprevenido contra invisíveis; reroll de chance de falha por camuflagem em erros.',
    prerequisites: [
      { kind: 'attribute', attr: 'wisdom', min: 1 },
      { kind: 'trained', expertise: 'Percepção' },
    ],
  },
  {
    id: 'sortudo',
    kind: 'destino',
    name: 'Sortudo',
    description: 'Gasta 3 PM para rolar um teste novamente.',
  },
  {
    id: 'surto-heroico',
    kind: 'destino',
    name: 'Surto Heroico',
    description: '1×/rodada: 5 PM para ação padrão OU de movimento adicional.',
  },
  {
    id: 'torcida',
    kind: 'destino',
    name: 'Torcida',
    description: '+2 em perícia e Defesa quando tem torcida a seu favor.',
    prerequisites: [{ kind: 'attribute', attr: 'charisma', min: 1 }],
  },
  {
    id: 'treinamento-em-pericia',
    kind: 'destino',
    name: 'Treinamento em Perícia',
    description:
      'Torna-se treinado em uma perícia à escolha. Pode ser pego várias vezes.',
  },
  {
    id: 'vontade-de-ferro',
    kind: 'destino',
    name: 'Vontade de Ferro',
    description: '+1 PM a cada dois níveis e +2 em Vontade.',
    prerequisites: [{ kind: 'attribute', attr: 'wisdom', min: 1 }],
  },

  // ─── Magia ────────────────────────────────────────────────────────
  {
    id: 'celebrar-ritual',
    kind: 'magia',
    name: 'Celebrar Ritual',
    description:
      'Lança magias como rituais: dobra o limite de PM, execução 1h, custa T$ 10 por PM gasto.',
    minLevel: 8,
    prerequisites: [
      { kind: 'note', description: 'Lançar magias; treinado em Misticismo ou Religião' },
    ],
  },
  {
    id: 'escrever-pergaminho',
    kind: 'magia',
    name: 'Escrever Pergaminho',
    description: 'Usa Ofício (escriba) para fabricar pergaminhos.',
    prerequisites: [
      { kind: 'note', description: 'Lançar magias; treinado em Ofício (escriba)' },
    ],
  },
  {
    id: 'foco-em-magia',
    kind: 'magia',
    name: 'Foco em Magia',
    description:
      'Custo da magia escolhida diminui em -1 PM (cumulativo). Pode ser pego várias vezes.',
    prerequisites: [{ kind: 'note', description: 'Lançar magias' }],
  },
  {
    id: 'magia-acelerada',
    kind: 'magia',
    name: 'Magia Acelerada',
    description: 'Aprimoramento: muda execução de uma magia para livre; +4 PM.',
    prerequisites: [
      { kind: 'note', description: 'Capacidade de lançar magias de 2º círculo' },
    ],
  },
  {
    id: 'magia-ampliada',
    kind: 'magia',
    name: 'Magia Ampliada',
    description:
      'Aprimoramento: aumenta o alcance da magia um passo OU dobra a área; +2 PM.',
    prerequisites: [{ kind: 'note', description: 'Lançar magias' }],
  },
  {
    id: 'magia-discreta',
    kind: 'magia',
    name: 'Magia Discreta',
    description:
      'Aprimoramento: lança a magia sem componentes verbais/gestuais; +2 PM.',
    prerequisites: [{ kind: 'note', description: 'Lançar magias' }],
  },
  {
    id: 'magia-ilimitada',
    kind: 'magia',
    name: 'Magia Ilimitada',
    description:
      'Soma seu atributo-chave ao limite de PM que pode gastar em uma magia (cap +9 PM).',
    prerequisites: [{ kind: 'note', description: 'Lançar magias' }],
  },
  {
    id: 'preparar-pocao',
    kind: 'magia',
    name: 'Preparar Poção',
    description:
      'Usa Ofício (alquimista) para fabricar poções de magias de 1º/2º círculo.',
    prerequisites: [
      { kind: 'note', description: 'Lançar magias; treinado em Ofício (alquimista)' },
    ],
  },
]

const generalPowersById = new Map<string, GeneralPower>(
  GENERAL_POWERS_CATALOG.map((p) => [p.id, p]),
)

export function getGeneralPower(id: string): GeneralPower | undefined {
  return generalPowersById.get(id)
}

export function generalPowersByKinds(
  allowedKinds: ReadonlyArray<PowerKind>,
): GeneralPower[] {
  const set = new Set(allowedKinds)
  return GENERAL_POWERS_CATALOG.filter((p) => set.has(p.kind))
}
