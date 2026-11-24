package sheet

import (
	"slices"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// O CONJURADOR DIZ O QUE CADA APRIMORAMENTO APAGA, a mesma regra que o `AugmentPrice` recusa (p171): o
// exclusivo apaga todos os outros, os outros apagam o exclusivo, e a Magia Acelerada e o aprimoramento que
// muda a execução se apagam entre si.
func TestEveryAugmentSaysWhatItClears(t *testing.T) {
	arcanist := CharacterDTO{
		Classes:     []ClassDTO{{ClassName: "Arcanista", Level: 9}, {ClassName: "Druida", Level: 1}},
		ClassPowers: `["magia-acelerada","magia-discreta"]`,
	}
	caster := CasterOf(arcanist)
	quickened, discreet := PowerAugmentKey(book.QuickenedSpellPower), PowerAugmentKey("magia-discreta")

	// O Armamento da Natureza: o aprimoramento 1 "muda a execução para ação de movimento".
	nature := caster.Spell("armamento-da-natureza")
	if got := nature.Clears(SpellAugmentKey(1)); !slices.Contains(got, quickened) {
		t.Errorf("a mudança de execução não apaga a Acelerada: %v", got)
	}
	if got := nature.Clears(quickened); !slices.Contains(got, SpellAugmentKey(1)) || slices.Contains(got, discreet) {
		t.Errorf("a Acelerada apaga %v; want a mudança de execução, e não a Discreta", got)
	}

	// A Explosão de Chamas: o aprimoramento 0 é o truque, exclusivo (p171).
	flames := caster.Spell("explosao-de-chamas")
	cantrip := SpellAugmentKey(0)
	if got := flames.Clears(cantrip); !slices.Contains(got, discreet) || slices.Contains(got, cantrip) {
		t.Errorf("o truque apaga %v; want todos os outros, e não ele mesmo", got)
	}
	if got := flames.Clears(discreet); !slices.Equal(got, []AugmentKey{cantrip}) {
		t.Errorf("a Discreta apaga %v; want só o truque", got)
	}
}

// A EXECUÇÃO DA CONJURAÇÃO com as escolhas feitas: a da magia, a que o aprimoramento dá, a ação livre da
// Acelerada e a longa do ritual (p131). É ela que o turno cobra.
func TestTheExecutionFollowsTheChoices(t *testing.T) {
	arcanist := CharacterDTO{
		Classes:     []ClassDTO{{ClassName: "Arcanista", Level: 9}, {ClassName: "Druida", Level: 1}},
		ClassPowers: `["magia-acelerada"]`,
	}
	one := func(i int) []AugmentPick { return []AugmentPick{{AugmentIndex: i, Stacks: 1}} }
	for _, c := range []struct {
		why, spell string
		picks      []AugmentPick
		powers     []PowerAugmentPick
		want       engine.ActionCost
	}{
		{"a da magia", "armamento-da-natureza", nil, nil, engine.ActionStandard},
		{"o aprimoramento que muda para movimento", "armamento-da-natureza", one(1), nil, engine.ActionMovement},
		{"o aprimoramento que muda para reação", "armadura-arcana", one(0), nil, engine.ActionReaction},
		{"o de uma hora não cabe num turno", "consagrar", one(2), nil, engine.ActionLong},
		{"a Acelerada é livre", "luz", nil, []PowerAugmentPick{{PowerID: book.QuickenedSpellPower, Stacks: 1}}, engine.ActionFree},
	} {
		if got := CasterOf(arcanist).Spell(c.spell).Execution(c.picks, c.powers); got != c.want {
			t.Errorf("%s (%s): %q; want %q", c.why, c.spell, got, c.want)
		}
	}
	ritual := arcanist
	ritual.Conditionals = []string{RitualSwitch}
	if got := CasterOf(ritual).Spell("luz").Execution(nil, nil); got != engine.ActionLong {
		t.Errorf("como ritual, a Luz leva %q; want a longa de 1 hora (p131)", got)
	}
}
