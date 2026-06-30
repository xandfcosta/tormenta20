/**
 * Poderes Concedidos — granted powers given by Devoção (Tormenta 20
 * PDF Cap 2). The list of which deus grants which poder is on the
 * Tabela 1-20 (p97) and per-deus blocks (p96-105); the mechanical
 * description text lives in the chapter's "Poderes Concedidos"
 * subsection (p132-136).
 *
 * Distinct from `GeneralPower`:
 *  - No `Prerequisite[]` — the gate is Devoção to one of `deuses`.
 *  - Multiple deuses may grant the same poder (e.g. Coragem Total →
 *    Arsenal/Khalmyr/Lin-Wu/Valkaria).
 *  - Side-grade powers (spell-likes, channels, situational triggers).
 *
 * `bookPage` reflects the description page in the PDF (132-136),
 * NOT the per-deus block. Cross-references against the `Deus` record
 * are validated in `__tests__/granted-powers.test.ts`.
 */
export type GrantedPowerKind =
  | 'ataque'
  | 'defesa'
  | 'utilidade'
  | 'sentido'
  | 'social'
  | 'movimento'
  | 'magia'

export type GrantedPower = {
  id: string
  name: string
  deuses: readonly string[]
  effect: string
  kind: GrantedPowerKind
  bookPage: number
}

