package sheetui

import (
	"strings"
	"testing"

	"t20engine/domain/book"
)

// A FOLHA ACOMPANHA OS APRIMORAMENTOS: a linha do dado soma o aumento ligado, o aprimoramento que troca o dado
// troca a linha, e o aumento sem tipo ganha a linha dele — com o campo da mão de cada parcela no sinal da posição
// que o servidor confere.
func TestTheRollLinesFollowTheAugments(t *testing.T) {
	fireball, _ := book.SpellByID("bola-de-fogo")
	lines := spellRollLinesOf(fireball)
	base, sphere := lines[0], lines[1]
	if !strings.Contains(base.Text, "6 + 2 * $augment0") || base.Signal != "spell_hand_1" {
		t.Errorf("a linha da Bola de Fogo não soma o aumento: %+v", base)
	}
	if !strings.Contains(sphere.Static, "3d6 de fogo") || !strings.Contains(sphere.Show, "$augment1 > 0") || sphere.Signal != "spell_hand_1" {
		t.Errorf("a esfera flamejante não troca a linha: %+v", sphere)
	}

	lance, _ := book.SpellByID("lanca-ignea-de-aleph")
	lines = spellRollLinesOf(lance)
	if len(lines) != 3 || lines[2].Signal != "spell_hand_3" || !strings.Contains(lines[2].Show, "$augment0 > 0") {
		t.Errorf("a Lança Ígnea não tem a linha dos +2d6 sem tipo com o terceiro campo: %+v", lines)
	}

	vampire, _ := book.SpellByID("toque-vampirico")
	byText := false
	for _, line := range spellRollLinesOf(vampire) {
		byText = byText || (line.ByText && strings.Contains(line.Show, "$augment2 > 0"))
	}
	if !byText {
		t.Error("o aprimoramento 2 do Toque Vampírico não diz que o dado sai pelo texto")
	}
}

// TestEverySpellHasAtMostOneUntypedAdd prende o que as linhas da folha supõem: a parcela do aumento sem tipo vai
// depois das parcelas do dado, na posição seguinte, e duas dessas numa magia disputariam o mesmo campo.
func TestEverySpellHasAtMostOneUntypedAdd(t *testing.T) {
	for _, spell := range book.Catalogs().Spells {
		untyped := 0
		for _, a := range spell.Augments {
			for _, add := range a.Adds {
				if add.Kind == "" {
					untyped++
				}
			}
		}
		if untyped > 1 {
			t.Errorf("%s tem %d aumentos sem tipo, e a folha dá um campo só para eles", spell.Name, untyped)
		}
	}
}
