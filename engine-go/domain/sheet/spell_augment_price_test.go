package sheet

import "testing"

// A FRONTEIRA DO CÍRCULO É DO SERVIDOR, e não da tela: até a fatia 6 o limite do aprimoramento existia SÓ na
// interface, e um pedido montado à mão conjurava o que a regra não permite. O preço dos aprimoramentos é do
// conjurador, e quem não alcança o círculo é recusado; quem alcança, passa — sem a segunda metade o guarda
// ficaria verde numa conferência que recusa TUDO.
func TestAnAugmentAboveTheCircleIsRefused(t *testing.T) {
	invisibility := func(level int64) SpellOnTheSheet {
		return CasterOf(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: level}}}).Spell("invisibilidade")
	}
	target := -1
	for i, a := range invisibility(17).Spell.Augments {
		if a.RequiresCircle != nil && *a.RequiresCircle >= 3 {
			target = i
			break
		}
	}
	if target < 0 {
		t.Fatal("a Invisibilidade não tem aprimoramento com `requiresCircle` ≥ 3: ou o catálogo mudou, ou o campo " +
			"parou de ser lido — nos dois casos este guarda mediria o vazio")
	}
	picks := []AugmentPick{{AugmentIndex: target, Stacks: 1}}
	// O arcanista de 5º alcança o 2º círculo; o de 17º, o 5º.
	if _, refusal := invisibility(5).AugmentPrice(picks); refusal == "" {
		t.Error("um aprimoramento de círculo alto passou para quem alcança só o 2º")
	}
	if lock := invisibility(5).AugmentLock(target); lock == "" {
		t.Error("o cadeado do aprimoramento de círculo alto está aberto para quem alcança só o 2º")
	}
	if _, refusal := invisibility(17).AugmentPrice(picks); refusal != "" {
		t.Errorf("quem alcança o 5º foi recusado: %s", refusal)
	}
	if lock := invisibility(17).AugmentLock(target); lock != "" {
		t.Errorf("o cadeado está fechado para quem alcança o 5º: %s", lock)
	}
}
