package sheetui

import (
	"testing"

	"t20engine/domain/book"
)

// TestEveryPowerAugmentHasASignal cobra a tabela `powerAugmentSignals` contra o catálogo: todo poder que é
// aprimoramento (p131) tem um sinal, e dois poderes não dividem um. Sem sinal, a linha da folha desenha um
// gesto que escreve em `$` — o navegador não acusa nada, e ligar o poder não faz coisa nenhuma.
func TestEveryPowerAugmentHasASignal(t *testing.T) {
	augments := book.PowerAugments()
	if len(augments) == 0 {
		t.Fatal("nenhum poder é aprimoramento no catálogo: não há o que conferir, e verde aqui não valeria nada")
	}
	for _, power := range augments {
		if powerAugmentSignalOf(power.PowerID) == "" {
			t.Errorf("%s (%s) é aprimoramento e não tem sinal: acrescente uma linha em `powerAugmentSignals`",
				power.Name, power.PowerID)
		}
	}
	seen := map[string]string{}
	for _, s := range powerAugmentSignals {
		if other, taken := seen[s.name]; taken {
			t.Errorf("o sinal %s é de %s e de %s", s.name, other, s.power)
		}
		seen[s.name] = s.power
	}
}
