package api

import (
	"testing"
)

// A ARMA MÁGICA é de "1 arma empunhada", e não sobe a arma natural nem o desarmado.

// "Alvo: 1 arma empunhada". A arma natural e o desarmado não são empunhados.
func TestMagicWeaponOnlyRaisesTheWieldedWeapon(t *testing.T) {
	f, id, _ := minotaurOnTurn(t)
	before := attackByWeapon(t, f, id)
	if refusal := sheetPost(t, f, id, "/efeitos/aplica/arma-magica", ""); refusal != "" {
		t.Fatalf("aplicar a Arma Mágica: %q", refusal)
	}
	after := attackByWeapon(t, f, id)
	for name, attack := range before {
		want := attack
		if name != "Chifres" && name != "Ataque desarmado" {
			want = attack + 1
		}
		if after[name] != want {
			t.Errorf("%s foi de %+d a %+d, e devia ficar em %+d", name, attack, after[name], want)
		}
	}
	if len(before) < 3 {
		t.Fatalf("a bancada pede arma de mão, desarmado e chifres, e tem %v", before)
	}
}
