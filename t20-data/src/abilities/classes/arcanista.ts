import { anyPower, autoPower, classChoice, electivePower, note, power } from './_helpers'
import type { ClassPower } from '../types'

const C = 'Arcanista'

/**
 * PDF Cap 1 (Arcanista, p36-39, Tabela 1-5). Auto: Caminho L1, Magias scaling
 * (1°/2°/3°/4°/5° em L1/5/9/13/17), Alta Arcana L20. 19 power slots L2-L20.
 *
 * Caminhos (escolhidos no L1, exclusivos): Bruxo (Cha, foco), Feiticeiro
 * (Car, linhagem), Mago (Int, grimório).
 */
export const ARCANISTA_POWERS: ClassPower[] = [
  autoPower(C, 1, 'Caminho do Arcanista',
    'Escolha Bruxo, Feiticeiro ou Mago. Define atributo-chave de magia (Cha/Int conforme caminho) e habilidade exclusiva. Escolha permanente.',
  ),
  autoPower(C, 1, 'Magias (1° círculo)',
    'Pode lançar magias arcanas de 1° círculo. Atributo-chave conforme caminho. Soma atributo-chave no total de PM.',
  ),
  autoPower(C, 5, 'Magias (2° círculo)',
    'Pode lançar magias arcanas de 2° círculo.',
  ),
  autoPower(C, 9, 'Magias (3° círculo)',
    'Pode lançar magias arcanas de 3° círculo.',
  ),
  autoPower(C, 13, 'Magias (4° círculo)',
    'Pode lançar magias arcanas de 4° círculo.',
  ),
  autoPower(C, 17, 'Magias (5° círculo)',
    'Pode lançar magias arcanas de 5° círculo.',
  ),
  autoPower(C, 20, 'Alta Arcana',
    'Capstone: custo em PM de magias arcanas reduzido à metade (após aplicar aprimoramentos e outros efeitos).',
  ),

  // Poderes de Arcanista (p38-39)
  electivePower(C, 'Arcano de Batalha',
    'Quando lança magia, soma atributo-chave na rolagem de dano.'),
  electivePower(C, 'Aumento de Atributo',
    '+1 em um atributo. Apenas uma vez por patamar para um mesmo atributo.'),
  electivePower(C, 'Caldeirão do Bruxo',
    'Pode criar poções como se tivesse Preparar Poção. Se tiver ambos, custo para criar poções de até 5° círculo é reduzido.',
    { prerequisites: [power('class.arcanista.caminho-bruxo'), note('Treinado em Ofício (alquimista)')] }),
  electivePower(C, 'Conhecimento Mágico',
    'Aprende 2 magias de qualquer círculo que possa lançar. Pode ser escolhido várias vezes.'),
  electivePower(C, 'Contramágica Aprimorada',
    'Uma vez por rodada, contramágica como reação.',
    { prerequisites: [note('Conhecer a magia Dissipar Magia')] }),
  electivePower(C, 'Envolto em Mistério',
    'Aparência assombrosa: +5 Enganação e Intimidação contra pessoas não treinadas em Conhecimento ou Misticismo.'),
  electivePower(C, 'Escriba Arcano',
    'Pode aprender magia copiando pergaminhos/grimórios. Exige 1 dia e T$ 250 por PM da magia.',
    { prerequisites: [power('class.arcanista.caminho-mago'), note('Treinado em Ofício (escriba)')] }),
  electivePower(C, 'Especialista em Escola',
    'Escolha uma escola — CD para resistir às suas magias dessa escola +2.',
    { prerequisites: [anyPower(['class.arcanista.caminho-bruxo', 'class.arcanista.caminho-mago'])] }),
  electivePower(C, 'Familiar',
    'Possui animal mágico telepático com habilidade especial. Ver quadro Familiares Arcanos.'),
  electivePower(C, 'Fluxo de Mana',
    'Pode manter 2 efeitos sustentados simultaneamente como ação livre, pagando custo de cada separadamente.',
    { minLevel: 10 }),
  electivePower(C, 'Foco Vital',
    'Se foco sofrer dano que o levaria a 0 PV ou menos, fica com 1 PV e perde PV foco igual ao excedente (ou é destruído).',
    { prerequisites: [power('class.arcanista.caminho-bruxo')] }),
  electivePower(C, 'Fortalecimento Arcano',
    'CD para resistir a suas magias +1. Se pode lançar magias de 4° círculo, sobe para +2.',
    { minLevel: 5 }),
  electivePower(C, 'Herança Aprimorada',
    'Recebe a herança aprimorada de sua linhagem.',
    { prerequisites: [power('class.arcanista.caminho-feiticeiro')], minLevel: 6 }),
  electivePower(C, 'Herança Superior',
    'Recebe a herança superior de sua linhagem.',
    { prerequisites: [power('class.arcanista.heranca-aprimorada')], minLevel: 11 }),
  electivePower(C, 'Magia Pungente',
    'Quando lança magia, pode pagar 1 PM para aumentar CD para resistir em +2.'),
  electivePower(C, 'Mestre em Escola',
    'Custo em PM das magias da escola escolhida reduzido em -1.',
    { prerequisites: [power('class.arcanista.especialista-em-escola')], minLevel: 8 }),
  electivePower(C, 'Poder Mágico',
    '+1 PM por nível de arcanista (retroativo e prospectivo).'),
  electivePower(C, 'Raio Arcano',
    'Ação padrão, 1 PM: causa 1d8 dano arcano em alvo em alcance curto. +1d8 por círculo de magia acima do 1° que pode lançar. Alvo: Reflexos (CD atributo-chave) reduz à metade.'),
  electivePower(C, 'Raio Elemental',
    'Quando usa Raio Arcano, pode pagar +1 PM para mudar dano para ácido/eletricidade/fogo/frio/trevas e impor condição.',
    { prerequisites: [power('class.arcanista.raio-arcano')] }),
  electivePower(C, 'Raio Poderoso',
    'Dados de dano do Raio Arcano sobem para d12 e alcance vira médio.',
    { prerequisites: [power('class.arcanista.raio-arcano')] }),
  electivePower(C, 'Tinta do Mago',
    'Pode criar pergaminhos (como Escrever Pergaminho). Se tiver ambos, custo reduzido à metade.',
    { prerequisites: [power('class.arcanista.caminho-mago'), note('Treinado em Ofício (escriba)')] }),

  // Caminhos do Arcanista — gated by the caminho choice in classChoices.
  // Player owns the row matching their picked caminho automatically once
  // the picker is set; the other two stay locked.
  electivePower(C, 'Caminho: Bruxo',
    'Lança magias via foco (varinha, cajado, chapéu). CD 20 + custo em PM, ou teste de Misticismo. Foco tem RD 10 e PV iguais à metade dos seus. Atributo-chave: Carisma.',
    {
      prerequisites: [
        classChoice(C, 'caminho', 'Caminho do Bruxo escolhido', {
          allowed: ['bruxo'],
        }),
      ],
    }),
  electivePower(C, 'Caminho: Feiticeiro',
    'Magia inata por linhagem (Draconia, Feérica, Rubra). Aprende magias a cada nível ímpar. Atributo-chave: Carisma.',
    {
      prerequisites: [
        classChoice(C, 'caminho', 'Caminho do Feiticeiro escolhido', {
          allowed: ['feiticeiro'],
        }),
      ],
    }),
  electivePower(C, 'Caminho: Mago',
    'Estudo formal. Memoriza magias do grimório uma vez por dia. Atributo-chave: Inteligência.',
    {
      prerequisites: [
        classChoice(C, 'caminho', 'Caminho do Mago escolhido', {
          allowed: ['mago'],
        }),
      ],
    }),
]
