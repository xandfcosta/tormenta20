package book

import "testing"

// A PARCELA DO DADO DA MAGIA diz o que ela é (ver GLOSSARY, **dado da magia**): a cura e os PV temporários não são
// dano, e todo o resto é, inclusive a parcela que o livro não diz o tipo (a Lança Ígnea de Aleph, "aumenta o dano").
func TestTheSpellRollSaysWhatItIs(t *testing.T) {
	for _, c := range []struct {
		kind            string
		damage, healing bool
	}{
		{"fogo", true, false},
		{"corte e fogo, metade de cada", true, false},
		{"", true, false},
		{"cura", false, true},
		{"PV temporários", false, false},
	} {
		part := SpellRoll{Dice: "2d6", Kind: c.kind}
		if part.IsDamage() != c.damage || part.IsHealing() != c.healing {
			t.Errorf("a parcela de %q diz dano=%v cura=%v; want dano=%v cura=%v",
				c.kind, part.IsDamage(), part.IsHealing(), c.damage, c.healing)
		}
	}
}
