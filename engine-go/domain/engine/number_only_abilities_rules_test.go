package engine

import (
	"strconv"
	"testing"
)

// AS HABILIDADES QUE SÃO SÓ UM NÚMERO (cap. 1, ALE-484).
//
// Cada linha é um verbete do livro cujo efeito é um número num alvo que o motor
// JÁ soma, e o esperado é o do texto da página, escrito à mão. Elas existiam no
// catálogo como prosa: a ficha listava a habilidade e não somava nada.
//
// Dois jeitos de valer, e o caso diz qual:
//
//   - FIXO entra na pilha sempre — o +2 em Atletismo do Gatuno;
//   - SITUACIONAL vira um interruptor que o jogador liga, porque só a mesa sabe
//     se a situação aconteceu — o +2 no subterrâneo do anão. O caso afirma que
//     ele é OFERECIDO com o número do livro; que o ligado soma é o
//     `ApplyActiveConditionals`, provado nos testes dele.
type numberOnlyCase struct {
	name     string
	ch       Character
	fixed    []expectedTerm
	switched []expectedTerm
}

type expectedTerm struct {
	target ModifierTarget
	amount int
	dice   string
}

func classCase(class, power string) Character {
	return Character{
		Level: 20, Classes: []CharacterClass{{ClassName: class, Level: 20}},
		ClassPowers: `["` + power + `"]`, PowerChoices: `{}`,
	}
}

func raceCase(race string) Character {
	return Character{Level: 1, Races: []CharacterRace{{Race: race}}}
}

func expertise(name string) ModifierTarget { return ModifierTarget{K: "expertise", Name: name} }

