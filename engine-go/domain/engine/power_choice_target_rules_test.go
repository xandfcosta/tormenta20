package engine

import (
	"path/filepath"
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
	world := BookRuleset(primeFromDump(t, filepath.Clean(
		filepath.Join(mustWd(t), "..", "..", "parity"))))

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
