package sheet

import (
	"reflect"
	"strings"
	"testing"

	"t20engine/domain/book"
)

// OS PODERES QUE SÃO APRIMORAMENTO agem no dado da conjuração (ver GLOSSARY, **dado da magia**): o Espírito dos
// Solstícios põe no máximo (p62), a Comunhão Vital divide a cura (p58), e a Magia Sagrada/Profana troca o tipo do
// dano pela energia do deus — e quem segue deus de energia qualquer diz qual (p58).
func TestThePowerAugmentsActOnTheCastRoll(t *testing.T) {
	holy := func(energy string) []PowerAugmentPick {
		return []PowerAugmentPick{{PowerID: book.HolySpellPower, Stacks: 1, Energy: energy}}
	}
	column := []book.SpellRoll{{Dice: "6d6", Kind: "fogo"}, {Dice: "6d6", Kind: "luz"}}
	for _, c := range []struct {
		why, god string
		picks    []PowerAugmentPick
		want     []book.SpellRoll
	}{
		{"o deus de energia positiva canaliza luz", "positiva", holy(""),
			[]book.SpellRoll{{Dice: "6d6", Kind: "luz"}, {Dice: "6d6", Kind: "luz"}}},
		{"o deus de energia negativa canaliza trevas", "negativa", holy(""),
			[]book.SpellRoll{{Dice: "6d6", Kind: "trevas"}, {Dice: "6d6", Kind: "trevas"}}},
		{"o deus de energia qualquer, com trevas escolhida", "qualquer", holy("trevas"),
			[]book.SpellRoll{{Dice: "6d6", Kind: "trevas"}, {Dice: "6d6", Kind: "trevas"}}},
		{"sem a Magia Sagrada, o tipo fica", "positiva", nil, column},
	} {
		powered, err := PoweredCastOf(c.picks, c.god)
		if err != nil {
			t.Errorf("%s: %v", c.why, err)
			continue
		}
		if got := powered.Apply(column); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v; want %v", c.why, got, c.want)
		}
	}
	healing := []book.SpellRoll{{Dice: "2d8+2", Kind: "cura"}}
	if powered, _ := PoweredCastOf(holy(""), "positiva"); !reflect.DeepEqual(powered.Apply(healing), healing) {
		t.Error("a Magia Sagrada trocou o tipo da cura, e ela muda o do dano")
	}
	if _, err := PoweredCastOf(holy(""), "qualquer"); err == nil || !strings.Contains(err.Error(), "luz ou trevas") {
		t.Errorf("a Magia Sagrada do deus de energia qualquer, sem escolha, devolveu %v; want a recusa que pede luz ou trevas", err)
	}

	solstice, err := PoweredCastOf([]PowerAugmentPick{{PowerID: book.SolsticeSpiritPower, Stacks: 1}}, "")
	if err != nil || !solstice.Maximize {
		t.Errorf("o Espírito dos Solstícios não maximiza: %+v (%v)", solstice, err)
	}
	communion, err := PoweredCastOf([]PowerAugmentPick{{PowerID: book.VitalCommunionPower, Stacks: 1}}, "")
	if err != nil || !communion.VitalShare {
		t.Errorf("a Comunhão Vital não divide a cura: %+v (%v)", communion, err)
	}
}
