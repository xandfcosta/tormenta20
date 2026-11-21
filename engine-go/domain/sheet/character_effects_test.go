package sheet

import (
	"errors"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// OS EFEITOS DE UMA FICHA (ALE-676) se montam num lugar só: as regras dela, o personagem do motor e os interruptores
// ligados. A ficha sem regras volta com o sentinela, e cada chamador decide se isso é erro ou "nada a ler".
func TestTheCharacterEffectsCarryTheConditionsAndTheSwitches(t *testing.T) {
	if _, _, err := CharacterEffects(CharacterDTO{}); !errors.Is(err, ErrSheetWithoutRules) {
		t.Errorf("a ficha sem regras voltou com o erro %v; want o ErrSheetWithoutRules", err)
	}

	catalogs, err := book.EngineCatalogs()
	if err != nil {
		t.Fatalf("carregar os catálogos do motor: %v", err)
	}
	broken := CharacterDTO{
		Classes: []ClassDTO{{ClassName: "Bardo", Level: 20}}, ActiveConditions: `["alquebrado"]`,
		Conditionals: []string{engine.FlagGroupID("inspiracao")}, Ruleset: engine.BookRuleset(catalogs),
	}
	effects, ec, err := CharacterEffects(broken)
	if err != nil {
		t.Fatalf("montar os efeitos do bardo: %v", err)
	}
	rules := effects.AbilityCostRules()
	if rules.Increase != 1 || rules.Factor("Bardo") != (engine.Ratio{Num: 1, Den: 2}) {
		t.Errorf("os efeitos do bardo 20 Alquebrado sob a Inspiração deram %+v; want +1 e a metade de bardo", rules)
	}
	if len(ec.Classes) != 1 || ec.Classes[0].ClassName != "Bardo" {
		t.Errorf("o personagem do motor voltou com as classes %+v", ec.Classes)
	}
}
