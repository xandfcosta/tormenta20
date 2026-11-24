package sheet

import (
	"strings"
	"testing"

	"t20engine/domain/book"
)

// O RITUAL MUDA A EXECUÇÃO para 1 hora ou o dobro (p131), e "mudanças na mesma característica da magia nunca
// se acumulam" (p171): com o Celebrar Ritual ligado, o aprimoramento da magia que muda a execução e o poder
// que a muda são recusados pelo conjurador, e a folha os tranca com a razão.
func TestTheRitualDoesNotStackWithAnotherExecutionChange(t *testing.T) {
	cleric := CharacterDTO{
		Classes:      []ClassDTO{{ClassName: "Clérigo", Level: 8}},
		ClassPowers:  `["magia-acelerada"]`,
		Conditionals: []string{RitualSwitch},
	}
	// O aprimoramento 1 do Armamento da Natureza muda a execução para ação de movimento (p181).
	nature := CasterOf(cleric).Spell("armamento-da-natureza")
	if lock := nature.AugmentLock(1); lock != "não combina com o ritual" {
		t.Errorf("o aprimoramento 1 tranca com %q no ritual; want \"não combina com o ritual\"", lock)
	}
	if lock := nature.AugmentLock(0); lock != "" {
		t.Errorf("o aprimoramento 0, que não muda a execução, tranca com %q no ritual", lock)
	}
	if _, refusal := nature.AugmentPrice([]AugmentPick{{AugmentIndex: 1, Stacks: 1}}, nil); !strings.Contains(refusal, "o ritual e o aprimoramento 1 mudam a execução") {
		t.Errorf("o ritual com o aprimoramento 1 foi recusado com %q", refusal)
	}
	for _, p := range nature.PowerAugments {
		if p.Power.PowerID == book.QuickenedSpellPower && p.Lock != "não combina com o ritual" {
			t.Errorf("a Acelerada tranca com %q no ritual", p.Lock)
		}
	}
}
