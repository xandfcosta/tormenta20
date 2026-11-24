package sheet

import (
	"testing"

	"t20engine/domain/book"
)

// O DADO DA VEZ: o que a magia rola de novo na vez de quem conjurou, com os aprimoramentos da conjuração (ver
// GLOSSARY, **dado da vez**).
func TestTheTurnRollFollowsTheAugmentsOfTheCast(t *testing.T) {
	caster := CasterOf(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 20}, {ClassName: "Clérigo", Level: 20}}})
	for _, c := range []struct {
		why, spell, version string
		picks               []AugmentPick
		want                string
	}{
		{"o Ferver Sangue no início do turno (p193)", "ferver-sangue", "", nil, "[4d8 fogo]"},
		{"o aumento do Ferver Sangue vale em toda vez", "ferver-sangue", "", []AugmentPick{{AugmentIndex: 0, Stacks: 2}}, "[6d8 fogo]"},
		{"a Poeira da Podridão com um aumento (p201)", "poeira-da-podridao", "", []AugmentPick{{AugmentIndex: 0, Stacks: 1}}, "[3d8+12 trevas]"},
		{"o Enxame de Pestes no fim do turno (p192)", "enxame-de-pestes", "", nil, "[2d12 corte]"},
		{"o Enxame de Pestes maior, com o aumento", "enxame-de-pestes", "",
			[]AugmentPick{{AugmentIndex: 0, Stacks: 1}, {AugmentIndex: 1, Stacks: 1}}, "[4d12 corte, impacto ou perfuração]"},
		{"o Enxame de Pestes elemental", "enxame-de-pestes", "", []AugmentPick{{AugmentIndex: 3, Stacks: 1}}, "[5d12 ácido, eletricidade, fogo ou frio]"},
		{"o Enxame Rubro com o aumento", "enxame-rubro-de-ichabod", "", []AugmentPick{{AugmentIndex: 1, Stacks: 1}}, "[5d12 ácido]"},
		// "Muda o dano para trevas" e "aumenta o dano em +1d12" juntos: o aumento segue o tipo novo.
		{"o Enxame Rubro de trevas com o aumento", "enxame-rubro-de-ichabod", "",
			[]AugmentPick{{AugmentIndex: 1, Stacks: 1}, {AugmentIndex: 2, Stacks: 1}}, "[5d12 trevas]"},
		{"o Manto de Luz (p197)", "manto-do-cruzado", "Manto de Luz", nil, "[2d8 cura]"},
		{"o Manto de Trevas", "manto-do-cruzado", "Manto de Trevas", nil, "[4d8 trevas]"},
	} {
		roll, err := caster.Spell(c.spell).TurnRoll(c.picks, c.version)
		if err != nil {
			t.Errorf("%s: %v", c.why, err)
			continue
		}
		if got := rollsWritten(roll); got != c.want {
			t.Errorf("%s: %s; want %s", c.why, got, c.want)
		}
	}
}

// A magia sem dado da vez, e a versão que ela não tem, são recusas com o nome do que faltou.
func TestTheTurnRollRefusesWhatTheSpellDoesNotRoll(t *testing.T) {
	caster := CasterOf(CharacterDTO{})
	if _, err := caster.Spell("bola-de-fogo").TurnRoll(nil, ""); err == nil {
		t.Error("a Bola de Fogo não rola nada na vez, e o dado da vez saiu")
	}
	if _, err := caster.Spell("manto-do-cruzado").TurnRoll(nil, "Manto de Fogo"); err == nil {
		t.Error("o Manto do Cruzado não tem a versão Manto de Fogo, e o dado da vez saiu")
	}
	if _, err := caster.Spell("manto-do-cruzado").TurnRoll(nil, ""); err == nil {
		t.Error("o Manto do Cruzado tem duas versões, e o dado da vez saiu sem dizer qual")
	}
}

// TestEveryAugmentComposesOnTheTurnRoll passa por todo dado da vez, toda versão e todo aprimoramento, até três
// repetições: a soma não pode juntar faces diferentes numa parcela.
func TestEveryAugmentComposesOnTheTurnRoll(t *testing.T) {
	caster := CasterOf(CharacterDTO{})
	composed := 0
	for _, spell := range book.Catalogs().Spells {
		onSheet := caster.Spell(spell.ID)
		for _, version := range onSheet.TurnVersions() {
			for i := range spell.Augments {
				for stacks := 1; stacks <= 3; stacks++ {
					if _, err := onSheet.TurnRoll([]AugmentPick{{AugmentIndex: i, Stacks: stacks}}, version); err != nil {
						t.Errorf("%s %q aprimoramento %d ×%d: %v", spell.Name, version, i, stacks, err)
					}
					composed++
				}
			}
		}
	}
	if composed == 0 {
		t.Fatal("nenhum aprimoramento sobre dado da vez: a varredura não mediu nada")
	}
}
