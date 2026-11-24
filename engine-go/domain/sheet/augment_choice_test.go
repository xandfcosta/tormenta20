package sheet

import (
	"slices"
	"testing"

	"t20engine/domain/book"
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
