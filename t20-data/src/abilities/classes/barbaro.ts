import { attr, autoPower, electivePower, power, trained } from './_helpers'
import type { Modifier } from '../../items/types'
import type { ClassPower } from '../types'

const C = 'Bárbaro'

/**
 * Fúria grants morale bonus +N to attack, damage, Fortitude, Vontade — only
 * while the `furia` flag is on. Engine emits these as conditional rows the
 * player toggles in the conditionals panel. Non-stacking morale rule keeps
 * only the highest tier (so L6+ Fúria +3 supersedes L1 Fúria +2).
 */
function furiaMods(bonus: number): Modifier[] {
  const cond = { c: 'flagOn' as const, flag: 'furia', label: 'Em Fúria' }
  return [
    { target: { k: 'attack', scope: 'all' }, amount: bonus, bonusType: 'morale', condition: cond },
    { target: { k: 'damage', scope: 'all' }, amount: bonus, bonusType: 'morale', condition: cond },
    { target: { k: 'expertise', name: 'Fortitude' }, amount: bonus, bonusType: 'morale', condition: cond },
    { target: { k: 'expertise', name: 'Vontade' }, amount: bonus, bonusType: 'morale', condition: cond },
  ]
}

/**
 * PDF Cap 1 (Bárbaro, p40-42, Tabela 1-6). Auto features by level + 19 power
 * slots (L2-L20). Each slot: Poder de Bárbaro OR poder geral (Cap 2).
 *
 * Auto track:
 *   L1  — Fúria +2
 *   L3  — Instinto Selvagem +1
 *   L5  — Redução de Dano 2
 *   L6  — Fúria +3
 *   L8  — Redução de Dano 4
 *   L9  — Instinto Selvagem +2
 *   L11 — Fúria +4, Redução de Dano 6
 *   L14 — Redução de Dano 8
 *   L15 — Instinto Selvagem +3
 *   L16 — Fúria +5
 *   L17 — Redução de Dano 10
 *   L20 — Fúria Titânica
 */
