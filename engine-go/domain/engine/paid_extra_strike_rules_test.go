package engine

import (
	"reflect"
	"testing"
)

// O ATAQUE PAGO QUE NÃO GASTA A VEZ (ALE-651).
//
//   - Ataque Extra: "quando usa a ação agredir, você pode gastar 2 PM para realizar um ataque adicional uma vez
//     por rodada" (guerreiro, 6º nível, p66) — com qualquer arma.
//   - Golpe Relâmpago: "quando usa a ação agredir para fazer um ataque desarmado, você pode gastar 1 PM para
//     realizar um ataque desarmado adicional" (lutador, p76) — só o desarmado.
func TestThePaidExtraStrikeRidesTheCardsItReaches(t *testing.T) {
	world := BookRuleset(primeBook(t))
	paid := func(ch Character) map[string][]PaidExtraStrike {
		out := map[string][]PaidExtraStrike{}
		for _, card := range world.ComputeWeaponCards(ch, nil) {
			out[card.Name] = card.PaidExtras
		}
		return out
	}
	extra := PaidExtraStrike{PowerID: "class.guerreiro.ataque-extra", Name: "Ataque Extra", PM: 2}
	lightning := PaidExtraStrike{PowerID: "class.lutador.golpe-relampago", Name: "Golpe Relâmpago", PM: 1, Unarmed: true}

	guerreiro := wielding(attributeSumHero("Guerreiro", 6, `[]`), "espada-longa")
	guerreiro.Proficiencies = `["armas-simples","armas-marciais"]`
	got := paid(guerreiro)
	if !reflect.DeepEqual(got["espada-longa"], []PaidExtraStrike{extra}) || !reflect.DeepEqual(got[UnarmedName], []PaidExtraStrike{extra}) {
		t.Errorf("o guerreiro de 6º nível tem o Ataque Extra em toda carta, e veio %+v", got)
	}
	if got := paid(wielding(attributeSumHero("Guerreiro", 5, `[]`), "espada-longa")); len(got["espada-longa"]) != 0 {
		t.Errorf("o guerreiro de 5º nível ainda não tem o Ataque Extra, e veio %+v", got["espada-longa"])
	}

	lutador := wielding(attributeSumHero("Lutador", 1, `[]`), "adaga")
	got = paid(lutador)
	if !reflect.DeepEqual(got[UnarmedName], []PaidExtraStrike{lightning}) {
		t.Errorf("o lutador tem o Golpe Relâmpago no desarmado, e veio %+v", got[UnarmedName])
	}
	if len(got["adaga"]) != 0 {
		t.Errorf("o Golpe Relâmpago é desarmado, e a adaga veio com %+v", got["adaga"])
	}
}

// O GOLPE IMPRUDENTE no Golpe Relâmpago: "seus ataques desarmados recebem um dado de dano extra do mesmo tipo
// (por exemplo, se o seu dano é 2d6, você causa 3d6), mas você sofre –5 na Defesa até o início de seu próximo
// turno" (p77). O dado é "extra", e dado extra não multiplica no crítico (p231).
func TestTheImpulsiveStrikeAddsOneDieOfTheSameKind(t *testing.T) {
	card := WeaponCard{Name: UnarmedName, Damage: "1d8", DamageType: "impacto"}
	got, err := ImpulsiveStrikeCard(card)
	if err != nil {
		t.Fatal(err)
	}
	want := []ExtraDamage{{Dice: "1d8", Type: "impacto"}}
	if !reflect.DeepEqual(got.ExtraDamage, want) {
		t.Errorf("o Golpe Imprudente deu %+v, e o livro dá um d8 a mais do mesmo tipo", got.ExtraDamage)
	}
	if len(card.ExtraDamage) != 0 {
		t.Errorf("o Golpe Imprudente mexeu na carta de origem: %+v", card.ExtraDamage)
	}
	if mods := ImpulsiveStrikeGuardModifiers(); len(mods) != 1 || mods[0].Amount != -5 || mods[0].Target.K != "defense" {
		t.Errorf("a guarda do Golpe Imprudente é –5 na Defesa, e veio %+v", mods)
	}
}
