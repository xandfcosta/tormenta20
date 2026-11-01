package api

import (
	"strings"
	"testing"
)

const barbarianIncrease = "escolha/class.barbaro.aumento-de-atributo/"

// p38: "apenas uma vez por patamar para um mesmo atributo". Os patamares são da p35, pelo
// nível do personagem: Iniciante até o 4º, Veterano até o 10º.
func TestTheAttributeIncreaseStopsAtOnePerTierForTheSameAttribute(t *testing.T) {
	f, id := barbaro(t, 4)
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("a primeira Força foi recusada: %q", refusal)
	}
	refusal := powerCommand(t, f, id, barbarianIncrease+"strength", "")
	for _, want := range []string{"patamar", "Força", "p38"} {
		if !strings.Contains(refusal, want) {
			t.Errorf("a segunda Força no 4º nível devia citar %q, e a recusa foi %q", want, refusal)
		}
	}
	// Outro atributo no mesmo patamar não tem nada a ver com isso.
	if refusal := powerCommand(t, f, id, barbarianIncrease+"wisdom", ""); refusal != "" {
		t.Errorf("Sabedoria foi recusada: %q", refusal)
	}
}

func TestEachTierAllowsOneMoreOfTheSameAttribute(t *testing.T) {
	f, id := barbaro(t, 5)
	for i := 1; i <= 2; i++ {
		if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
			t.Fatalf("a %dª Força no 5º nível (Iniciante e Veterano) foi recusada: %q", i, refusal)
		}
	}
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); !strings.Contains(refusal, "patamar") {
		t.Errorf("a 3ª Força no 5º nível devia ser recusada, e a recusa foi %q", refusal)
	}
}

// O patamar é do personagem, então a conta atravessa as classes: o Aumento do bárbaro e o do
// guerreiro são o mesmo poder do livro.
func TestTheTierLimitCountsEveryClass(t *testing.T) {
	f, id := barbaro(t, 3)
	if refusal := enterClass(t, f, id, "Guerreiro"); refusal != "" {
		t.Fatalf("entrar: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
		t.Fatalf("a Força do bárbaro foi recusada: %q", refusal)
	}
	second := "escolha/class.guerreiro.aumento-de-atributo/strength"
	if refusal := powerCommand(t, f, id, second, ""); !strings.Contains(refusal, "patamar") {
		t.Errorf("a Força do guerreiro no mesmo patamar devia ser recusada, e a recusa foi %q", refusal)
	}
}

// Descer de patamar com a escolha a mais é recusado, como as outras escolhas contadas por nível.
func TestLoweringOutOfATierKeepsTheLimit(t *testing.T) {
	f, id := barbaro(t, 5)
	for i := 0; i < 2; i++ {
		if refusal := powerCommand(t, f, id, barbarianIncrease+"strength", ""); refusal != "" {
			t.Fatalf("Força: %q", refusal)
		}
	}
	refusal := stepClass(t, f, id, "Bárbaro", -1)
	if !strings.Contains(refusal, "patamar") {
		t.Errorf("descer para o 4º nível com duas Forças devia ser recusado, e a recusa foi %q", refusal)
	}
}
