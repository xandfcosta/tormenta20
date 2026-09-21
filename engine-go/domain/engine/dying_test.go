package engine

import (
	"errors"
	"reflect"
	"testing"
)

// A MORTE COM O NÚMERO DO LIVRO (T20 p236):
//
//	"Quando seus pontos de vida chegam a –10 ou a um número negativo igual à
//	metade de seus PV totais (o que for mais baixo), você morre. Por exemplo:
//	Oberon, o Martelo, um arcanista com 12 PV, morre se chegar a –10 PV. Mais
//	tarde na campanha, Oberon sobe vários níveis e chega a 30 PV. Agora, ele só
//	morre se chegar a –15 PV."
func TestDeathComesAtMinusTenOrMinusHalfTheTotal(t *testing.T) {
	cases := []struct {
		hpMax int64
		want  int64
	}{
		{12, -10}, // o Oberon de nível baixo, do livro
		{30, -15}, // o Oberon de depois, do livro
		{20, -10}, // metade é -10: os dois critérios empatam
		{31, -15}, // metade de 31 arredonda para baixo, como toda conta do livro
		{0, -10},  // sem total conhecido (NPC sem PV máximo): só o -10 existe
	}
	for _, c := range cases {
		if got := DeathThreshold(c.hpMax); got != c.want {
			t.Errorf("com %d PV totais morre-se em %d, e a conta deu %d", c.hpMax, c.want, got)
		}
	}
}

// OS QUATRO ESTADOS, e o que os separa é o PV mais a condição Sangrando: estável
// é quem está a 0 ou menos SEM sangrar.
func TestTheVitalStateComesFromTheHitPointsAndTheBleeding(t *testing.T) {
	cases := []struct {
		name     string
		hp       int64
		bleeding bool
		want     VitalState
	}{
		{"1 PV está de pé", 1, false, VitalConscious},
		{"0 PV sangrando está morrendo", 0, true, VitalDying},
		{"-9 sangrando ainda está morrendo", -9, true, VitalDying},
		{"a 0 PV sem sangrar está estável", 0, false, VitalStable},
		{"no limiar está morto", -10, true, VitalDead},
		{"abaixo do limiar também", -12, false, VitalDead},
	}
	for _, c := range cases {
		if got := VitalStateOf(c.hp, 12, c.bleeding); got != c.want {
			t.Errorf("%s: deu %q, queria %q", c.name, got, c.want)
		}
	}
}

// AS CONDIÇÕES ACOMPANHAM O PV, e cada ramo é uma frase da p236.
func TestTheDyingConditionsFollowTheHitPoints(t *testing.T) {
	cases := []struct {
		name              string
		before, after     int64
		wantAdd, wantDrop []string
	}{
		// "Se ficar com 0 PV ou menos, você cai inconsciente e fica sangrando."
		{"cair a 0 PV", 5, 0, []string{ConditionUnconscious, ConditionBleeding}, nil},
		{"cair abaixo de 0", 3, -4, []string{ConditionUnconscious, ConditionBleeding}, nil},
		// "…e não precisa mais fazer esse teste (exceto se perder mais PV)."
		{"estável que apanha volta a sangrar", -2, -5, []string{ConditionBleeding}, nil},
		// "…estabilizado… com qualquer efeito que cure pelo menos 1 PV."
		{"cura que não chega a 1 estabiliza", -5, -1, nil, []string{ConditionBleeding}},
		// "…recupere PV até um valor positivo (1 ou mais)… recobra a consciência"
		{"cura até 1 acorda", -5, 1, nil, []string{ConditionUnconscious, ConditionBleeding}},
		// quem morreu não rola mais nada
		{"chegar ao limiar para de sangrar", -8, -10, nil, []string{ConditionBleeding}},
		// morrer de uma vez, de pé: o morto também está inconsciente
		{"morrer de pé cai inconsciente", 1, -10, []string{ConditionUnconscious}, []string{ConditionBleeding}},
		// de pé para de pé, nada muda
		{"apanhar de pé não mexe em condição", 10, 4, nil, nil},
		{"nada mudou", -3, -3, nil, nil},
	}
	for _, c := range cases {
		add, drop := DyingConditionChange(c.before, c.after, 12)
		if !reflect.DeepEqual(add, c.wantAdd) || !reflect.DeepEqual(drop, c.wantDrop) {
			t.Errorf("%s (%d → %d): +%v -%v, queria +%v -%v", c.name, c.before, c.after, add, drop, c.wantAdd, c.wantDrop)
		}
	}
}

// O TESTE DE CONSTITUIÇÃO (CD 15): passar estabiliza. Igual à CD passa — a
// regra de todo teste do livro é "igual ou maior".
func TestTheConstitutionTestAgainstFifteenStabilizes(t *testing.T) {
	if !BleedingCheckPasses(12, 3) {
		t.Error("12 + 3 = 15 alcança a CD 15, e alcançar é passar")
	}
	if BleedingCheckPasses(11, 3) {
		t.Error("11 + 3 = 14 não alcança a CD 15")
	}
}

// O INSTANTE SAI DAS CONDIÇÕES, e a inconsciência tira até a reação: "sem ações
// (incluindo reações)" (p395) — é o que separa quem caiu de quem está
// atordoado, que ainda reage (p233).
func TestTheMomentComesFromTheConditions(t *testing.T) {
	stunned := MomentFor(true, 10, []string{"atordoado"})
	if stunned.CanAct || !stunned.CanReact {
		t.Errorf("atordoado não age e ainda reage (p233), e deu %+v", stunned)
	}
	down := MomentFor(false, 0, []string{ConditionUnconscious, ConditionBleeding})
	if down.CanAct || down.CanReact {
		t.Errorf("inconsciente não age nem reage (p395), e deu %+v", down)
	}
	if err := UsableNow(ActionReaction, down); !errors.Is(err, ErrCannotAct) {
		t.Errorf("a reação de quem caiu tinha de ser recusada, e veio %v", err)
	}
	// A 0 PV sem a condição gravada (uma ficha de antes desta regra): o PV basta.
	if m := MomentFor(true, 0, nil); m.CanReact {
		t.Errorf("a 0 PV ninguém reage, com ou sem a condição gravada, e deu %+v", m)
	}
	if m := MomentFor(true, 7, nil); !m.CanAct || !m.CanReact || !m.OnTurn {
		t.Errorf("de pé e na vez, age e reage, e deu %+v", m)
	}
}
