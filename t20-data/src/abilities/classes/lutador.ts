import { attributeBoostPower, attr, autoPower, electivePower, power, trained } from './_helpers'
import type { Modifier } from '../../items/types'
import type { ClassPower } from '../types'

const C = 'Lutador'

/**
 * Casca Grossa (p77): "soma sua Constituição na Defesa ... apenas se não
 * estiver usando armadura pesada. Além disso, no 7º nível, e a cada quatro
 * níveis, você recebe +1 na Defesa." Só a parte FIXA (+1..+4 em L7/11/15/19)
 * é modelada — Defesa escalada por atributo não tem ModifierTarget (scale só
 * vale para maxPv/maxPm). Mesmo bonusType 'training' em todos os tiers para
 * o resolveStack manter só o maior (+2 do L11 supera o +1 do L7).
 */
function cascaGrossaMods(bonus: number): Modifier[] {
  return [
    { target: { k: 'defense' }, amount: bonus, bonusType: 'training' },
  ]
}

/**
 * PDF Cap 1 (Lutador, p75-77, Tabela 1-16). Auto: Briga (1d6) + Golpe
 * Relâmpago L1. Briga scaling (1d8 L5, 1d10 L9, 2d6 L13, 2d8 L17, 2d10 L20).
 * Casca Grossa Con+0/1/2/3/4 em L3/7/11/15/19. Golpe Cruel L5. Golpe
 * Violento L9. Dono da Rua L20. 19 power slots L2-L20.
 */
