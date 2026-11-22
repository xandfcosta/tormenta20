package api

import (
	"context"
	"strings"
	"testing"
)

// A TELA E O GESTO decidem pelo MESMO portão (ALE-679), e as flags que uma habilidade exige vêm de uma fonte só:
// os condicionais ligados, as mesmas flags que o motor aplica nos números (decisão do dono). Antes a tela lia os
// condicionais e o gesto as posturas pagas: com a Fúria acesa só no condicional, a tela oferecia o Frenesi (p43)
// e o gesto o recusava por "requer furia".
func TestTheScreenAndTheGestureReadTheSameFlags(t *testing.T) {
	f, id := barbarianWithMpSpent(t, 6, 0)
	seedPowerChoices(t, f.s, id, `["class.barbaro.frenesi"]`, "{}")
	if _, err := f.s.db.ExecContext(context.Background(),
		"INSERT INTO character_conditionals (characterId, conditionalId) VALUES (?, 'flag:furia')", id); err != nil {
		t.Fatalf("acender a Fúria no condicional: %v", err)
	}
	if screen := sheetTab(t, f, id, "abilities"); strings.Contains(screen, "requer furia") {
		t.Fatal("a tela recusou o Frenesi com a Fúria acesa; o teste precisa dela oferecendo")
	}
	if refusal := powerCommand(t, f, id, "usa/class.barbaro.frenesi", "{}"); refusal != "" {
		t.Errorf("a tela ofereceu o Frenesi e o gesto o recusou: %q", refusal)
	}
}
