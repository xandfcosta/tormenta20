package engine

import (
	"reflect"
	"testing"
)

// A MANOBRA EM SEGUIDA (ALE-651): o poder que dá uma manobra livre depois de outro gesto.
//
//   - Lutador de Chão: "quando agarra uma criatura, pode gastar 1 PM para fazer uma manobra derrubar contra
//     ela como uma ação livre" (lutador, p77).
//   - Mão na Boca: "quando acerta um ataque furtivo contra uma criatura desprevenida, você pode fazer um teste
//     de agarrar como uma ação livre" (ladino, p74).
//
// A Mão na Boca só fica PRONTA com o Ataque Furtivo ligado: é o golpe com ele que a declara.
func TestTheFollowUpManeuverComesFromThePowerThatGrantsIt(t *testing.T) {
	world := BookRuleset(primeBook(t))

	lutador := attributeSumHero("Lutador", 1, `["class.lutador.lutador-de-chao"]`)
	want := []FollowUpManeuver{{
		PowerID: "class.lutador.lutador-de-chao", Name: "Lutador de Chão",
		Maneuver: "derrubar", After: "agarrar", PM: 1, Ready: true,
	}}
	if got := world.FollowUpManeuvers(lutador, nil); !reflect.DeepEqual(got, want) {
		t.Errorf("o Lutador de Chão dá a derrubada em seguida ao agarrar, por 1 PM, e veio %+v", got)
	}
	if got := world.FollowUpManeuvers(attributeSumHero("Lutador", 1, `[]`), nil); len(got) != 0 {
		t.Errorf("o lutador sem o poder não tem manobra em seguida, e veio %+v", got)
	}

	ladino := attributeSumHero("Ladino", 1, `["class.ladino.mao-na-boca"]`)
	hand := FollowUpManeuver{
		PowerID: "class.ladino.mao-na-boca", Name: "Mão na Boca",
		Maneuver: "agarrar", NeedsFlag: "ataque-furtivo",
	}
	if got := world.FollowUpManeuvers(ladino, nil); !reflect.DeepEqual(got, []FollowUpManeuver{hand}) {
		t.Errorf("a Mão na Boca sem o Ataque Furtivo ligado não está pronta, e veio %+v", got)
	}
	hand.Ready = true
	on := map[string]bool{FlagGroupID("ataque-furtivo"): true}
	if got := world.FollowUpManeuvers(ladino, on); !reflect.DeepEqual(got, []FollowUpManeuver{hand}) {
		t.Errorf("a Mão na Boca com o Ataque Furtivo ligado está pronta, e veio %+v", got)
	}
}