func TestTheNumberOnlyAbilitiesAddWhatTheBookSays(t *testing.T) {
	world := BookRuleset(primeBook(t))

	resistance := ModifierTarget{K: "resistance"}
	attackAll := ModifierTarget{K: "attack", Scope: "all"}
	damageAll := ModifierTarget{K: "damage", Scope: "all"}
	grapple := ModifierTarget{K: "maneuver", Name: "agarrar"}
	trip := ModifierTarget{K: "maneuver", Name: "derrubar"}

	cases := []numberOnlyCase{
		// p42 "Você recebe resistência a magia +5."
		{name: "Superstição", ch: classCase("Bárbaro", "class.barbaro.supersticao"),
			switched: []expectedTerm{{target: resistance, amount: 5}}},
		// p42 "Seu deslocamento aumenta em +3m." — 3m são 2 quadrados.
		{name: "Fúria da Savana", ch: classCase("Bárbaro", "class.barbaro.furia-da-savana"),
			fixed: []expectedTerm{{target: ModifierTarget{K: "displacement"}, amount: 2}}},
		// p76 "Você recebe resistência a medo e mental +5."
		{name: "Convencido", ch: classCase("Lutador", "class.lutador.convencido"),
			switched: []expectedTerm{{target: resistance, amount: 5}, {target: resistance, amount: 5}}},
		// p48 "Você recebe +2 em Acrobacia e Atletismo."
		{name: "Pernas do Mar", ch: classCase("Bucaneiro", "class.bucaneiro.pernas-do-mar"),
			fixed: []expectedTerm{{target: expertise("Acrobacia"), amount: 2}, {target: expertise("Atletismo"), amount: 2}}},
		// p54 "Você recebe +2 em Diplomacia e Nobreza"
		{name: "Etiqueta", ch: classCase("Cavaleiro", "class.cavaleiro.etiqueta"),
			fixed: []expectedTerm{{target: expertise("Diplomacia"), amount: 2}, {target: expertise("Nobreza"), amount: 2}}},
		// p54 "Você recebe +2 em Diplomacia, por estar sempre aprumado"
		{name: "Pajem", ch: classCase("Cavaleiro", "class.cavaleiro.pajem"),
			fixed: []expectedTerm{{target: expertise("Diplomacia"), amount: 2}}},
		// p54 "Suas armas fornecem +1 em rolagens de dano"
		{name: "Escudeiro", ch: classCase("Cavaleiro", "class.cavaleiro.escudeiro"),
			fixed: []expectedTerm{{target: damageAll, amount: 1}}},
		// p54 "Você recebe redução de dano 5 se estiver caído, desprevenido ou flanqueado."
		{name: "Desprezar os Covardes", ch: classCase("Cavaleiro", "class.cavaleiro.desprezar-os-covardes"),
			switched: []expectedTerm{{target: ModifierTarget{K: "damageReduction"}, amount: 5}}},
		// p62 "Você recebe +2 em Reflexos."
		{name: "Liberdade da Pradaria", ch: classCase("Druida", "class.druida.liberdade-da-pradaria"),
			fixed: []expectedTerm{{target: expertise("Reflexos"), amount: 2}}},
		// p61 "recebe redução de frio 5"
		{name: "Aspecto do Inverno", ch: classCase("Druida", "class.druida.aspecto-do-inverno"),
			fixed: []expectedTerm{{target: ModifierTarget{K: "damageReduction", DamageType: "frio"}, amount: 5}}},
		// p74 "Você recebe +2 em Atletismo."
		{name: "Gatuno", ch: classCase("Ladino", "class.ladino.gatuno"),
			fixed: []expectedTerm{{target: expertise("Atletismo"), amount: 2}}},
		// p74 "Você recebe +2 em Furtividade"
		{name: "Sombra", ch: classCase("Ladino", "class.ladino.sombra"),
			fixed: []expectedTerm{{target: expertise("Furtividade"), amount: 2}}},
		// p74 "Você recebe +2 em testes de agarrar."
		{name: "Mão na Boca", ch: classCase("Ladino", "class.ladino.mao-na-boca"),
			fixed: []expectedTerm{{target: grapple, amount: 2}}},
		// p77 "+2 em testes de ataque para agarrar e derrubar"
		{name: "Lutador de Chão", ch: classCase("Lutador", "class.lutador.lutador-de-chao"),
			fixed: []expectedTerm{{target: grapple, amount: 2}, {target: trip, amount: 2}}},
		// p80 "Você recebe +2 em Diplomacia e Intimidação."
		{name: "Voz Poderosa", ch: classCase("Nobre", "class.nobre.voz-poderosa"),
			fixed: []expectedTerm{{target: expertise("Diplomacia"), amount: 2}, {target: expertise("Intimidação"), amount: 2}}},
		// p38 "A CD para resistir a suas magias aumenta em +1."
		{name: "Fortalecimento Arcano", ch: classCase("Arcanista", "class.arcanista.fortalecimento-arcano"),
			fixed: []expectedTerm{{target: ModifierTarget{K: "spellDC"}, amount: 1}}},
		// p83 "+5 em testes de Intuição para perceber blefes"
		{name: "Virtude Paladinesca: Castidade", ch: classCase("Paladino", "class.paladino.virtude-paladinesca-castidade"),
			switched: []expectedTerm{{target: expertise("Intuição"), amount: 5}}},
		// p38 "+5 em Enganação e Intimidação contra pessoas não treinadas…"
		{name: "Envolto em Mistério", ch: classCase("Arcanista", "class.arcanista.envolto-em-misterio"),
			switched: []expectedTerm{{target: expertise("Enganação"), amount: 5}, {target: expertise("Intimidação"), amount: 5}}},
		// p47 "Quando ataca uma criatura sob efeito de uma condição de medo, seu dano aumenta em um passo."
		{name: "Abusar dos Fracos", ch: classCase("Bucaneiro", "class.bucaneiro.abusar-dos-fracos"),
			switched: []expectedTerm{{target: ModifierTarget{K: "damageStep", Scope: "all"}, amount: 1}}},
		// p47 "+2 nesse teste de ataque e na rolagem de dano"
		{name: "Ataque Acrobático", ch: classCase("Bucaneiro", "class.bucaneiro.ataque-acrobatico"),
			switched: []expectedTerm{{target: attackAll, amount: 2}, {target: damageAll, amount: 2}}},
		// p48 "Nesses locais, você recebe +2 em testes de perícias de Carisma"
		{name: "Folião", ch: classCase("Bucaneiro", "class.bucaneiro.foliao"),
			switched: []expectedTerm{{target: ModifierTarget{K: "expertiseByAttribute", Attribute: "charisma"}, amount: 2}}},
		// p51 "+2 na Defesa e Reflexos e +1d8 nas rolagens de dano"
		{name: "Escaramuça", ch: classCase("Caçador", "class.cacador.escaramuca"),
			switched: []expectedTerm{
				{target: ModifierTarget{K: "defense"}, amount: 2}, {target: expertise("Reflexos"), amount: 2},
				{target: damageAll, dice: "1d8"}}},
		// p57 "+5 em testes de Diplomacia ou Intimidação ao lidar com devotos"
		{name: "Autoridade Eclesiástica", ch: classCase("Clérigo", "class.clerigo.autoridade-eclesiastica"),
			switched: []expectedTerm{{target: expertise("Diplomacia"), amount: 5}, {target: expertise("Intimidação"), amount: 5}}},
		// p73 "+5 em testes de Acrobacia para escapar […] e em testes para resistir a efeitos de movimento"
		{name: "Escapista", ch: classCase("Ladino", "class.ladino.escapista"),
			switched: []expectedTerm{{target: expertise("Acrobacia"), amount: 5}, {target: resistance, amount: 5}}},
		// p74 "+5 em testes de Investigação para encontrar armadilhas e em testes de resistência contra elas"
		{name: "Saqueador de Tumbas", ch: classCase("Ladino", "class.ladino.saqueador-de-tumbas"),
			switched: []expectedTerm{{target: expertise("Investigação"), amount: 5}, {target: resistance, amount: 5}}},
		// p80 "sofre uma penalidade de –5, em vez de –10" — o +5 que desfaz metade.
		{name: "Língua Rápida", ch: classCase("Nobre", "class.nobre.lingua-rapida"),
			switched: []expectedTerm{{target: expertise("Diplomacia"), amount: 5}}},
		// p66 "+2 em testes de ataque e rolagens de dano contra oponentes caídos…"
		{name: "Valentão", ch: classCase("Guerreiro", "class.guerreiro.valentao"),
			switched: []expectedTerm{{target: attackAll, amount: 2}, {target: damageAll, amount: 2}}},
		// p20 "+2 em testes de Percepção e Sobrevivência realizados no subterrâneo"
		{name: "Conhecimento das Rochas", ch: raceCase("anao"),
			switched: []expectedTerm{{target: expertise("Percepção"), amount: 2}, {target: expertise("Sobrevivência"), amount: 2}}},
		// p23 "Se usar a ferramenta necessária, recebe +2 no teste de perícia."
		{name: "Engenhoso", ch: raceCase("goblin"),
			switched: []expectedTerm{{target: ModifierTarget{K: "expertiseAll"}, amount: 2}}},
		// p24 "+5 em testes de resistência contra efeitos causados por lefeu e pela Tormenta"
		{name: "Cria da Tormenta", ch: raceCase("lefou"),
			switched: []expectedTerm{{target: resistance, amount: 5}}},
		// p28 "Você recebe resistência a veneno +5"
		{name: "Natureza Venenosa", ch: raceCase("medusa"),
			switched: []expectedTerm{{target: resistance, amount: 5}}},
		// p31 "se estiver sem armadura ou roupas pesadas, +5 em Furtividade"
		{name: "Reptiliano", ch: raceCase("trog"),
			switched: []expectedTerm{{target: expertise("Furtividade"), amount: 5}}},
	}

	for _, c := range cases {
		effects := ComputeItemEffects(world.ActiveItemsFor(c.ch))
		for _, want := range c.fixed {
			if got := StatFor(effects, want.target).Total; got != want.amount {
				t.Errorf("%s: %s soma %+d, e o livro dá %+d", c.name, targetKey(want.target), got, want.amount)
			}
		}
		// O SITUACIONAL é oferecido e não soma desligado. Contado por alvo, para
		// "+5 contra medo" e "+5 contra mental" serem duas ofertas e não uma.
		offered := map[string]int{}
		for _, ce := range effects.Conditional {
			if ce.Source == c.name || ce.Source == "Raça: "+raceNameOf(c.ch) {
				offered[targetKey(ce.Target)+"|"+ce.Dice+"|"+strconv.Itoa(ce.Amount)]++
			}
		}
		for _, want := range c.switched {
			key := targetKey(want.target) + "|" + want.dice + "|" + strconv.Itoa(want.amount)
			if offered[key] == 0 {
				t.Errorf("%s: o interruptor de %s (%+d %s) não é oferecido", c.name,
					targetKey(want.target), want.amount, want.dice)
				continue
			}
			offered[key]--
		}
	}
}

func raceNameOf(ch Character) string {
	names := map[string]string{"anao": "Anão", "goblin": "Goblin", "lefou": "Lefou", "medusa": "Medusa", "trog": "Trog"}
	if len(ch.Races) == 0 {
		return ""
	}
	return names[ch.Races[0].Race]
}