export const BARBARO_POWERS: ClassPower[] = [
  autoPower(C, 1, 'Fúria',
    'Gasta 2 PM para invocar fúria. +2 ataque e dano corpo-a-corpo, +2 Fortitude/Vontade, mas não pode fazer ações que exijam calma (Furtividade, lançar magias). Fúria termina se ao fim da rodada não tiver atacado nem sido alvo de efeito hostil. A cada 5 níveis, pode gastar +1 PM para aumentar bônus em +1.',
    furiaMods(2),
  ),
  autoPower(C, 3, 'Instinto Selvagem +1',
    '+1 em rolagens de dano, Percepção e Reflexos. A cada 6 níveis aumenta em +1 (+2 em L9, +3 em L15).',
  ),
  autoPower(C, 5, 'Redução de Dano',
    'Recebe RD 2 contra dano físico. Sobe +2 a cada 3 níveis (RD 4 em L8, RD 6 em L11, RD 8 em L14, RD 10 em L17).',
  ),
  autoPower(C, 6, 'Fúria +3',
    'Bônus base de Fúria sobe para +3 (escala da Fúria).',
    furiaMods(3),
  ),
  autoPower(C, 8, 'Redução de Dano 4',
    'Redução de Dano aumenta para 4.',
  ),
  autoPower(C, 9, 'Instinto Selvagem +2',
    'Bônus de Instinto Selvagem sobe para +2.',
  ),
  autoPower(C, 11, 'Fúria +4',
    'Bônus base de Fúria sobe para +4.',
    furiaMods(4),
  ),
  autoPower(C, 11, 'Redução de Dano 6',
    'Redução de Dano aumenta para 6.',
  ),
  autoPower(C, 14, 'Redução de Dano 8',
    'Redução de Dano aumenta para 8.',
  ),
  autoPower(C, 15, 'Instinto Selvagem +3',
    'Bônus de Instinto Selvagem sobe para +3.',
  ),
  autoPower(C, 16, 'Fúria +5',
    'Bônus base de Fúria sobe para +5.',
    furiaMods(5),
  ),
  autoPower(C, 17, 'Redução de Dano 10',
    'Redução de Dano aumenta para 10.',
  ),
  autoPower(C, 20, 'Fúria Titânica',
    'Capstone: bônus de Fúria nos testes de ataque e rolagens de dano é dobrado. Exemplo: 5 PM gastos = bônus de +10 em vez de +5.',
  ),

  // Poderes de Bárbaro — eletivos (p41-42)
  electivePower(C, 'Alma de Bronze',
    'Quando entra em fúria, recebe PV temporários = nível + Força.'),
  electivePower(C, 'Aumento de Atributo',
    '+1 em um atributo. Apenas uma vez por patamar para um mesmo atributo. Pode ser escolhido várias vezes.'),
  electivePower(C, 'Brado Assustador',
    'Ação de movimento + 1 PM: solta berro. Inimigos em alcance curto ficam vulneráveis até fim da cena (Vontade CD Intimidação anula). Medo.',
    { prerequisites: [trained('Intimidação')] }),
  electivePower(C, 'Crítico Brutal',
    'Multiplicador de crítico com armas corpo-a-corpo aumenta em +1. Ex: x3 vira x4.',
    { minLevel: 6 }),
  electivePower(C, 'Destruidor',
    'Em dano corpo-a-corpo com arma de duas mãos, pode rolar novamente resultados 1 ou 2 da rolagem de dano.',
    { prerequisites: [attr('strength', 1)] }),
  electivePower(C, 'Espírito Inquebrável',
    'Em fúria não fica inconsciente por 0 PV ou menos (ainda morre em valor negativo igual a metade do PV máx).',
    { prerequisites: [power('class.barbaro.alma-de-bronze')] }),
  electivePower(C, 'Esquiva Sobrenatural',
    'Instintos apurados — nunca fica surpreendido.'),
  electivePower(C, 'Força Indomável',
    'Em teste de Força ou Atletismo, gasta 1 PM para somar nível. Pode usar após rolar dado, antes de o mestre dizer se passou.'),
  electivePower(C, 'Frenesi',
    'Uma vez por rodada, se em fúria e usar ação agredir corpo-a-corpo ou arma de arremesso, gasta 2 PM para fazer ataque adicional.'),
  electivePower(C, 'Fúria da Savana',
    'Deslocamento aumenta em +3m. Em fúria, bônus aplica também a armas de arremesso.'),
  electivePower(C, 'Fúria Raivosa',
    'Se Fúria fosse terminar por não ter atacado nem sido alvo de efeito hostil, paga 1 PM para continuar em fúria.'),
  electivePower(C, 'Golpe Poderoso',
    'Ao acertar ataque corpo-a-corpo ou arma de arremesso, gasta 1 PM para causar dado de dano extra do mesmo tipo.'),
  electivePower(C, 'Ímpeto',
    'Gasta 1 PM para aumentar deslocamento em +6m por uma rodada.'),
  electivePower(C, 'Investida Imprudente',
    'Em investida, pode aumentar penalidade de Defesa para -5 para receber +1d12 na rolagem de dano deste ataque.'),
  electivePower(C, 'Pele de Aço',
    'Bônus de Pele de Ferro aumenta para +8.',
    { prerequisites: [power('class.barbaro.pele-de-ferro')], minLevel: 8 }),
  electivePower(C, 'Pele de Ferro',
    '+4 Defesa se não estiver usando armadura pesada.'),
  electivePower(C, 'Sangue dos Inimigos',
    'Em fúria, quando confirma crítico ou reduz inimigo a 0 PV, recebe +1 cumulativo em ataque e dano (até fim da cena, limitado por nível).'),
  electivePower(C, 'Superstição',
    'Odeia magia: recebe resistência a magia +5.'),
  electivePower(C, 'Totem Espiritual',
    'Soma Sabedoria no PM total. Escolha um animal totêmico — aprende e pode lançar magia definida pelo animal (atributo-chave Sab), mesmo em fúria.',
    { prerequisites: [attr('wisdom', 1)], minLevel: 4 }),
  electivePower(C, 'Vigor Primal',
    'Ação de movimento + PM (limitado por Con). Para cada PM, recupera 1d12 PV.'),
]
