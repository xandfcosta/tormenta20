package sheetui

import (
	"testing"

	"t20engine/domain/engine"
)

// AS LINHAS DO DANO DESARMADO NA CARTA (ALE-655). Abaixo do total do dano, uma linha por situação, com o dado
// um passo acima e o poder que a dá com a página: é a leitura de quem rola na mão. Nada para ligar: a linha
// que vale se escolhe no passo do dano da mesa.
func TestTheUnarmedCardReadsOneDamageLinePerSituation(t *testing.T) {
	card := engine.WeaponCard{Name: "Ataque desarmado", Skill: "Luta", Damage: "1d6", DamageBonus: 4, CritRange: 20, CritMult: 2,
		DamageVariants: []engine.DamageVariant{
			{ID: "class.lutador.arma-improvisada", Label: "com arma improvisada", Source: "Arma Improvisada", BookPage: 76, Damage: "1d8", Steps: 1},
			{ID: "class.lutador.chave", Label: "esmagando quem agarra", Source: "Chave", BookPage: 76, Damage: "1d8", Steps: 1},
		}}
	tile := weaponTiles([]engine.WeaponCard{card})[0]
	if len(tile.DamageLines) != 2 {
		t.Fatalf("a carta desarmada tem as linhas %+v; want a Arma Improvisada e a Chave", tile.DamageLines)
	}
	for i, want := range []struct{ label, note, value string }{
		{"com arma improvisada", "Arma Improvisada (p76)", "1d8+4"},
		{"esmagando quem agarra", "Chave (p76)", "1d8+4"},
	} {
		got := tile.DamageLines[i]
		if got.Label != want.label || got.Note != want.note || got.Value != want.value {
			t.Errorf("a linha %d diz %q · %q · %q; want %q · %q · %q", i, got.Label, got.Note, got.Value, want.label, want.note, want.value)
		}
	}
	if plain := weaponTiles([]engine.WeaponCard{{Name: "Ataque desarmado", Damage: "1d3"}})[0]; len(plain.DamageLines) != 0 {
		t.Errorf("a carta sem situação tem as linhas %+v", plain.DamageLines)
	}
}
