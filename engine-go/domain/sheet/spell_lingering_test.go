package sheet

import "testing"

// A MARCA NA FILA: o dado que a magia deixa na criatura atingida e rola na vez dela, com os aumentos da conjuração
// (ver GLOSSARY, **marca na fila**). O aumento que vale para os dois dados soma nos dois, e o da Lança Ígnea soma
// diferente em cada um (p196).
func TestTheLingeringRollFollowsTheAugmentsOfTheCast(t *testing.T) {
	caster := CasterOf(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 20}, {ClassName: "Clérigo", Level: 20}}})
	for _, c := range []struct {
		why, spell string
		picks      []AugmentPick
		want       string
	}{
		{"o muco da Flecha Ácida (p193)", "flecha-acida", nil, "[2d6 ácido]"},
		{"o aumento da Flecha Ácida vale no muco", "flecha-acida", []AugmentPick{{AugmentIndex: 2, Stacks: 1}}, "[3d6 ácido]"},
		{"as chamas da Lança Ígnea de Aleph (p196)", "lanca-ignea-de-aleph", nil, "[2d6 fogo]"},
		{"o aumento da Lança soma +1d6 nas chamas", "lanca-ignea-de-aleph", []AugmentPick{{AugmentIndex: 0, Stacks: 2}}, "[4d6 fogo]"},
		{"a Ilusão Lacerante (p195)", "ilusao-lacerante", nil, "[3d6 psíquico]"},
		{"o aumento da Ilusão Lacerante", "ilusao-lacerante", []AugmentPick{{AugmentIndex: 0, Stacks: 1}}, "[5d6 psíquico]"},
		{"o Círculo da Restauração (p184)", "circulo-da-restauracao", nil, "[3d8+3 cura]"},
		{"o aumento do Círculo da Restauração", "circulo-da-restauracao", []AugmentPick{{AugmentIndex: 0, Stacks: 1}}, "[4d8+4 cura]"},
	} {
		roll, err := caster.Spell(c.spell).LingeringRoll(c.picks)
		if err != nil {
			t.Errorf("%s: %v", c.why, err)
			continue
		}
		if got := rollsWritten(roll); got != c.want {
			t.Errorf("%s: %s; want %s", c.why, got, c.want)
		}
	}
	if _, err := caster.Spell("bola-de-fogo").LingeringRoll(nil); err == nil {
		t.Error("a Bola de Fogo não deixa nada no alvo, e o dado saiu")
	}
}
