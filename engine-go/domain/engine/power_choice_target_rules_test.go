package engine

import (
	"testing"
)

// O AUMENTO DE ATRIBUTO SOMA NO ATRIBUTO QUE O JOGADOR ESCOLHEU (p38, ALE-491).
//
//	"Aumento de Atributo. Você recebe +1 em um atributo. Você pode escolher este
//	 poder várias vezes, mas apenas uma vez por patamar para um mesmo atributo."
//
// Toda classe tem o poder, e ele não somava nada: a entrada da classe
// (`class.guerreiro.aumento-de-atributo`) APONTA para o verbete genérico, o
// motor não seguia o ponteiro de poder de classe, e o verbete não tinha
// modificador porque o alvo dele é uma ESCOLHA — que mora em `powerChoices` e o
// motor não lia.
//
// # O que se prende
//
// O PREDICADO: o +1 cai no atributo escolhido e em nenhum outro. A REPETIÇÃO:
// duas escolhas são o poder tomado duas vezes, e as duas somam. E o poder sem
// escolha não soma nada — um +1 sem alvo não é +1 em lugar nenhum.
func TestTheAttributeIncreaseLandsOnTheChosenAttribute(t *testing.T) {
	world := BookRuleset(primeBook(t))

	const poder = "class.guerreiro.aumento-de-atributo"
	base := Character{
		Level: 5, Strength: 2, Dexterity: 1, Wisdom: 1,
		Classes:      []CharacterClass{{ClassName: "Guerreiro", Level: 5}},
		ClassPowers:  `["` + poder + `"]`,
		PowerChoices: `{}`,
	}
	atributos := func(ch Character) (forca, destreza, sabedoria int) {
		a := world.ComputeSheet(ch, nil).Attributes
		return a["strength"].Total, a["dexterity"].Total, a["wisdom"].Total
	}

	// O controle: o poder sem escolha não soma. Se somasse, o caso abaixo
	// mediria um +1 que não veio da escolha.
	if f, d, s := atributos(base); f != 2 || d != 1 || s != 1 {
		t.Fatalf("o controle já estava errado: com o poder e sem escolha, Força %d, "+
			"Destreza %d, Sabedoria %d — deviam ser 2, 1 e 1", f, d, s)
	}

	umaVez := base
	umaVez.PowerChoices = `{"` + poder + `":["strength"]}`
	if f, d, s := atributos(umaVez); f != 3 || d != 1 || s != 1 {
		t.Errorf("Aumento de Atributo em Força deu Força %d, Destreza %d, Sabedoria %d; "+
			"a p38 dá +1 no atributo escolhido: 3, 1 e 1", f, d, s)
	}

	duasVezes := base
	duasVezes.PowerChoices = `{"` + poder + `":["strength","wisdom"]}`
	if f, d, s := atributos(duasVezes); f != 3 || d != 1 || s != 2 {
		t.Errorf("o poder tomado duas vezes (Força e Sabedoria) deu Força %d, Destreza %d, "+
			"Sabedoria %d; cada escolha é um +1: 3, 1 e 2", f, d, s)
	}

	// O MESMO atributo em patamares diferentes é permitido, e as duas cópias
	// vêm da mesma fonte: a pilha não pode tratá-las como um bônus repetido.
	mesmoAtributo := base
	mesmoAtributo.PowerChoices = `{"` + poder + `":["strength","strength"]}`
	if f, _, _ := atributos(mesmoAtributo); f != 4 {
		t.Errorf("Força escolhida duas vezes deu Força %d; são dois +1 (p38, em patamares "+
			"diferentes), e o esperado é 4", f)
	}
}

