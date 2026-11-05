package engine

import (
	"reflect"
	"strings"
	"testing"
)

// O golpe que volta da proposta rola como a carta que foi: a dúvida é se algum campo da
// conta ficou fora do JSON. Os dois dados são escritos à mão, como nos casos do livro.
func TestAFrozenStrikeRollsLikeTheCard(t *testing.T) {
	card := WeaponCard{
		Name: "Espada longa", Skill: "Pontaria", Attack: 9, Damage: "1d8", DamageBonus: 4, CritRange: 19, CritMult: 3,
		ShootsIntoMelee: true,
		ExtraDamage:     []ExtraDamage{{Dice: "1d6", Type: "fogo"}},
	}
	target := AttackTarget{Defense: 15, DamageReduction: 2, DamageReductionByType: map[string]int{"fogo": 1}}
	situations := []SpecialSituation{TargetUnderLightCover}

	raw, err := FreezeStrike(card, target, situations)
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := ThawStrike(raw)
	if err != nil {
		t.Fatal(err)
	}
	dice := func() func(int) (int, error) {
		rolled := []int{5, 3, 4, 2}
		return func(int) (int, error) {
			next := rolled[0]
			rolled = rolled[1:]
			return next, nil
		}
	}
	want, err := RollAttackDamage(card, target, situations, D20(15), RulingCritical, dice())
	if err != nil {
		t.Fatal(err)
	}
	got, err := RollAttackDamage(frozen.Weapon, frozen.Target, frozen.Situations, D20(15), RulingCritical, dice())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("o golpe congelado rolou\n%+v\ne a carta rola\n%+v", got, want)
	}
	if frozen.Weapon.ShootsIntoMelee != true {
		t.Error("o Disparo Preciso se perdeu ao congelar")
	}
}

// Campo novo fora do fio pode ser da conta do golpe, e aí o congelado o perderia em
// silêncio. A lista obriga a decidir.
func TestEveryCardFieldOutOfTheWireIsDecided(t *testing.T) {
	notInTheStrike := map[string]bool{
		"CatalogID": true, "ItemID": true, "Doses": true, "OtherUse": true, "RefusesBoost": true,
		"DexBy": true, "DexDamage": true, "Natural": true,
		"ShootsIntoMelee": true, // este o FreezeStrike leva à parte
	}
	typed := reflect.TypeOf(WeaponCard{})
	for i := range typed.NumField() {
		field := typed.Field(i)
		if strings.Split(field.Tag.Get("json"), ",")[0] == "-" && !notInTheStrike[field.Name] {
			t.Errorf("WeaponCard.%s não vai no JSON: se a conta do golpe o lê, o FreezeStrike tem de levá-lo; se não, ponha-o nesta lista", field.Name)
		}
	}
}
