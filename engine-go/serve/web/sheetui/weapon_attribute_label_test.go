package sheetui

import (
	"testing"

	"t20engine/domain/engine"
)

// A LICENÇA DO ATRIBUTO DA ARMA se lê como frase inteira, na lista de efeitos: o Armeiro troca no ataque E no dano
// (p68), e por isso tem duas linhas; a Balística só no ataque (p69).
func TestTheWeaponAttributeLicenseReadsAsAPhrase(t *testing.T) {
	for _, c := range []struct {
		target engine.ModifierTarget
		want   string
	}{
		{engine.ModifierTarget{K: "weaponAttribute", Name: "melee", Scope: "attack", Attribute: "intelligence"}, "Inteligência no ataque corpo a corpo"},
		{engine.ModifierTarget{K: "weaponAttribute", Name: "melee", Scope: "damage", Attribute: "intelligence"}, "Inteligência no dano corpo a corpo"},
		{engine.ModifierTarget{K: "weaponAttribute", Name: "ranged", Scope: "attack", Attribute: "intelligence"}, "Inteligência no ataque à distância"},
	} {
		if got := targetLabel(c.target); got != c.want {
			t.Errorf("a licença %+v diz %q; want %q", c.target, got, c.want)
		}
	}
}