// A ESPECIALIZAÇÃO EM ARMA SOMA +2 NO DANO DA ARMA ESCOLHIDA (p65, ALE-491).
//
//	"Especialização em Arma. Escolha uma arma. Você recebe +2 em rolagens de
//	 dano com essa arma. Você pode escolher este poder outras vezes para armas
//	 diferentes."
//
// O alvo é o ESCOPO do dano, e a escolha o preenche: `damage:weapon:<id>`. O
// predicado é a arma — a espada longa escolhida ganha +2, e a adaga na outra mão
// não ganha nada. E a decomposição da carta diz quem deu o +2: a parcela tem
// de aparecer nomeada, ou a soma das linhas não fecha no total.
func TestWeaponSpecializationAddsDamageOnlyToTheChosenWeapon(t *testing.T) {
	world := BookRuleset(primeBook(t))

	espada, adaga := "espada-longa", "adaga"
	maoPrincipal, outraMao := "wielded", "wielded2"
	const poder = "class.guerreiro.especializacao-em-arma"
	ch := Character{
		Level: 5, Strength: 3,
		Classes:      []CharacterClass{{ClassName: "Guerreiro", Level: 5}},
		ClassPowers:  `["` + poder + `"]`,
		PowerChoices: `{"` + poder + `":["espada-longa"]}`,
		Items: []CharacterItem{
			{CatalogID: &espada, Name: "Espada longa", Quantity: 1, Equipped: &maoPrincipal},
			{CatalogID: &adaga, Name: "Adaga", Quantity: 1, Equipped: &outraMao},
		},
	}
	bonus := map[string]WeaponCard{}
	for _, card := range world.ComputeWeaponCards(ch, nil) {
		bonus[card.Name] = card
	}
	if len(bonus) != 2 {
		t.Fatalf("esperava as cartas da espada e da adaga, e vieram %d", len(bonus))
	}

	// Força 3 no dano das duas; a espada escolhida soma +2.
	if got := bonus["Espada longa"].DamageBonus; got != 5 {
		t.Errorf("a espada longa escolhida na Especialização somou %d no dano; "+
			"Força 3 + 2 da p65 dão 5", got)
	}
	if got := bonus["Adaga"].DamageBonus; got != 3 {
		t.Errorf("a adaga, que não foi escolhida, somou %d no dano; só a Força 3 vale", got)
	}

	soma := bonus["Espada longa"].StrDamage
	for _, c := range append(bonus["Espada longa"].DamageAll.Contributions,
		bonus["Espada longa"].DamageOwn.Contributions...) {
		soma += c.Amount
	}
	if soma != bonus["Espada longa"].DamageBonus {
		t.Errorf("as parcelas do dano da espada somam %d e o total é %d: a decomposição "+
			"esconde uma parcela", soma, bonus["Espada longa"].DamageBonus)
	}
}

// O MESTRE EM ARMA SOBE UM PASSO NO DANO DA ARMA ESCOLHIDA (p66, ALE-491).
//
//	"Mestre em Arma. Escolha uma arma. Com esta arma, seu dano aumenta em um
//	 passo […]"
//
// A mesma forma da Especialização, no alvo `damageStep`. O predicado é a arma:
// a espada longa (1d8) passa a 1d10, e a adaga na outra mão fica em 1d4.
func TestWeaponMasteryStepsUpOnlyTheChosenWeapon(t *testing.T) {
	world := BookRuleset(primeBook(t))

	espada, adaga := "espada-longa", "adaga"
	maoPrincipal, outraMao := "wielded", "wielded2"
	const poder = "class.guerreiro.mestre-em-arma"
	ch := Character{
		Level: 12, Strength: 3,
		Classes:      []CharacterClass{{ClassName: "Guerreiro", Level: 12}},
		ClassPowers:  `["` + poder + `"]`,
		PowerChoices: `{"` + poder + `":["espada-longa"]}`,
		Items: []CharacterItem{
			{CatalogID: &espada, Name: "Espada longa", Quantity: 1, Equipped: &maoPrincipal},
			{CatalogID: &adaga, Name: "Adaga", Quantity: 1, Equipped: &outraMao},
		},
	}
	dado := map[string]string{}
	for _, card := range world.ComputeWeaponCards(ch, nil) {
		dado[card.Name] = card.Damage
	}
	if dado["Espada longa"] != "1d10" {
		t.Errorf("a espada longa do Mestre em Arma rola %q; um passo acima de 1d8 é 1d10 (p66)",
			dado["Espada longa"])
	}
	if dado["Adaga"] != "1d4" {
		t.Errorf("a adaga, que não foi escolhida, rola %q; ela fica em 1d4", dado["Adaga"])
	}
}
