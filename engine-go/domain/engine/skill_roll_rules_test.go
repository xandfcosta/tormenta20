package engine

import (
	"strings"
	"testing"
)

// O TESTE, que é a mecânica BASE do livro (p220-221).
//
//	"Um teste é uma rolagem de 1d20 + um modificador. Você passa no teste se este
//	 resultado for igual ou maior que a CD." (p220)
//
//	"Ao fazer um teste, um 20 natural (quando o resultado do d20 é 20) sempre é
//	 um sucesso, e um 1 natural (quando o resultado do d20 é 1) sempre é uma
//	 falha, não importando o valor a ser alcançado." (p221)
//
// A SOMA NÃO É O QUE SE PRENDE AQUI — `d20 + modificador` é aritmética, e um
// caso que a afirmasse estaria testando o compilador. O que esta regra carrega é
// o que a página diz ALÉM da soma: o d20 é um dado de 1 a 20, e os dois extremos
// dele não dependem de CD nenhuma.
//
// O NATURAL É DO DADO e não do total, e é essa a distinção inteira: um +19 que
// rolou 1 falha, e um −3 que rolou 20 passa.

// O D20 RECEBIDO É CONFERIDO, e é a mesma linha do `d20Of` do ataque: um número
// fora de 1..20 não é um dado, é um pedido montado à mão — e esta fatia ACEITA o
// valor que a mesa digitou, então a porta é mais larga do que a do ataque.
func TestADieOutsideOneToTwentyIsNotADie(t *testing.T) {
	for _, fora := range []int{0, 21, -1, 100} {
		if _, err := ResolveSkillTest(7, fora); err == nil {
			t.Errorf("o d20 %d foi aceito, e um d20 vai de 1 a 20", fora)
		} else if !strings.Contains(err.Error(), "d20") {
			t.Errorf("a recusa do %d não falou do d20: %v", fora, err)
		}
	}
	// O CONTROLE: as duas pontas VÁLIDAS passam. Sem ele, uma função que
	// recusasse tudo passaria no laço acima.
	for _, dentro := range []int{1, 20} {
		if _, err := ResolveSkillTest(7, dentro); err != nil {
			t.Errorf("o d20 %d foi recusado: %v", dentro, err)
		}
	}
}

// O 20 NATURAL É DO DADO, e o caso que o prende é o do modificador NEGATIVO: um
// total de 17 com um d20 20 é natural; um total de 20 com um d20 13 não é.
func TestTheNaturalTwentyIsAboutTheDieAndNotTheTotal(t *testing.T) {
	natural, err := ResolveSkillTest(-3, 20)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !natural.Natural20 {
		t.Errorf("um d20 20 com modificador −3 não foi 20 natural")
	}
	if natural.Total != 17 {
		t.Errorf("o total veio %d, e 20 − 3 é 17", natural.Total)
	}

	// O CONTROLE, e ele é a metade que uma leitura do TOTAL perderia: um total
	// de 20 que saiu de um d20 13 não é natural nenhum.
	alto, err := ResolveSkillTest(7, 13)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if alto.Natural20 {
		t.Errorf("um total 20 vindo de um d20 13 foi lido como 20 natural")
	}
}

// E O 1 NATURAL pelo mesmo caminho: um herói de +19 que rola 1 falha.
func TestTheNaturalOneIsAboutTheDieAndNotTheTotal(t *testing.T) {
	natural, err := ResolveSkillTest(19, 1)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !natural.Natural1 {
		t.Errorf("um d20 1 com modificador +19 não foi 1 natural")
	}
	if natural.Total != 20 {
		t.Errorf("o total veio %d, e 1 + 19 é 20 — o natural não apaga a soma", natural.Total)
	}
	if natural.Natural20 {
		t.Errorf("o mesmo teste se disse 20 natural E 1 natural")
	}
}

// E O TESTE COMUM não é natural de nada. Sem este caso, uma função que marcasse
// os dois sempre passaria nos dois acima.
func TestAnOrdinaryRollIsNaturalOfNothing(t *testing.T) {
	comum, err := ResolveSkillTest(7, 14)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if comum.Natural20 || comum.Natural1 {
		t.Errorf("um d20 14 veio marcado como natural: %+v", comum)
	}
	if comum.Total != 21 || comum.Roll != 14 || comum.Modifier != 7 {
		t.Errorf("a conta do teste comum veio %+v, e 14 + 7 é 21", comum)
	}
}
