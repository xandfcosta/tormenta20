/**
 * Poderes Concedidos — rule text (book p132-136, Cap 2 "Poderes Concedidos").
 *
 * Keyed by power NAME (the same names `DIVINE_POWERS` uses) because several
 * powers are shared by multiple gods (Coragem Total ×4, Almejar o Impossível
 * ×2, …) — one description, many grantors. Section rules (p132): prerequisite
 * is being devoto of one of the listed gods; the key attribute is Sabedoria.
 *
 * Transcribed from the PDF pages read as images (pdftotext jumbles the
 * two-column layout). 72 unique powers across the 20 deuses maiores.
 */
export const DIVINE_POWER_DESCRIPTIONS: Readonly<Record<string, string>> = {
  'Afinidade com a Tormenta':
    'Você recebe +10 em testes de resistência contra efeitos da Tormenta, de suas criaturas e de devotos de Aharadak. Além disso, seu primeiro poder da Tormenta não conta para perda de Carisma.',
  'Almejar o Impossível':
    'Quando faz um teste de perícia, um resultado de 19 ou mais no dado sempre é um sucesso, não importando o valor a ser alcançado.',
  Anfíbio:
    'Você pode respirar embaixo d’água e adquire deslocamento de natação igual a seu deslocamento terrestre.',
  'Apostar com o Trapaceiro':
    'Quando faz um teste de perícia, você pode gastar 1 PM para apostar com Hyninn. Você e o mestre rolam 1d20, mas o mestre mantém o resultado dele em segredo. Você então escolhe entre usar seu próprio resultado ou o resultado oculto do mestre (neste caso, ele revela o resultado).',
  'Armas da Ambição':
    'Você recebe +1 em testes de ataque e na margem de ameaça com armas nas quais é proficiente.',
  'Arsenal das Profundezas':
    'Você recebe +2 nas rolagens de dano com azagaias, lanças e tridentes e seu multiplicador de crítico com essas armas aumenta em +1.',
  'Astúcia da Serpente':
    'Você recebe +2 em Enganação, Furtividade e Intuição.',
  'Ataque Piedoso':
    'Você pode usar armas corpo a corpo para causar dano não letal sem sofrer a penalidade de –5 no teste de ataque.',
  'Aura de Medo':
    'Você pode gastar 2 PM para gerar uma aura de medo de 9m de raio e duração até o fim da cena. Todos os inimigos que entrem na aura devem fazer um teste de Vontade (CD Car) ou ficam abalados até o fim da cena. Uma criatura que passe no teste de Vontade fica imune a esta habilidade por um dia.',
  'Aura de Paz':
    'Você pode gastar 2 PM para gerar uma aura de paz com 9m de raio e duração de uma cena. Qualquer inimigo dentro da aura que tente fazer uma ação hostil contra você deve fazer um teste de Vontade (CD Car). Se falhar, perderá sua ação. Se passar, fica imune a esta habilidade por um dia.',
  'Aura Restauradora':
    'Efeitos de cura usados por você e seus aliados em um raio de 9m recuperam +1 PV por dado.',
  'Bênção do Mana': 'Você recebe +1 PM a cada nível ímpar.',
  'Carícia Sombria':
    'Você pode gastar 1 PM e uma ação padrão para cobrir sua mão com energia negativa e tocar uma criatura em alcance corpo a corpo. A criatura sofre 2d6 pontos de dano de trevas (Fortitude CD Sab reduz à metade) e você recupera PV iguais à metade do dano causado. Você pode aprender Toque Vampírico como uma magia divina. Se fizer isso, o custo dela diminui em –1 PM.',
  'Centelha Mágica':
    'Escolha uma magia arcana ou divina de 1º círculo. Você aprende e pode lançar essa magia.',
  'Compreender os Ermos':
    'Você recebe +2 em Sobrevivência e pode usar Sabedoria para Adestramento (em vez de Carisma).',
  'Conhecimento Enciclopédico':
    'Você se torna treinado em duas perícias baseadas em Inteligência a sua escolha.',
  'Conjurar Arma':
    'Você pode gastar 1 PM para invocar uma arma corpo a corpo ou de arremesso com a qual seja proficiente. A arma surge em sua mão, fornece +1 em testes de ataque e rolagens de dano, é considerada mágica e dura pela cena. Você não pode criar armas de disparo, mas pode criar 20 munições.',
  'Coragem Total':
    'Você é imune a efeitos de medo, mágicos ou não. Este poder não elimina fobias raciais (como o medo de altura dos minotauros).',
  'Cura Gentil':
    'Você soma seu Carisma aos PV restaurados por seus efeitos mágicos de cura.',
  'Curandeira Perfeita':
    'Você sempre pode escolher 10 em testes de Cura. Além disso, não sofre penalidade por usar essa perícia sem uma maleta de medicamentos. Se possuir o item, recebe +2 no teste de Cura (ou +5, se ele for aprimorado).',
  'Dedo Verde':
    'Você aprende e pode lançar Controlar Plantas. Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
  'Descanso Natural':
    'Para você, dormir ao relento conta como condição de descanso confortável.',
  'Dom da Esperança':
    'Você soma sua Sabedoria em seus PV em vez de Constituição, e se torna imune às condições alquebrado, esmorecido e frustrado.',
  'Dom da Imortalidade':
    'Você é imortal. Sempre que morre, não importando o motivo, volta à vida após 3d6 dias. Apenas paladinos podem escolher este poder. Um personagem pode ter Dom da Imortalidade ou Dom da Ressurreição, mas não ambos.',
  'Dom da Profecia':
    'Você pode lançar Augúrio. Caso aprenda novamente essa magia, seu custo diminui em –1 PM. Você também pode gastar 2 PM para receber +2 em um teste.',
  'Dom da Ressurreição':
    'Você pode gastar uma ação completa e todos os PM que possui (mínimo 1 PM) para tocar o corpo de uma criatura morta há menos de um ano e ressuscitá-la. A criatura volta à vida com 1 PV e 0 PM, e perde 1 ponto de Constituição permanentemente. Este poder só pode ser usado uma vez em cada criatura. Apenas clérigos podem escolher este poder. Um personagem pode ter Dom da Imortalidade ou Dom da Ressurreição, mas não ambos.',
  'Dom da Verdade':
    'Você pode pagar 2 PM para receber +5 em testes de Intuição, e em testes de Percepção contra Enganação e Furtividade, até o fim da cena.',
  'Escamas Dracônicas': 'Você recebe +2 na Defesa e em Fortitude.',
  'Escudo Mágico':
    'Quando lança uma magia, você recebe um bônus na Defesa igual ao círculo da magia lançada até o início do seu próximo turno.',
  'Espada Justiceira':
    'Você pode gastar 1 PM para encantar sua espada (ou outra arma corpo a corpo de corte que esteja empunhando). Ela tem seu dano aumentado em um passo até o fim da cena.',
  'Espada Solar':
    'Você pode gastar 1 PM para fazer uma arma corpo a corpo de corte que esteja empunhando causar +1d6 de dano por fogo até o fim da cena.',
  'Êxtase da Loucura':
    'Toda vez que uma ou mais criaturas falham em um teste de Vontade contra uma de suas habilidades mágicas, você recebe 1 PM temporário cumulativo. Você pode ganhar um máximo de PM temporários por cena desta forma igual a sua Sabedoria.',
  'Familiar Ofídico':
    'Você recebe um familiar cobra (veja a página 38) que não conta em seu limite de parceiros.',
  'Farsa do Fingidor':
    'Você aprende e pode lançar Criar Ilusão. Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
  'Fé Guerreira':
    'Você pode usar Sabedoria para Guerra (em vez de Inteligência). Além disso, em combate, quando vai fazer um teste de perícia, você pode gastar 2 PM para substituí-lo por um teste de Guerra (exceto para testes de ataque).',
  'Forma de Macaco':
    'Você pode gastar uma ação completa e 2 PM para se transformar em um macaco. Você adquire tamanho Minúsculo (o que fornece +5 em Furtividade e –5 em testes de manobra) e recebe deslocamento de escalar 9m. Seu equipamento desaparece (e você perde seus benefícios) até você voltar ao normal, mas suas outras estatísticas não são alteradas. A transformação dura indefinidamente, mas termina caso você faça um ataque, lance uma magia ou sofra dano.',
  'Fulgor Solar':
    'Você recebe redução de frio e trevas 5. Além disso, quando é alvo de um ataque você pode gastar 1 PM para emitir um clarão solar que deixa o atacante ofuscado por uma rodada.',
  'Fúria Divina':
    'Você pode gastar 2 PM para invocar uma fúria selvagem, tornando-se temível em combate. Até o fim da cena, você recebe +2 em testes de ataque e rolagens de dano corpo a corpo, mas não pode executar nenhuma ação que exija paciência ou concentração (como usar a perícia Furtividade ou lançar magias). Se usar este poder em conjunto com a habilidade Fúria, ela também dura uma cena (e não termina se você não atacar ou for alvo de uma ação hostil).',
  'Golpista Divino':
    'Você recebe +2 em Enganação, Jogatina e Ladinagem.',
  'Habitante do Deserto':
    'Você recebe redução de fogo 10 e pode pagar 1 PM para criar água pura e potável suficiente para um odre (ou outro recipiente pequeno).',
  'Inimigo de Tenebra':
    'Seus ataques e habilidades causam +1d6 pontos de dano contra mortos-vivos. Quando você usa um efeito que gera luz, o alcance da iluminação dobra.',
  'Kiai Divino':
    'Uma vez por rodada, quando faz um ataque corpo a corpo, você pode pagar 3 PM. Se acertar o ataque, causa dano máximo, sem necessidade de rolar dados.',
  'Liberdade Divina':
    'Você pode gastar 2 PM para receber imunidade a efeitos de movimento por uma rodada.',
  'Manto da Penumbra':
    'Você aprende e pode lançar Escuridão. Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
  'Mente Analítica':
    'Você recebe +2 em Intuição, Investigação e Vontade.',
  'Mente Vazia': 'Você recebe +2 em Iniciativa, Percepção e Vontade.',
  'Mestre dos Mares':
    'Você pode falar com animais aquáticos (como o efeito da magia Voz Divina) e aprende e pode lançar Acalmar Animal, mas só contra criaturas aquáticas. Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
  'Olhar Amedrontador':
    'Você aprende e pode lançar Amedrontar. Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
  'Palavras de Bondade':
    'Você aprende e pode lançar Enfeitiçar. Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
  'Percepção Temporal':
    'Você pode gastar 3 PM para somar sua Sabedoria (limitado por seu nível e não cumulativo com efeitos que somam este atributo) a seus ataques, Defesa e testes de Reflexos até o fim da cena.',
  'Pesquisa Abençoada':
    'Se passar uma hora pesquisando seus livros e anotações, você pode rolar novamente um teste de perícia baseada em Inteligência ou Sabedoria que tenha feito desde a última cena. Se tiver acesso a mais livros, você recebe um bônus no teste: +2 para uma coleção particular ou biblioteca pequena e +5 para a biblioteca de um templo ou universidade.',
  'Poder Oculto':
    'Você pode gastar uma ação de movimento e 2 PM para invocar a força, a rapidez ou o vigor dos loucos. Role 1d6 para receber +2 em Força (1 ou 2), Destreza (3 ou 4) ou Constituição (5 ou 6) até o fim da cena. Você pode usar este poder várias vezes, mas bônus no mesmo atributo não são cumulativos.',
  'Presas Primordiais':
    'Você pode gastar 1 PM para transformar seus dentes em presas afiadas até o fim da cena. Você recebe uma arma natural de mordida (dano 1d6, crítico x2, perfuração). Uma vez por rodada, quando usa a ação agredir com outra arma, você pode gastar 1 PM para fazer um ataque corpo a corpo extra com a mordida. Se já possuir outro ataque natural de mordida, em vez disso, o dano desse ataque aumenta em dois passos.',
  'Presas Venenosas':
    'Você pode gastar uma ação de movimento e 1 PM para envenenar uma arma corpo a corpo que esteja empunhando. Em caso de acerto, a arma causa perda de 1d12 pontos de vida. A arma permanece envenenada até atingir uma criatura ou até o fim da cena, o que acontecer primeiro.',
  'Rejeição Divina': 'Você recebe resistência a magia divina +5.',
  'Reparar Injustiça':
    'Uma vez por rodada, quando um oponente em alcance curto acerta um ataque em você ou em um de seus aliados, você pode gastar 2 PM para fazer este oponente repetir o ataque, escolhendo o pior entre os dois resultados.',
  'Sangue de Ferro':
    'Você pode pagar 3 PM para receber +2 em rolagens de dano e redução de dano 5 até o fim da cena.',
  'Sangue Ofídico':
    'Você recebe resistência a veneno +5 e a CD para resistir aos seus venenos aumenta em +2.',
  'Servos do Dragão':
    'Você pode gastar uma ação completa e 2 PM para invocar 2d4+1 kobolds capangas em espaços desocupados em alcance curto. Você pode gastar uma ação de movimento para fazer os kobolds andarem (eles têm deslocamento 9m) ou uma ação padrão para fazê-los causar dano a criaturas adjacentes (1d6–1 pontos de dano de perfuração cada). Os kobolds têm For –1, Des 1, Defesa 12, 1 PV e falham automaticamente em qualquer teste de resistência ou oposto. Eles desaparecem quando morrem ou no fim da cena. Os kobolds não agem sem receber uma ordem. Usos criativos para capangas fora de combate ficam a critério do mestre.',
  'Sopro do Mar':
    'Você pode gastar uma ação padrão e 1 PM para soprar vento marinho em um cone de 6m. Criaturas na área sofrem 2d6 pontos de dano de frio (Reflexos CD Sab reduz à metade). Você pode aprender Sopro das Uivantes como uma magia divina. Se fizer isso, o custo dela diminui em –1 PM.',
  'Sorte dos Loucos':
    'Quando faz um teste, você pode pagar 1 PM para rolá-lo novamente (você pode fazer isso mais de uma vez por teste). Se ainda assim falhar, perde 1d6 PM para cada vez que utilizou este poder neste teste.',
  'Talento Artístico':
    'Você recebe +2 em Acrobacia, Atuação e Diplomacia.',
  'Teurgista Místico':
    'Até uma magia de cada círculo que você aprender poderá ser escolhida entre magias divinas (se você for um conjurador arcano) ou entre magias arcanas (se for um conjurador divino). Pré-requisito: habilidade de classe Magias.',
  'Tradição de Lin-Wu':
    'Você considera a katana uma arma simples e, se for proficiente em armas marciais, recebe +1 na margem de ameaça com ela.',
  'Transmissão da Loucura':
    'Você pode lançar Sussurros Insanos (CD Car). Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
  'Tropas Duyshidakk':
    'Você pode gastar uma ação completa e 2 PM para invocar 1d4+1 goblinoides capangas em espaços desocupados em alcance curto. Você pode gastar uma ação de movimento para fazer os goblinoides andarem (eles têm deslocamento 9m) ou uma ação padrão para fazê-los causar dano a criaturas adjacentes (1d6+1 pontos de dano de corte cada). Os goblinoides têm For 1, Des 1, Defesa 15, 1 PV e falham automaticamente em qualquer teste de resistência ou oposto. Eles desaparecem quando morrem ou no fim da cena. Os goblinoides não agem sem receber uma ordem. Usos criativos para capangas fora de combate ficam a critério do mestre.',
  'Urro Divino':
    'Quando faz um ataque ou lança uma magia, você pode pagar 1 PM para somar sua Constituição (mínimo +1) à rolagem de dano desse ataque ou magia.',
  'Visão nas Trevas':
    'Você enxerga perfeitamente no escuro, incluindo em magias de escuridão.',
  'Voz da Civilização': 'Você está sempre sob efeito de Compreensão.',
  'Voz da Natureza':
    'Você pode falar com animais (como o efeito da magia Voz Divina) e aprende e pode lançar Acalmar Animal, mas só contra animais. Caso aprenda novamente essa magia, seu custo diminui em –1 PM.',
  'Voz dos Monstros':
    'Você conhece os idiomas de todos os monstros inteligentes e pode se comunicar livremente com monstros não inteligentes (Int –4 ou menor), como se estivesse sob efeito da magia Voz Divina.',
  Zumbificar:
    'Você pode gastar uma ação completa e 3 PM para reanimar o cadáver de uma criatura Pequena ou Média adjacente por um dia. O cadáver funciona como um parceiro iniciante do tipo a sua escolha entre combatente, fortão ou guardião. Além disso, quando sofre dano, você pode sacrificar esse parceiro; se fizer isso, você sofre apenas metade do dano, mas o cadáver é destruído.',
}