export const LUTADOR_POWERS: ClassPower[] = [
  autoPower(C, 1, 'Briga (1d6)',
    'Ataques desarmados causam 1d6 dano e podem ser letais ou não letais (sem penalidades). A cada 4 níveis, dano desarmado aumenta. Médias 1d6, Pequenas/Minúsculas diminuem 1 passo, Grandes/Enormes aumentam, Colossais 2 passos.',
  ),
  autoPower(C, 1, 'Golpe Relâmpago',
    'Quando usa ação agredir para fazer ataque desarmado, gasta 1 PM para fazer ataque desarmado adicional.',
  ),
  autoPower(C, 3, 'Casca Grossa (Con)',
    'Soma Constituição na Defesa, limitado pelo seu nível e apenas se não estiver usando armadura pesada. A cada 4 níveis (L7/11/15/19) recebe +1 na Defesa.',
  ),
  autoPower(C, 5, 'Briga (1d8)',
    'Dano desarmado sobe para 1d8.',
  ),
  autoPower(C, 5, 'Golpe Cruel',
    'Margem de ameaça com ataques desarmados aumenta em +1.',
    // p77: "Sua margem de ameaça com ataques desarmados aumenta em +1" —
    // critRange não tem escopo por arma; context ⇒ toggle opt-in.
    [{ target: { k: 'critRange' }, amount: 1, bonusType: 'untyped', condition: { c: 'context', note: 'com ataques desarmados' } }],
  ),
  autoPower(C, 7, 'Casca Grossa (Con+1)',
    '+1 na Defesa (cumulativo com Casca Grossa).',
    cascaGrossaMods(1),
  ),
  autoPower(C, 9, 'Briga (1d10)',
    'Dano desarmado sobe para 1d10.',
  ),
  autoPower(C, 9, 'Golpe Violento',
    'Multiplicador de crítico com ataques desarmados aumenta em +1.',
    // p77: "Seu multiplicador de crítico com ataques desarmados aumenta em +1".
    [{ target: { k: 'critMult' }, amount: 1, bonusType: 'untyped', condition: { c: 'context', note: 'com ataques desarmados' } }],
  ),
  autoPower(C, 11, 'Casca Grossa (Con+2)',
    '+2 na Defesa.',
    cascaGrossaMods(2),
  ),
  autoPower(C, 13, 'Briga (2d6)',
    'Dano desarmado sobe para 2d6.',
  ),
  autoPower(C, 15, 'Casca Grossa (Con+3)',
    '+3 na Defesa.',
    cascaGrossaMods(3),
  ),
  autoPower(C, 17, 'Briga (2d8)',
    'Dano desarmado sobe para 2d8.',
  ),
  autoPower(C, 19, 'Casca Grossa (Con+4)',
    '+4 na Defesa.',
    cascaGrossaMods(4),
  ),
  autoPower(C, 20, 'Dono da Rua (2d10)',
    'Capstone: dano desarmado sobe para 2d10. Quando usa ação agredir para ataque desarmado, faz dois ataques (podendo usar Golpe Relâmpago para terceiro).',
  ),

  // Poderes de Lutador (p76-77)
  electivePower(C, 'Arma Improvisada',
    'Pode atacar com armas improvisadas como ataques desarmados, mas dano aumenta um passo. Ação de movimento + teste Percepção CD 20: encontra arma improvisada. Frágeis (se erra + d20 ímpar, arma quebra).'),
  electivePower(C, 'Até Acertar',
    'Quando erra ataque desarmado, recebe +2 cumulativo em ataque e dano desarmado contra mesmo oponente até acertar ou fim cena.'),
  attributeBoostPower(C),
  electivePower(C, 'Braços Calejados',
    'Se não estiver usando armadura, soma Força na Defesa, limitado pelo nível.'),
  electivePower(C, 'Cabeçada',
    'Em ataque desarmado, gasta 2 PM. Se acertar, oponente fica desprevenido contra este ataque. 1x/cena contra um mesmo alvo.'),
  electivePower(C, 'Chave',
    'Se está agarrando criatura e faz teste de manobra para causar dano, dano desarmado aumenta um passo.',
    { prerequisites: [attr('intelligence', 1), power('class.lutador.lutador-de-chao')], minLevel: 4 }),
  electivePower(C, 'Confiança dos Ringues',
    'Quando inimigo erra ataque corpo-a-corpo, recebe 2 PM temp (cumulativos). Máx = nível por cena. Desaparecem fim cena.',
    { minLevel: 8 }),
  electivePower(C, 'Convencido',
    'Desdém por artes mais sofisticadas. Resistência a medo e mental +5.'),
  electivePower(C, 'Golpe Baixo',
    'Em ataque desarmado, gasta 2 PM. Se acertar, oponente Fort CD For ou atordoado por uma rodada (1x/cena).'),
  electivePower(C, 'Golpe Imprudente',
    'Quando usa Golpe Relâmpago, ataca de forma impulsiva. Ataques desarmados recebem dado de dano extra do mesmo tipo, mas sofre -5 Defesa.'),
  electivePower(C, 'Imobilização',
    'Se está agarrando criatura, gasta ação completa para imobilizá-la. Teste manobra contra ela. Sucesso = indefesa, não pode realizar ações exceto soltar.',
    { prerequisites: [power('class.lutador.chave')], minLevel: 8 }),
  electivePower(C, 'Língua dos Becos',
    'Em teste de perícia baseada em Carisma, paga 1 PM para usar Força no lugar deste atributo.',
    { prerequisites: [attr('strength', 1), trained('Intimidação')] }),
  electivePower(C, 'Lutador de Chão',
    '+2 em testes de ataque para agarrar e derrubar. Quando agarra criatura, gasta 1 PM para manobra derrubar como ação livre.'),
  electivePower(C, 'Nome na Arena',
    'Reputação no circuito de lutas. Uma vez por cena, gasta ação completa + teste Luta CD 10. Se passar, +2 todos testes de perícias baseadas em Carisma até fim cena. Escala com sucesso.',
    { minLevel: 11 }),
  electivePower(C, 'Punhos de Adamante',
    'Ataques desarmados ignoram 10 pontos de redução de dano do alvo.',
    { minLevel: 8 }),
  electivePower(C, 'Rasteira',
    'Em ataque desarmado contra criatura até uma categoria de tamanho maior que sua, paga 2 PM. Se acertar, criatura fica caída.'),
  electivePower(C, 'Sarado',
    'Soma Força no PV total e em Fortitude. Pode usar Força em vez de Carisma em Diplomacia com pessoas atraídas por físicos definidos.',
    {
      prerequisites: [attr('strength', 3)],
      // Soma Força no PV total (Fortitude +Força ainda não modelado — save target).
      modifiers: [
        { target: { k: 'maxPv' }, amount: 1, bonusType: 'untyped', scale: { per: 'attribute', attribute: 'strength' } },
      ],
    }),
  electivePower(C, 'Sequência Destruidora',
    'Início do turno: gasta 2 PM para dizer número (≥2). Se fizer e acertar quantidade de ataques igual, último recebe +4 cumulativo na rolagem de dano por ataque feito.',
    { minLevel: 8 }),
  electivePower(C, 'Trincado',
    'Esculpido à exaustão. Soma Constituição nas rolagens de dano desarmado.',
    { prerequisites: [attr('constitution', 3), power('class.lutador.sarado')], minLevel: 10 }),
  electivePower(C, 'Trocação',
    'Quando começa a bater, não para. Ao acertar ataque desarmado, pode fazer outro ataque contra mesmo alvo pagando PM = ataques já realizados.',
    { minLevel: 6 }),
  electivePower(C, 'Trocação Tumultuosa',
    'Quando usa ação agredir para ataque desarmado, gasta 2 PM para atingir TODAS criaturas adjacentes (incluindo aliados). Único teste de ataque.',
    { prerequisites: [power('class.lutador.trocacao')], minLevel: 8 }),
  electivePower(C, 'Valentão',
    '+2 em testes de ataque e rolagens de dano contra oponentes caídos, desprevenidos, flanqueados ou indefesos.'),
  electivePower(C, 'Voadora',
    'Em investida desarmada, gasta 2 PM. Recebe +1d6 dano para cada 3m que se deslocou (limitado pelo nível).'),
]
