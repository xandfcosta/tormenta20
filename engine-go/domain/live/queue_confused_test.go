package live

import (
	"strings"
	"testing"
)

// CONFUSO (p394): "Role 1d6 no início de seus turnos", e o resultado diz o que
// a criatura faz. A fila pede o dado, e a faixa escreve o resultado; quem
// aplica é o mestre.
func TestTheConfusedRollsTheD6AtTheStartOfTheTurn(t *testing.T) {
	due := DueRolls(InitiativeEntry{Label: "Goblin"}, []string{ConditionConfused})
	if len(due) != 1 || due[0].Condition != "Confuso" || due[0].At != "inicio" ||
		len(due[0].Parts) != 1 || due[0].Parts[0].Dice != "1d6" || due[0].Parts[0].Kind != "" {
		t.Fatalf("o confuso tem %+v na vez; want 1d6 sem tipo no início do turno", due)
	}
}

// A TABELA DO CONFUSO, resultado a resultado (p394).
func TestTheConfusedTableSaysWhatEachResultDoes(t *testing.T) {
	for d6, want := range map[int]string{
		1: "1d8", 2: "não age", 3: "não age", 4: "ataca a criatura mais próxima",
		5: "ataca a criatura mais próxima", 6: "a condição termina",
	} {
		if got := ConditionRollNote("Confuso", d6); !strings.Contains(got, want) {
			t.Errorf("confuso com %d no 1d6 diz %q; want algo com %q", d6, got, want)
		}
	}
	if got := ConditionRollNote("Em Chamas", 3); got != "" {
		t.Errorf("as chamas não têm tabela, e a nota veio %q", got)
	}
}
