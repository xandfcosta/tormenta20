package sheet

import (
	"fmt"
	"testing"

	"t20engine/domain/book"
)

// O DADO DA CONJURAÇÃO: as parcelas da magia, trocadas pelo aprimoramento que muda e somadas, por repetição,
// pelo que aumenta (ver GLOSSARY, **dado da magia**).
func TestTheCastRollFollowsTheAugments(t *testing.T) {
	caster := CasterOf(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 20}, {ClassName: "Clérigo", Level: 20}}})
	for _, c := range []struct {
		why, spell string
		picks      []AugmentPick
		want       string
	}{
		{"a Bola de Fogo com dois aumentos (p182)", "bola-de-fogo", []AugmentPick{{AugmentIndex: 0, Stacks: 2}}, "[10d6 fogo]"},
		{"a esfera flamejante troca o dado", "bola-de-fogo", []AugmentPick{{AugmentIndex: 1, Stacks: 1}}, "[3d6 fogo]"},
		{"o aumento de luz da Coluna de Chamas", "coluna-de-chamas", []AugmentPick{{AugmentIndex: 1, Stacks: 1}}, "[6d6 fogo 7d6 luz]"},
		{"o +2d6 sem tipo da Lança Ígnea fica à parte", "lanca-ignea-de-aleph", []AugmentPick{{AugmentIndex: 0, Stacks: 1}}, "[4d6 fogo 4d6 perfuração 2d6 ]"},
		{"a Curar Ferimentos com dois aumentos", "curar-ferimentos", []AugmentPick{{AugmentIndex: 1, Stacks: 2}}, "[4d8+4 cura]"},
		// O aumento do Raio Solar soma à luz e à cura, e a cura só existe com o aprimoramento 2 (p203).
		{"o aumento de cura sem a parcela de cura não cria uma", "raio-solar", []AugmentPick{{AugmentIndex: 1, Stacks: 1}}, "[5d8 luz]"},
		{"com o aprimoramento 2, o aumento soma às duas", "raio-solar", []AugmentPick{{AugmentIndex: 1, Stacks: 1}, {AugmentIndex: 2, Stacks: 1}}, "[5d8 cura 5d8 luz]"},
	} {
		roll, err := caster.Spell(c.spell).CastRoll(c.picks)
		if err != nil || roll.ByText {
			t.Errorf("%s: erro %v, pelo texto %v", c.why, err, roll.ByText)
		}
		if got := rollsWritten(roll.Parts); got != c.want {
			t.Errorf("%s: %s; want %s", c.why, got, c.want)
		}
	}
	// O toque à distância do Toque Vampírico muda o dado para depois: a conjuração não o rola.
	if roll, err := caster.Spell("toque-vampirico").CastRoll([]AugmentPick{{AugmentIndex: 2, Stacks: 1}}); err != nil || !roll.ByText || len(roll.Parts) != 0 {
		t.Errorf("o aprimoramento 2 do Toque Vampírico rolou %s (%v); want o dado pelo texto", rollsWritten(roll.Parts), err)
	}
}

// TestEveryAugmentThatAddsToTheRollComposes passa por toda magia que rola na conjuração e todo aprimoramento que
// soma, até três repetições: a soma não pode juntar faces diferentes numa parcela.
func TestEveryAugmentThatAddsToTheRollComposes(t *testing.T) {
	caster := CasterOf(CharacterDTO{})
	composed := 0
	for _, spell := range book.Catalogs().Spells {
		for i, a := range spell.Augments {
			if len(a.Adds) == 0 {
				continue
			}
			for stacks := 1; stacks <= 3; stacks++ {
				if _, err := caster.Spell(spell.ID).CastRoll([]AugmentPick{{AugmentIndex: i, Stacks: stacks}}); err != nil {
					t.Errorf("%s aprimoramento %d ×%d: %v", spell.Name, i, stacks, err)
				}
				composed++
			}
		}
	}
	if composed == 0 {
		t.Fatal("nenhum aprimoramento soma ao dado: a varredura não mediu nada")
	}
}

func rollsWritten(parts []book.SpellRoll) string {
	out := "["
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%s %s", p.Dice, p.Kind)
	}
	return out + "]"
}
