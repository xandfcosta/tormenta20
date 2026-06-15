import { autoPower, classChoice, electivePower } from './_helpers'
import { CULTO_PALADINO_DO_BEM } from '../deuses'
import type { ClassPower } from '../types'

const C = 'Paladino'

/**
 * PDF Cap 1 (Paladino, p81-84, Tabela 1-18). Auto: Abençoado + Código do
 * Herói + Golpe Divino (+1d8) L1. Cura pelas Mãos (1d8+1) L2. Aura Sagrada
 * L3. Bênção da Justiça + Golpe Divino (+2d8) L5. Cura pelas Mãos (2d8+2)
 * L6. Scaling +1d8 Golpe Divino e +1d8+1 Cura pelas Mãos a cada 4 níveis.
 * Vingador Sagrado L20. 19 power slots L2-L20.
 *
 * Bênção da Justiça: escolha Égide Sagrada ou Montaria Sagrada (L5).
 */
export const PALADINO_POWERS: ClassPower[] = [
  autoPower(C, 1, 'Abençoado',
    'Soma Carisma no PM total no 1° nível. Devoto de deus disponível para paladinos (Azgher, Khalmyr, Lena, Lin-Wu, Marah, Tanna-Toh, Thyatis, Valkaria). Recebe DOIS poderes concedidos. Alternativa: paladino do bem (sem Poder Concedido nem Obrigação/Restrição além do Código).',
  ),
  autoPower(C, 1, 'Código do Herói',
    'Deve manter palavra. Nunca recusar pedido de ajuda de inocente. Nunca mentir, trapacear ou roubar. Se violar, perde todos PM e só recupera no próximo dia.',
  ),
  autoPower(C, 1, 'Golpe Divino (+1d8)',
    'Em ataque corpo-a-corpo, gasta 2 PM para desferir golpe destruidor. Soma Carisma no teste de ataque e +1d8 na rolagem de dano. A cada 4 níveis, pode gastar +1 PM para aumentar dano em +1d8.',
  ),
  autoPower(C, 2, 'Cura pelas Mãos (1d8+1 PV)',
    'Ação de movimento + 1 PM por luz: cura 1d8+1 PV em alvo em alcance corpo-a-corpo (incluindo você). A cada 4 níveis pode gastar +1 PM para aumentar cura em +1d8+1. Causa dano de luz a mortos-vivos (CD Car). L6+: gasta +1 PM para anular condição (abalado, atordoado, apavorado, cego, doente, exausto, fatigado ou surdo).',
  ),
  autoPower(C, 3, 'Aura Sagrada',
    'Gasta 1 PM para gerar aura com 9m de raio. Você e aliados na aura somam Carisma em testes de resistência. Aura emite luz dourada. Duração sustentada.',
  ),
  autoPower(C, 5, 'Bênção da Justiça',
    'Escolha Égide Sagrada ou Montaria Sagrada. Permanente.',
  ),
  autoPower(C, 5, 'Golpe Divino (+2d8)',
    'Dano base do Golpe Divino sobe para +2d8.',
  ),
  autoPower(C, 6, 'Cura pelas Mãos (2d8+2 PV)',
    'Cura base sobe para 2d8+2 PV.',
  ),
  autoPower(C, 9, 'Golpe Divino (+3d8)',
    'Dano base do Golpe Divino sobe para +3d8.',
  ),
  autoPower(C, 10, 'Cura pelas Mãos (3d8+3 PV)',
    'Cura base sobe para 3d8+3 PV.',
  ),
  autoPower(C, 13, 'Golpe Divino (+4d8)',
    'Dano base do Golpe Divino sobe para +4d8.',
  ),
  autoPower(C, 14, 'Cura pelas Mãos (4d8+4 PV)',
    'Cura base sobe para 4d8+4 PV.',
  ),
  autoPower(C, 17, 'Golpe Divino (+5d8)',
    'Dano base do Golpe Divino sobe para +5d8.',
  ),
  autoPower(C, 18, 'Cura pelas Mãos (5d8+5 PV)',
    'Cura base sobe para 5d8+5 PV.',
  ),
  autoPower(C, 20, 'Vingador Sagrado',
    'Capstone: ação completa + 10 PM. Cobre-se de energia divina, assume forma de vingador sagrado até fim cena. Deslocamento de voo 18m e RD 20. Golpe Divino custa metade e causa +2 dados de dano.',
  ),

  // Poderes de Paladino (p82-84)
  electivePower(C, 'Arma Sagrada',
    'Quando usa Golpe Divino com arma preferida da divindade, dado de dano do Golpe Divino aumenta para d12.',
    {
      prerequisites: [
        // Book p82: "devoto de uma divindade (exceto Lena e Marah)". The
        // paladino-do-bem alternative is not devoto of a divindade, so it
        // must also be blocked here even though the book doesn't list it
        // explicitly (it predates the alt being a picker value).
        classChoice(C, 'devoto', 'Devoto, exceto Lena ou Marah', {
          forbidden: ['lena', 'marah', CULTO_PALADINO_DO_BEM],
        }),
      ],
    }),
  electivePower(C, 'Aumento de Atributo',
    '+1 em um atributo. Apenas uma vez por patamar para um mesmo atributo.'),
  electivePower(C, 'Aura Antimagia',
    'Sua aura ativa: você e aliados dentro podem rolar novamente teste de resistência contra magia recém realizado.',
    { minLevel: 14 }),
  electivePower(C, 'Aura Ardente',
    'Sua aura ativa: início de cada turno, espíritos e mortos-vivos dentro sofrem dano de luz = 5 + Carisma.',
    { minLevel: 10 }),
  electivePower(C, 'Aura de Cura',
    'Sua aura ativa: início dos turnos, você e aliados a sua escolha dentro curam PV = 5 + Carisma.',
    { minLevel: 6 }),
  electivePower(C, 'Aura de Invencibilidade',
    'Sua aura ativa: ignora primeiro dano que sofrer na cena. Mesmo para aliados dentro da aura.',
    { minLevel: 18 }),
  electivePower(C, 'Aura Poderosa',
    'Raio da sua aura aumenta para 30m.',
    { minLevel: 6 }),
  electivePower(C, 'Fulgor Divino',
    'Quando usa Golpe Divino, todos inimigos em alcance curto ficam ofuscados até início do próximo turno.'),
  electivePower(C, 'Julgamento Divino: Arrependimento',
    'Gasta 2 PM para marcar inimigo em alcance curto. Próxima vez que esse inimigo acertar ataque em você/aliado, Vontade CD Car ou atordoado no próximo turno dele. Julgamento.'),
  electivePower(C, 'Julgamento Divino: Autoridade',
    'Gasta 1 PM para comandar criatura em alcance curto. Teste Diplomacia oposto Vontade. Vencer = criatura obedece a comando simples (uma vez/cena). Mental. Julgamento.'),
  electivePower(C, 'Julgamento Divino: Coragem',
    'Gasta 2 PM para inspirar coragem em criatura em alcance curto (incluindo você). Imune a medo e +2 ataque contra inimigo com maior ND na cena. Julgamento.'),
  electivePower(C, 'Julgamento Divino: Iluminação',
    'Gasta 2 PM para marcar inimigo em alcance curto. Quando acertar ataque corpo-a-corpo nesse inimigo, recebe 2 PM temp. 1x/cena. Julgamento.'),
  electivePower(C, 'Julgamento Divino: Justiça',
    'Gasta 2 PM para marcar inimigo em alcance curto. Próxima vez que esse inimigo causar dano a você ou aliado, deve fazer teste de Vontade CD Car. Falhar = sofre dano de luz = metade do dano causado. Julgamento.'),
  electivePower(C, 'Julgamento Divino: Libertação',
    'Gasta 5 PM para cancelar uma condição negativa qualquer (abalado, paralisado etc.) afetando criatura em alcance curto. Julgamento.'),
  electivePower(C, 'Julgamento Divino: Salvação',
    'Gasta 2 PM para marcar inimigo em alcance curto. Até fim da cena, quando acertar ataque corpo-a-corpo nesse inimigo, recupera 5 PV. Julgamento.'),
  electivePower(C, 'Julgamento Divino: Vindicação',
    'Gasta 2 PM para marcar inimigo que causou dano a você/aliado. +1 ataque e +1d8 dano contra ele, mas -5 ataque contra outros. L5+: paga +1 PM para aumentar ataque em +1 e dano em +1d8. Termina se alvo inconsciente. Julgamento.'),
  electivePower(C, 'Julgamento Divino: Zelo',
    'Gasta 1 PM para marcar alvo em alcance longo. Pelo restante cena, sempre que se mover em direção a esse alvo, dobra deslocamento. Julgamento.'),
  electivePower(C, 'Orar',
    'Aprende e pode lançar magia divina de 1° círculo a sua escolha. Atributo-chave: Sabedoria. Pode escolher várias vezes.'),
  electivePower(C, 'Virtude Paladinesca: Caridade',
    'Custo de habilidades de paladino que tenham aliado como alvo é reduzido em -1 PM.'),
  electivePower(C, 'Virtude Paladinesca: Castidade',
    'Imune a efeitos de encantamento. +5 Intuição para perceber blefes.'),
  electivePower(C, 'Virtude Paladinesca: Compaixão',
    'Pode usar Cura pelas Mãos em alcance curto. Para cada PM gasto, cura 2d6+1 (em vez de 1d8+1).'),
  electivePower(C, 'Virtude Paladinesca: Humildade',
    'Na primeira rodada de combate, gasta ação completa para rezar/pedir orientação. Recebe PM temporários iguais ao Carisma (duram fim cena).'),
  electivePower(C, 'Virtude Paladinesca: Temperança',
    'Quando ingere alimento, item alquímico ou poção, consome apenas metade. Cada item desses rende duas "doses" para você.'),

  // Caminhos da Bênção da Justiça (L5) — gated by caminho choice.
  electivePower(C, 'Caminho: Égide Sagrada',
    'Ação de movimento + 2 PM: recobre escudo/símbolo sagrado de energia. Você e aliados adjacentes somam Carisma na Defesa até fim cena. L11+: paga 5 PM para refazer resistência se passar e mágica era contra você, devolve a magia ao conjurador.',
    {
      minLevel: 5,
      prerequisites: [
        classChoice(C, 'caminho', 'Caminho da Égide Sagrada escolhido', {
          allowed: ['egide-sagrada'],
        }),
      ],
    }),
  electivePower(C, 'Caminho: Montaria Sagrada',
    'Ação de movimento + 2 PM: invoca montaria sagrada (cavalo de guerra Médio ou pônei para Pequenos). Parceiro veterano. L11 vira parceiro mestre. Se morrer, nova após 1 dia de prece. Variante: animal mundano (sem PM, vínculo mental).',
    {
      minLevel: 5,
      prerequisites: [
        classChoice(C, 'caminho', 'Caminho da Montaria Sagrada escolhido', {
          allowed: ['montaria-sagrada'],
        }),
      ],
    }),
]
