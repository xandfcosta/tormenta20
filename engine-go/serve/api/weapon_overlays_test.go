package api

import (
	"context"
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A MELHORIA, O MATERIAL E O ENCANTO SÃO DA ARMA EM QUE ESTÃO: "a arma fornece +1 nas rolagens de dano"
// (Cruel, p165), "a margem de ameaça [desta arma] aumenta em 1" (Precisa, p166). Nada disso sobe a
// outra mão nem o ataque desarmado.

func cardsByWeapon(t *testing.T, f sceneFixture, id int64) map[string]engine.WeaponCard {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	out := map[string]engine.WeaponCard{}
	for _, card := range dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals)) {
		out[card.Name] = card
	}
	return out
}

func swordAndDagger(t *testing.T) (sceneFixture, int64, int64) {
	t.Helper()
	f, id := fighterFixture(t)
	saveHand(t, f, id, `["armas-simples","armas-marciais"]`)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, id, "adaga", "Adaga", "wielded2")
	return f, id, sword
}

func TestAnOverlayStaysOnItsWeapon(t *testing.T) {
	type number func(engine.WeaponCard) int
	damage := func(c engine.WeaponCard) int { return c.DamageAll.Total + c.DamageOwn.Total }
	margin := func(c engine.WeaponCard) int { return c.CritRange }
	mult := func(c engine.WeaponCard) int { return c.CritMult }
	cases := []struct {
		name, overlay string
		enchant       bool
		read          number
		moves         int
	}{
		{"Cruel no dano", "melhoria-cruel", false, damage, 1},
		{"Atroz no dano", "melhoria-atroz", false, damage, 2},
		{"Precisa na margem", "melhoria-precisa", false, margin, -1},
		{"Maciça no multiplicador", "melhoria-macica", false, mult, 1},
		{"Mitral na margem", "material-mitral", false, margin, -1},
		{"Formidável no dano", "encanto-formidavel", true, damage, 2},
		{"Ameaçadora na margem", "encanto-ameacadora", true, margin, -2}, // "espada longa ameaçadora tem margem 17" (p335)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, id, sword := swordAndDagger(t)
			before := cardsByWeapon(t, f, id)
			switch {
			case c.enchant:
				if _, err := f.s.db.ExecContext(context.Background(), "INSERT INTO character_item_enchants (itemId, enchantId) VALUES (?, ?)", sword, c.overlay); err != nil {
					t.Fatal(err)
				}
			case c.overlay == "material-mitral":
				if _, err := f.s.db.ExecContext(context.Background(), "UPDATE character_items SET material = ? WHERE id = ?", c.overlay, sword); err != nil {
					t.Fatal(err)
				}
			default:
				if _, err := f.s.db.ExecContext(context.Background(), "UPDATE character_items SET improvements = ? WHERE id = ?", `["`+c.overlay+`"]`, sword); err != nil {
					t.Fatal(err)
				}
			}
			after := cardsByWeapon(t, f, id)
			if got, was := c.read(after["Espada longa"]), c.read(before["Espada longa"]); got != was+c.moves {
				t.Errorf("a espada foi de %d a %d, e devia andar %d", was, got, c.moves)
			}
			for _, other := range []string{"Adaga", "Ataque desarmado"} {
				if got, was := c.read(after[other]), c.read(before[other]); got != was {
					t.Errorf("%s foi de %d a %d, e a sobreposição está na espada", other, was, got)
				}
			}
		})
	}
}

// O dado de energia do encanto (Flamejante: +1d6 de fogo) é da arma encantada.
func TestAnEnergyDieStaysOnItsWeapon(t *testing.T) {
	f, id, sword := swordAndDagger(t)
	if _, err := f.s.db.ExecContext(context.Background(), "INSERT INTO character_item_enchants (itemId, enchantId) VALUES (?, 'encanto-flamejante')", sword); err != nil {
		t.Fatal(err)
	}
	cards := cardsByWeapon(t, f, id)
	if len(cards["Espada longa"].ExtraDamage) != 1 {
		t.Errorf("a espada flamejante devia ter o dado de fogo, e tem %v", cards["Espada longa"].ExtraDamage)
	}
	for _, other := range []string{"Adaga", "Ataque desarmado"} {
		if len(cards[other].ExtraDamage) != 0 {
			t.Errorf("%s ganhou o dado de fogo da espada: %v", other, cards[other].ExtraDamage)
		}
	}
}