export const GRANTED_POWERS: readonly GrantedPower[] = Object.freeze([
  {
    id: 'coragem-total',
    name: 'Coragem Total',
    deuses: ['Arsenal', 'Khalmyr', 'Lin-Wu', 'Valkaria'],
    effect:
      'Você é imune a efeitos de medo, mágicos ou não. Este poder não elimina fobias raciais (como o medo de altura dos minotauros).',
    kind: 'defesa',
    bookPage: 133,
  },
  {
    id: 'espada-justiceira',
    name: 'Espada Justiceira',
    deuses: ['Khalmyr'],
    effect:
      'Você pode gastar 1 PM para encantar sua espada (ou outra arma corpo a corpo de corte que esteja empunhando). Ela tem seu dano aumentado em um passo até o fim da cena.',
    kind: 'ataque',
    bookPage: 133,
  },
  {
    id: 'dom-da-verdade',
    name: 'Dom da Verdade',
    deuses: ['Khalmyr'],
    effect:
      'Você pode gastar 2 PM para receber +5 em testes de Intuição, e em testes de Percepção contra Enganação e Furtividade, até o fim da cena.',
    kind: 'sentido',
    bookPage: 133,
  },
  {
    id: 'reparar-injustica',
    name: 'Reparar Injustiça',
    deuses: ['Khalmyr'],
    effect:
      'Uma vez por rodada, quando um oponente em alcance curto acerta um ataque em você ou em um de seus aliados, você pode gastar 2 PM para fazer este oponente repetir o ataque, escolhendo o pior entre os dois resultados.',
    kind: 'defesa',
    bookPage: 135,
  },
  {
    id: 'conjurar-arma',
    name: 'Conjurar Arma',
    deuses: ['Arsenal'],
    effect:
      'Você pode gastar 1 PM para invocar uma arma corpo a corpo ou de arremesso com a qual seja proficiente. A arma surge em sua mão, fornece +1 em testes de ataque e rolagens de dano, é considerada mágica e dura pela cena. Você não pode criar armas de disparo, mas pode criar 20 munições.',
    kind: 'ataque',
    bookPage: 132,
  },
  {
    id: 'fe-guerreira',
    name: 'Fé Guerreira',
    deuses: ['Arsenal'],
    effect:
      'Você pode usar Sabedoria para Guerra (em vez de Inteligência). Além disso, quando vai fazer um teste de perícia, você pode gastar 2 PM para substituí-lo por um teste de Guerra (exceto para testes de ataque).',
    kind: 'utilidade',
    bookPage: 133,
  },
  {
    id: 'sangue-de-ferro',
    name: 'Sangue de Ferro',
    deuses: ['Arsenal'],
    effect:
      'Você pode pagar 3 PM para receber +2 em rolagens de dano e redução de dano 5 até o fim da cena.',
    kind: 'ataque',
    bookPage: 135,
  },
  {
    id: 'espada-solar',
    name: 'Espada Solar',
    deuses: ['Azgher'],
    effect:
      'Você pode gastar 1 PM para fazer uma arma corpo a corpo de corte que esteja empunhando causar +1d6 de dano por fogo até o fim da cena.',
    kind: 'ataque',
    bookPage: 133,
  },
  {
    id: 'fulgor-solar',
    name: 'Fulgor Solar',
    deuses: ['Azgher'],
    effect:
      'Você recebe redução de frio e trevas 5. Além disso, quando é alvo de um ataque, você pode gastar 1 PM para emitir um clarão solar que deixa o atacante ofuscado por uma rodada.',
    kind: 'defesa',
    bookPage: 134,
  },
  {
    id: 'habitante-do-deserto',
    name: 'Habitante do Deserto',
    deuses: ['Azgher'],
    effect:
      'Você recebe redução de fogo 10 e pode pagar 1 PM para criar água boa e potável suficiente para um odre (ou outro recipiente pequeno).',
    kind: 'utilidade',
    bookPage: 134,
  },
  {
    id: 'inimigo-de-tenebra',
    name: 'Inimigo de Tenebra',
    deuses: ['Azgher'],
    effect:
      'Seus ataques e habilidades causam +1d6 pontos de dano contra mortos-vivos. Quando você usa um efeito que gera luz, o alcance da iluminação dobra.',
    kind: 'ataque',
    bookPage: 134,
  },
  {
    id: 'afinidade-com-a-tormenta',
    name: 'Afinidade com a Tormenta',
    deuses: ['Aharadak'],
    effect:
      'Você recebe +10 em testes de resistência contra efeitos da Tormenta, de suas criaturas e de devotos de Aharadak. Além disso, seu primeiro poder da Tormenta não conta para perda de Carisma.',
    kind: 'defesa',
    bookPage: 132,
  },
  {
    id: 'extase-da-loucura',
    name: 'Êxtase da Loucura',
    deuses: ['Aharadak', 'Nimb'],
    effect:
      'Toda vez que uma ou mais criaturas falham em um teste de Vontade contra uma de suas habilidades mágicas, você recebe 1 PM temporário cumulativo. Você pode ganhar um máximo de PM temporários por cena desta forma igual a sua Sabedoria.',
    kind: 'magia',
    bookPage: 133,
  },
  {
    id: 'percepcao-temporal',
    name: 'Percepção Temporal',
    deuses: ['Aharadak'],
    effect:
      'Você pode gastar 3 PM para somar sua Sabedoria (limitado por seu nível e não cumulativo com efeitos que somam este atributo) a seus ataques, Defesa e testes de Reflexos até o fim da cena.',
    kind: 'sentido',
    bookPage: 134,
  },
  {
    id: 'rejeicao-divina',
    name: 'Rejeição Divina',
    deuses: ['Aharadak'],
    effect: 'Você recebe resistência a magia divina +5.',
    kind: 'defesa',
    bookPage: 135,
  },
  {
    id: 'compreender-os-ermos',
    name: 'Compreender os Ermos',
    deuses: ['Allihanna'],
    effect:
      'Você recebe +2 em Sobrevivência e pode usar Sabedoria para Adestramento (em vez de Carisma).',
    kind: 'utilidade',
    bookPage: 132,
  },
  {
    id: 'dedo-verde',
    name: 'Dedo Verde',
    deuses: ['Allihanna'],
    effect:
      'Você aprende e pode lançar Controlar Plantas. Caso aprenda novamente essa magia, seu custo diminui em -1 PM.',
    kind: 'magia',
    bookPage: 133,
  },
  {
    id: 'descanso-natural',
    name: 'Descanso Natural',
    deuses: ['Allihanna'],
    effect:
      'Para você, dormir ao relento conta como condição de descanso confortável.',
    kind: 'utilidade',
    bookPage: 133,
  },
  {
    id: 'voz-da-natureza',
    name: 'Voz da Natureza',
    deuses: ['Allihanna'],
    effect:
      'Você pode falar com animais (como o efeito da magia Voz Divina) e aprende e pode lançar Acalmar Animal, mas só contra animais. Caso aprenda novamente essa magia, seu custo diminui em -1 PM.',
    kind: 'magia',
    bookPage: 136,
  },
  {
    id: 'apostar-com-o-trapaceiro',
    name: 'Apostar com o Trapaceiro',
    deuses: ['Hyninn'],
    effect:
      'Quando faz um teste de perícia, você pode gastar 1 PM para apostar com Hyninn. Você e o mestre rolam 1d20, mas o mestre mantém o resultado dele em segredo. Você então escolhe entre usar seu próprio resultado ou o resultado oculto do mestre (neste caso, ele revela o resultado).',
    kind: 'utilidade',
    bookPage: 132,
  },
  {
    id: 'farsa-do-fingidor',
    name: 'Farsa do Fingidor',
    deuses: ['Hyninn'],
    effect:
      'Você aprende e pode lançar Criar Ilusão. Caso aprenda novamente essa magia, seu custo diminui em -1 PM.',
    kind: 'magia',
    bookPage: 133,
  },
  {
    id: 'golpista-divino',
    name: 'Golpista Divino',
    deuses: ['Hyninn'],
    effect: 'Você recebe +2 em Enganação, Jogatina e Ladinagem.',
    kind: 'social',
    bookPage: 134,
  },
  {
    id: 'forma-de-macaco',
    name: 'Forma de Macaco',
    deuses: ['Hyninn'],
    effect:
      'Você pode gastar uma ação completa e 2 PM para se transformar em um macaco. Você adquire tamanho Minúsculo (o que fornece +5 em Furtividade e -5 em testes de manobra) e recebe deslocamento de escalar 9m. Seu equipamento desaparece (e você perde seus benefícios) até você voltar ao normal, mas suas outras estatísticas não são alteradas. A transformação dura indefinidamente, terminando caso você faça um ataque, lance uma magia ou sofra dano.',
    kind: 'movimento',
    bookPage: 134,
  },
  {
    id: 'aura-de-medo',
    name: 'Aura de Medo',
    deuses: ['Kallyadranoch'],
    effect:
      'Você pode gastar 2 PM para gerar uma aura de medo de 9m de raio e duração até o fim da cena. Todos os inimigos que entrem na aura devem fazer um teste de Vontade (CD Car) ou ficam abalados até o fim da cena. Uma criatura que passe no teste de Vontade fica imune a esta habilidade por um dia.',
    kind: 'ataque',
    bookPage: 132,
  },
  {
    id: 'escamas-draconicas',
    name: 'Escamas Dracônicas',
    deuses: ['Kallyadranoch'],
    effect: 'Você recebe +2 na Defesa e em Fortitude.',
    kind: 'defesa',
    bookPage: 133,
  },
  {
    id: 'presas-primordiais',
    name: 'Presas Primordiais',
    deuses: ['Kallyadranoch', 'Megalokk'],
    effect:
      'Você pode gastar 1 PM para transformar seus dentes em presas afiadas até o fim da cena. Você recebe uma arma natural de mordida (dano 1d6, crítico x2, perfuração). Uma vez por rodada, quando usa a ação agredir com outra arma, você pode gastar 1 PM para fazer um ataque corpo a corpo extra com a mordida. Se já possuir outro ataque natural de mordida, em vez disso, o dano desse ataque aumenta em dois passos.',
    kind: 'ataque',
    bookPage: 135,
  },
  {
    id: 'servos-do-dragao',
    name: 'Servos do Dragão',
    deuses: ['Kallyadranoch'],
    effect:
      'Você pode gastar uma ação completa e 2 PM para invocar 2d4+1 kobolds capangas em espaços desocupados em alcance curto. Você pode gastar uma ação de movimento para fazer os kobolds andarem (eles têm deslocamento 9m) ou uma ação padrão para fazê-los causar dano a criaturas adjacentes (1d6-1 pontos de dano de perfuração cada). Os kobolds têm For -1, Des 1, Defesa 12, 1 PV e falham automaticamente em qualquer teste de resistência ou oposto. Eles desaparecem quando morrem ou no fim da cena. Os kobolds não agem sem receber uma ordem.',
    kind: 'magia',
    bookPage: 135,
  },
  {
    id: 'ataque-piedoso',
    name: 'Ataque Piedoso',
    deuses: ['Lena', 'Thyatis'],
    effect:
      'Você pode usar armas corpo a corpo para causar dano não letal sem sofrer a penalidade de -5 no teste de ataque.',
    kind: 'ataque',
    bookPage: 132,
  },
  {
    id: 'aura-restauradora',
    name: 'Aura Restauradora',
    deuses: ['Lena'],
    effect:
      'Efeitos de cura usados por você e seus aliados em um raio de 9m recuperam +1 PV por dado.',
    kind: 'utilidade',
    bookPage: 132,
  },
  {
    id: 'cura-gentil',
    name: 'Cura Gentil',
    deuses: ['Lena'],
    effect:
      'Você soma seu Carisma aos PV restaurados por seus efeitos mágicos de cura.',
    kind: 'utilidade',
    bookPage: 133,
  },
  {
    id: 'curandeira-perfeita',
    name: 'Curandeira Perfeita',
    deuses: ['Lena'],
    effect:
      'Você sempre pode escolher 10 em testes de Cura. Além disso, não sofre penalidade por usar essa perícia sem uma maleta de medicamentos. Se possuir o item, recebe +2 no teste de Cura (ou +5, se ele for aprimorado).',
    kind: 'utilidade',
    bookPage: 133,
  },
  {
    id: 'kiai-divino',
    name: 'Kiai Divino',
    deuses: ['Lin-Wu'],
    effect:
      'Uma vez por rodada, quando faz um ataque corpo a corpo, você pode pagar 3 PM. Ao acertar o ataque, causa dano máximo, sem necessidade de rolar dados.',
    kind: 'ataque',
    bookPage: 134,
  },
  {
    id: 'mente-vazia',
    name: 'Mente Vazia',
    deuses: ['Lin-Wu'],
    effect: 'Você recebe +2 em Iniciativa, Percepção e Vontade.',
    kind: 'defesa',
    bookPage: 134,
  },
  {
    id: 'tradicao-de-lin-wu',
    name: 'Tradição de Lin-Wu',
    deuses: ['Lin-Wu'],
    effect:
      'Você considera a katana uma arma simples e, se for proficiente em armas marciais, recebe +1 na margem de ameaça com ela.',
    kind: 'ataque',
    bookPage: 135,
  },
  {
    id: 'aura-de-paz',
    name: 'Aura de Paz',
    deuses: ['Marah'],
    effect:
      'Você pode gastar 2 PM para gerar uma aura de paz com 9m de raio e duração de uma cena. Qualquer inimigo dentro da aura que tente fazer uma ação hostil contra você deve fazer um teste de Vontade (CD Car). Se falhar, perderá sua ação. Se passar, fica imune a esta habilidade por um dia.',
    kind: 'defesa',
    bookPage: 132,
  },
  {
    id: 'bencao-do-mana',
    name: 'Bênção do Mana',
    deuses: ['Wynna'],
    effect: 'Você recebe +1 PM a cada nível ímpar.',
    kind: 'magia',
    bookPage: 132,
  },
])

const byId = new Map(GRANTED_POWERS.map((p) => [p.id, p]))
const byName = new Map(GRANTED_POWERS.map((p) => [p.name, p]))

export function grantedPowerById(id: string): GrantedPower | undefined {
  return byId.get(id)
}

export function grantedPowerByName(name: string): GrantedPower | undefined {
  return byName.get(name)
}

/**
 * Powers granted by a specific deus. Lookup is by `Deus.name` since
 * `deuses[]` in the catalog stores names (not ids) to mirror the
 * book's prose.
 */
export function grantedPowersByDeus(deusName: string): readonly GrantedPower[] {
  return GRANTED_POWERS.filter((p) => p.deuses.includes(deusName))
}

export function grantedPowersByKind(
  kind: GrantedPowerKind,
): readonly GrantedPower[] {
  return GRANTED_POWERS.filter((p) => p.kind === kind)
}
