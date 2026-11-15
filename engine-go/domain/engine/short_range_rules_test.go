package engine

import (
	"slices"
	"testing"
)

// O DADO QUE SÓ VALE EM ALCANCE CURTO (ALE-652): a Escaramuça dá "+1d8 nas rolagens de dano de ataques corpo a
// corpo e à distância em alcance curto" (p51). Com o alvo além de 9m, o golpe À DISTÂNCIA perde esse dado, e a
// conta diz por quê; o corpo a corpo nunca o perde. A linha só aparece quando tirou alguma coisa: um arqueiro sem
// dado de alcance curto não precisa ler que o alvo está longe.
func TestTheShortRangeDieStaysOutBeyondShortRange(t *testing.T) {
	skirmish := ExtraDamage{Dice: "1d8", Type: "perfuracao", Range: "curto"}
	bow := WeaponCard{Name: "arco-curto", Skill: "Pontaria", Damage: "1d6", DamageType: "perfuracao", CritRange: 20, CritMult: 3,
		ExtraDamage: []ExtraDamage{skirmish}}
	far := []SpecialSituation{TargetBeyondShortRange}
	die := AttackDie{Rolled: 15, Natural: 15}

	judged, err := JudgeAttack(bow, AttackTarget{Defense: 10}, far, die, func(int) (int, error) { return 1, nil })
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(judged.Situations, "além do alcance curto") {
		t.Errorf("o arco com o dado da Escaramuça contra alvo distante não diz por que o dado saiu: %v", judged.Situations)
	}
	frozen, err := FreezeStrike(bow, AttackTarget{Defense: 10}, far)
	if err != nil {
		t.Fatal(err)
	}
	thawed, err := ThawStrike(frozen)
	if err != nil {
		t.Fatal(err)
	}
	if len(thawed.Weapon.ExtraDamage) != 0 {
		t.Errorf("o golpe congelado contra alvo distante ainda leva o dado de alcance curto: %+v", thawed.Weapon.ExtraDamage)
	}

	sword := bow
	sword.Name, sword.Skill = "espada-longa", "Luta"
	melee, _ := FreezeStrike(sword, AttackTarget{Defense: 10}, far)
	if kept, _ := ThawStrike(melee); len(kept.Weapon.ExtraDamage) != 1 {
		t.Errorf("o corpo a corpo nunca perde o dado da Escaramuça, e o perdeu: %+v", kept.Weapon.ExtraDamage)
	}
	plain := bow
	plain.ExtraDamage = nil
	if out, _ := JudgeAttack(plain, AttackTarget{Defense: 10}, far, die, func(int) (int, error) { return 1, nil }); slices.Contains(out.Situations, "além do alcance curto") {
		t.Errorf("o arco sem dado de alcance curto ganhou a linha da distância: %v", out.Situations)
	}
}

// O dado da Escaramuça SAI DO CATÁLOGO marcado como de alcance curto: é o que o golpe lê.
func TestTheSkirmishDieIsAShortRangeDie(t *testing.T) {
	world := BookRuleset(primeBook(t))
	on := map[string]bool{FlagGroupID("escaramuca"): true}
	for _, powers := range []string{`["class.cacador.escaramuca"]`, `["class.cacador.escaramuca","class.cacador.escaramuca-superior"]`} {
		hunter := wielding(attributeSumHero("Caçador", 12, powers), "adaga")
		dice := wieldedCards(world.ComputeWeaponCards(hunter, on))[0].ExtraDamage
		if len(dice) != 1 || dice[0].Range != "curto" {
			t.Errorf("%s: o dado da Escaramuça não sai como de alcance curto: %+v", powers, dice)
		}
	}
}
