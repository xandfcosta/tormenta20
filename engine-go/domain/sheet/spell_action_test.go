package sheet

import (
	"testing"

	"t20engine/domain/book"
)

// AS AÇÕES DA MAGIA: o que a magia em vigor deixa quem conjurou fazer depois, com os aprimoramentos da conjuração
// (ver GLOSSARY, **ação da magia**).
func TestTheSpellActionRollsWithTheAugmentsOfTheCast(t *testing.T) {
	caster := CasterOf(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 20}, {ClassName: "Clérigo", Level: 20}}})
	for _, c := range []struct {
		why, spell, option string
		picks              []AugmentPick
		cost               string
		want               string
	}{
		{"o Esmagar da Mão Poderosa (p198)", "mao-poderosa-de-talude", "Esmagar", nil, "movimento", "[2d6+10 impacto]"},
		{"o Esmagar com dois aumentos", "mao-poderosa-de-talude", "Esmagar", []AugmentPick{{AugmentIndex: 0, Stacks: 2}}, "movimento", "[4d6+20 impacto]"},
		{"a bola de fogo do Relâmpago com o aumento (p204)", "relampago-flamejante-de-reynard", "Bola de fogo", []AugmentPick{{AugmentIndex: 0, Stacks: 1}}, "movimento", "[11d6 fogo]"},
		{"o relâmpago com o aumento", "relampago-flamejante-de-reynard", "Relâmpago", []AugmentPick{{AugmentIndex: 0, Stacks: 1}}, "movimento", "[11d6 eletricidade]"},
		{"a energia mista com o aumento", "relampago-flamejante-de-reynard", "Energia mista", []AugmentPick{{AugmentIndex: 0, Stacks: 1}}, "padrao", "[22d12 fogo e eletricidade, metade de cada]"},
		{"os raios da Fúria do Panteão (p194)", "furia-do-panteao", "Raios", nil, "movimento", "[10d8 eletricidade]"},
		// O dardo causa "o efeito normal da magia" (p196): o dado da conjuração, com o aumento escolhido ao conjurar.
		{"o dardo de lava da Lança Ígnea", "lanca-ignea-de-aleph", "Dardo de lava",
			[]AugmentPick{{AugmentIndex: 0, Stacks: 1}, {AugmentIndex: 1, Stacks: 1}}, "livre", "[4d6 fogo 4d6 perfuração 2d6 ]"},
	} {
		action, roll, err := caster.Spell(c.spell).ActionRoll(c.picks, c.option)
		if err != nil {
			t.Errorf("%s: %v", c.why, err)
			continue
		}
		if action.Action != c.cost {
			t.Errorf("%s: custa %q; want %q", c.why, action.Action, c.cost)
		}
		if got := rollsWritten(roll); got != c.want {
			t.Errorf("%s: %s; want %s", c.why, got, c.want)
		}
	}
}

// Os dardos são CONTADOS, e só existem com o aprimoramento: a conjuração com ele não rola nada, e diz quantos.
func TestTheDartsOfTheIgneousSpearAreChargesOfTheCast(t *testing.T) {
	spear := CasterOf(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 20}}}).Spell("lanca-ignea-de-aleph")
	withDarts := []AugmentPick{{AugmentIndex: 1, Stacks: 1}}
	roll, err := spear.CastRoll(withDarts)
	if err != nil || len(roll.Parts) != 0 || roll.Charges != 4 {
		t.Errorf("conjurar com os dardos rolou %s (%v) e deu %d carga(s); want nada rolado e 4 dardos", rollsWritten(roll.Parts), err, roll.Charges)
	}
	if actions := spear.Actions(withDarts); actions == nil || actions.Options[0].Name != "Dardo de lava" {
		t.Errorf("a Lança com os dardos oferece %+v; want o dardo de lava", actions)
	}
	if actions := spear.Actions(nil); actions != nil {
		t.Errorf("a Lança sem o aprimoramento oferece ações: %+v", actions)
	}
	if _, _, err := spear.ActionRoll(nil, "Dardo de lava"); err == nil {
		t.Error("a Lança conjurada sem os dardos disparou um dardo")
	}
}

// A opção que a magia não tem é recusa com o nome do que ela tem.
func TestTheSpellActionRefusesWhatTheSpellDoesNotOffer(t *testing.T) {
	_, _, err := CasterOf(CharacterDTO{}).Spell("furia-do-panteao").ActionRoll(nil, "Granizo")
	if err == nil {
		t.Fatal("a Fúria do Panteão rolou o Granizo, que ela não tem")
	}
}

// TestEveryAugmentComposesOnTheSpellActions passa por toda ação de magia e todo aumento, até três repetições.
func TestEveryAugmentComposesOnTheSpellActions(t *testing.T) {
	caster := CasterOf(CharacterDTO{})
	composed := 0
	for _, spell := range book.Catalogs().Spells {
		if spell.Actions == nil {
			continue
		}
		for _, option := range spell.Actions.Options {
			for i := range spell.Augments {
				for stacks := 1; stacks <= 3; stacks++ {
					if _, _, err := caster.Spell(spell.ID).ActionRoll([]AugmentPick{{AugmentIndex: i, Stacks: stacks}}, option.Name); err != nil {
						t.Errorf("%s %q aprimoramento %d ×%d: %v", spell.Name, option.Name, i, stacks, err)
					}
					composed++
				}
			}
		}
	}
	if composed == 0 {
		t.Fatal("nenhum aumento sobre ação de magia: a varredura não mediu nada")
	}
}
