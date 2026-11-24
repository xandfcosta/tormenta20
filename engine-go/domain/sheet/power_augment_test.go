package sheet

import (
	"strings"
	"testing"

	"t20engine/domain/book"
)

// OS PODERES QUE SÃO APRIMORAMENTO (p131) moram no conjurador: a magia na ficha lista os que a ficha TEM, cada
// um com o cadeado da magia, e o preço da conjuração soma os dois tipos de aprimoramento numa resposta só.
func TestTheSpellListsThePowerAugmentsTheSheetHas(t *testing.T) {
	arcanist := CharacterDTO{
		Classes:     []ClassDTO{{ClassName: "Arcanista", Level: 8}},
		ClassPowers: `["magia-acelerada","magia-discreta"]`,
	}
	// A Queda Suave é reação: a Acelerada só vale em movimento, padrão ou completa (p131).
	listed := CasterOf(arcanist).Spell("queda-suave").PowerAugments
	var names []string
	for _, p := range listed {
		names = append(names, p.Power.Name)
	}
	if strings.Join(names, ", ") != "Magia Acelerada, Magia Discreta" {
		t.Fatalf("a magia lista %v; want os dois poderes que a ficha tem, sem a Ampliada", names)
	}
	if !strings.Contains(listed[0].Lock, "execução de movimento, padrão ou completa") {
		t.Errorf("a Acelerada numa reação tranca com %q", listed[0].Lock)
	}
	if listed[1].Lock != "" {
		t.Errorf("a Discreta tranca com %q", listed[1].Lock)
	}
}

func TestThePriceSumsTheSpellAndThePowerAugments(t *testing.T) {
	arcanist := CharacterDTO{
		Classes:     []ClassDTO{{ClassName: "Arcanista", Level: 8}},
		ClassPowers: `["magia-acelerada","magia-discreta"]`,
	}
	light := CasterOf(arcanist).Spell("luz")
	for _, c := range []struct {
		why    string
		powers []PowerAugmentPick
		pm     int
		says   string
	}{
		{"a Discreta soma 2 (p131)", []PowerAugmentPick{{PowerID: "magia-discreta", Stacks: 1}}, 2, ""},
		{"a Acelerada soma 4 (p131)", []PowerAugmentPick{{PowerID: book.QuickenedSpellPower, Stacks: 1}}, 4, ""},
		{"o poder que a ficha não tem", []PowerAugmentPick{{PowerID: "magia-ampliada", Stacks: 1}}, 0, "esta ficha não tem Magia Ampliada"},
		{"a mudança não se acumula (p171)", []PowerAugmentPick{{PowerID: "magia-discreta", Stacks: 2}}, 0, "não se acumula"},
	} {
		pm, refusal := light.AugmentPrice(nil, c.powers)
		if pm != c.pm || !strings.Contains(refusal, c.says) || (c.says == "" && refusal != "") {
			t.Errorf("%s: %d PM, recusa %q; want %d PM e %q", c.why, pm, refusal, c.pm, c.says)
		}
	}

	// O RITUAL muda a execução, como a Acelerada, e mudança não se acumula (p171).
	ritual := arcanist
	ritual.Conditionals = []string{RitualSwitch}
	_, refusal := CasterOf(ritual).Spell("luz").AugmentPrice(nil, []PowerAugmentPick{{PowerID: book.QuickenedSpellPower, Stacks: 1}})
	if !strings.Contains(refusal, "o ritual e Magia Acelerada mudam a execução") {
		t.Errorf("a Acelerada no ritual é recusada com %q", refusal)
	}
}
