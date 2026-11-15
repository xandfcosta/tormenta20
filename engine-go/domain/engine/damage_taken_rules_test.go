package engine

import (
	"reflect"
	"testing"
)

// O DANO QUE O PERSONAGEM SOFRE OBEDECE A ELE (ALE-650).
//
//   - Ossos Frágeis: "Você sofre 1 ponto de dano adicional por dado de dano de impacto" (kliren, p28).
//   - Sangue Frio: "Você sofre 1 ponto de dano adicional por dado de dano de frio" (trog, p31).
//   - Fonte Elemental: "Você é imune a dano desse tipo. Se fosse sofrer dano mágico desse tipo, em vez disso
//     cura PV em quantidade igual à metade do dano" (golem, p27).
//   - Evasão e Evasão Aprimorada: o teste de Reflexos que reduz à metade zera o dano se passar, e a aprimorada
//     corta à metade se falhar; nenhuma vale de armadura pesada ou imóvel (p47, p48, p73, p74).
func TestWhatTheBodyDoesWithTheDamageItTakes(t *testing.T) {
	r := BookRuleset(primeBook(t))
	race := func(id, variants string) Character {
		return Character{Races: []CharacterRace{{Race: id}}, RaceAbilityChoices: variants, ClassPowers: "[]"}
	}
	for _, c := range []struct {
		why  string
		ch   Character
		want DamageTaken
	}{
		{"o kliren", race("kliren", ""), DamageTaken{
			PerDie: []DamageTakenLine{{Type: "impacto", Amount: 1, Source: "Ossos Frágeis"}},
		}},
		{"o trog", race("trog", ""), DamageTaken{
			PerDie: []DamageTakenLine{{Type: "frio", Amount: 1, Source: "Sangue Frio"}},
		}},
		{"o golem do fogo", race("golem", `["golem-fonte-fogo"]`), DamageTaken{
			Immune:        []DamageTakenLine{{Type: "fogo", Amount: 1, Source: "Fonte Elemental"}},
			HealedByMagic: []DamageTakenLine{{Type: "fogo", Amount: 1, Source: "Fonte Elemental"}},
		}},
		{"o golem sem a origem escolhida", race("golem", ""), DamageTaken{}},
		{"o humano", race("humano", ""), DamageTaken{}},
		{"o bucaneiro de 2º nível", attributeSumHero("Bucaneiro", 2, `[]`), DamageTaken{
			Evasion: DamageTakenLine{Type: EvasionFlag, Amount: 1, Source: "Evasão"},
		}},
		{"o ladino de 10º nível", attributeSumHero("Ladino", 10, `[]`), DamageTaken{
			Evasion: DamageTakenLine{Type: ImprovedEvasionFlag, Amount: 1, Source: "Evasão Aprimorada"},
		}},
		{"o ladino de 1º nível", attributeSumHero("Ladino", 1, `[]`), DamageTaken{}},
		{"o bucaneiro de armadura pesada", wearing(attributeSumHero("Bucaneiro", 2, `[]`), "armadura-completa"), DamageTaken{}},
	} {
		got := r.DamageTakenOf(c.ch, nil)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: o dano o atinge assim\n  %+v\ne o livro diz\n  %+v", c.why, got, c.want)
		}
	}
}

// O GOLPE NO KLIREN: a clava (1d6 de impacto) num crítico x2 rola dois dados, e cada um traz +1 (p28). A RD
// come DEPOIS: o ponto a mais é dano sofrido como qualquer outro.
func TestTheStrikeAddsTheFragileBonesPerDie(t *testing.T) {
	clava := WeaponCard{Name: "Clava", Damage: "1d6", DamageType: "impacto", CritRange: 20, CritMult: 2}
	target := AttackTarget{Defense: 10, DamageReduction: 1,
		TakenPerDie: []DamageTakenLine{{Type: "impacto", Amount: 1, Source: "Ossos Frágeis"}}}
	out, err := RollAttackDamage(clava, target, nil, D20(20), RulingCritical, fixedDice(t, 3, 4))
	if err != nil {
		t.Fatal(err)
	}
	// 3 + 4 de dados, +2 dos Ossos Frágeis, −1 de RD.
	if out.Damage != 8 {
		t.Errorf("o crítico de clava no kliren deu %d de dano, e são 3+4 dos dados, +2 por dois dados de impacto, −1 de RD = 8", out.Damage)
	}
	if !reflect.DeepEqual(out.Taken, []TakenRow{{Label: "Ossos Frágeis", Amount: 2}}) {
		t.Errorf("a conta não diz de onde veio o dano a mais: %+v", out.Taken)
	}
}

// O GOLPE NO GOLEM DO FOGO: a espada flamejante tem o 1d6 de fogo do encanto, e a imunidade o come inteiro — e
// só ele: o corte da espada passa.
func TestTheStrikeLetsTheImmunityEatItsType(t *testing.T) {
	espada := WeaponCard{Name: "Espada longa", Damage: "1d8", DamageType: "corte", CritRange: 19, CritMult: 2,
		ExtraDamage: []ExtraDamage{{Dice: "1d6", Type: "fogo"}}}
	target := AttackTarget{Defense: 10, ImmuneTo: []DamageTakenLine{{Type: "fogo", Amount: 1, Source: "Fonte Elemental"}}}
	out, err := RollAttackDamage(espada, target, nil, D20(15), RulingHit, fixedDice(t, 5, 6))
	if err != nil {
		t.Fatal(err)
	}
	if out.Damage != 5 {
		t.Errorf("a espada flamejante no golem do fogo deu %d, e o fogo (6) some: são os 5 do corte", out.Damage)
	}
	if !reflect.DeepEqual(out.Taken, []TakenRow{{Label: "imune a fogo (Fonte Elemental)", Amount: -6}}) {
		t.Errorf("a conta não diz o que a imunidade comeu: %+v", out.Taken)
	}
}
