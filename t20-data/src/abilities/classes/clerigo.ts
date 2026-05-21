import { anyPower, autoPower, classChoice, electivePower, power } from './_helpers'
import { DEUSES } from '../deuses'

const CLERIGO_DEUSES_MAIORES = DEUSES.filter((d) => d.major).map((d) => d.id)
import type { ClassPower } from '../types'

const C = 'Clérigo'

/**
 * PDF Cap 1 (Clérigo, p56-58, Tabela 1-11). Auto: Devoto Fiel + Magias L1.
 * Magias 2°/3°/4°/5° em L5/9/13/17. Mão da Divindade L20. 19 power slots
 * L2-L20.
 */
export const CLERIGO_POWERS: ClassPower[] = [
  autoPower(C, 1, 'Devoto Fiel',
    'Devoto de deus maior. Recebe DOIS poderes concedidos por se tornar devoto (em vez de um). Alternativa: cultuar o Panteão (não recebe Poder Concedido; só pode usar maças). Atributo-chave para magias: Sabedoria.',
  ),
  autoPower(C, 1, 'Magias (1° círculo)',
    'Pode lançar magias divinas de 1° círculo. Soma Sabedoria no PM total.',
  ),
  autoPower(C, 5, 'Magias (2° círculo)',
    'Pode lançar magias divinas de 2° círculo.',
  ),
  autoPower(C, 9, 'Magias (3° círculo)',
    'Pode lançar magias divinas de 3° círculo.',
  ),
  autoPower(C, 13, 'Magias (4° círculo)',
    'Pode lançar magias divinas de 4° círculo.',
  ),
  autoPower(C, 17, 'Magias (5° círculo)',
    'Pode lançar magias divinas de 5° círculo.',
  ),
  autoPower(C, 20, 'Mão da Divindade',
    'Capstone: ação completa + 15 PM. Lança 3 magias divinas (qualquer círculo, incluindo magias que não conhece), uma como ação livre e sem gastar PM (paga outros custos). Depois fica atordoado 1d4 rodadas.',
  ),

  // Poderes de Clérigo (p57-58)
  electivePower(C, 'Abençoar Arma',
    'Treinado na arma preferida da divindade. Pode empunhar e gastar ação de movimento + 3 PM: arma vira mágica, +1 passo de dano, usa Sabedoria nas rolagens de ataque e dano.'),
  electivePower(C, 'Aumento de Atributo',
    '+1 em um atributo. Apenas uma vez por patamar para um mesmo atributo.'),
  electivePower(C, 'Autoridade Eclesiástica',
    'Posição formal em igreja. +5 em testes de Diplomacia ou Intimidação com devotos. Metade do preço em itens alquímicos/poções/serviços em templos da divindade.',
    {
      prerequisites: [
        classChoice(C, 'devoto', 'Devoto de deus maior', {
          allowed: CLERIGO_DEUSES_MAIORES,
        }),
      ],
      minLevel: 5,
    }),
  electivePower(C, 'Canalizar Energia Positiva/Negativa',
    'Ação padrão + PM. Onda de luz/trevas afeta criaturas em alcance curto. Cada PM gasto: luz cura 1d6 PV em criaturas vivas e causa 1d6 dano de luz em mortos-vivos (Vontade CD Sab reduz à metade). Trevas inverso.'),
  electivePower(C, 'Canalizar Amplo',
    'Quando usa Canalizar Energia, gasta +2 PM para aumentar alcance para médio.',
    { prerequisites: [power('class.clerigo.canalizar-energia-positiva-negativa')] }),
  electivePower(C, 'Comunhão Vital',
    'Quando lança magia que cura criatura, paga +2 PM para outra criatura em alcance curto (incluindo você mesmo) recuperar PV iguais à metade da cura original.'),
  electivePower(C, 'Conhecimento Mágico',
    'Aprende 2 magias de qualquer círculo que possa lançar. Pode escolher várias vezes.'),
  electivePower(C, 'Expulsar/Comandar Mortos-Vivos',
    'Ação padrão + 3 PM: expulsa (energia positiva) ou comanda (energia negativa) todos mortos-vivos em alcance curto. Expulsos ficam apavorados por 1d6 rodadas. Comandados não inteligentes seguem ordens por 1 dia. Vontade CD Sab evita.',
    { prerequisites: [power('class.clerigo.canalizar-energia-positiva-negativa')] }),
  electivePower(C, 'Liturgia Mágica',
    'Ação de movimento: liturgia breve. Sua fé aumenta. CD para resistir à sua próxima habilidade de clérigo (usada até fim do próximo turno) +1.'),
  electivePower(C, 'Magia Sagrada/Profana',
    'Quando lança magia divina que causa dano, paga +1 PM para mudar tipo do dano da magia para luz (energia positiva) ou trevas (energia negativa).'),
  electivePower(C, 'Mestre Celebrante',
    'Quantidade de pessoas afetadas por suas missas aumenta em 10x e benefícios dobram.',
    { prerequisites: [anyPower(['class.clerigo.missa-bencao-da-vida', 'class.clerigo.missa-chamado-as-armas', 'class.clerigo.missa-elevacao-do-espirito', 'class.clerigo.missa-escudo-divino', 'class.clerigo.missa-superar-as-limitacoes'])], minLevel: 12 }),
  electivePower(C, 'Missa: Bênção da Vida',
    'Participantes recebem PV temporários = seu nível + Sabedoria. Missa.'),
  electivePower(C, 'Missa: Chamado às Armas',
    'Participantes recebem +1 em testes de ataque e rolagens de dano. Missa.'),
  electivePower(C, 'Missa: Elevação do Espírito',
    'Participantes recebem PM temporários = sua Sabedoria. Missa.'),
  electivePower(C, 'Missa: Escudo Divino',
    'Participantes recebem +1 Defesa e testes de resistência. Missa.'),
  electivePower(C, 'Missa: Superar as Limitações',
    'Cada participante recebe +1d6 num único teste a sua escolha. Pode usá-lo até próximo dia. Missa.'),
  electivePower(C, 'Prece de Combate',
    'Quando lança magia divina com tempo de conjuração de ação padrão em si mesmo, paga +2 PM para lançá-la como ação de movimento.'),
  electivePower(C, 'Símbolo Sagrado Energizado',
    'Ação de movimento + 1 PM: símbolo emite luz dourada/púrpura até fim da cena. Custo em PM para magias divinas diminui em 1 enquanto empunha símbolo energizado.'),
]
