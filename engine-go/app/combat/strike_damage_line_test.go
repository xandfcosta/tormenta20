package combat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// A LINHA DO DANO NA MESA (ALE-655, desenho do dono). O "acertou" do mestre põe no passo do dano uma linha por
// situação da carta — o normal marcado, e a Chave um passo acima —; quem atacou escolhe a linha, e o "Rolar" e o
// "rolei na mão" seguem a escolha. Quem diz QUAIS linhas a carta tem é o motor (`unarmed_damage_variant_rules_test.go`).

// aHold é o desarmado de um lutador com a Chave: 1d8+3, e 1d10+3 esmagando quem agarra.
func aHold() engine.WeaponCard {
	return engine.WeaponCard{Name: "Ataque desarmado", Skill: "Luta", Damage: "1d8", DamageBonus: 3, Attack: 5,
		CritRange: 20, CritMult: 2, DamageVariants: []engine.DamageVariant{{
			ID: "class.lutador.chave", Label: "esmagando quem agarra", Source: "Chave", BookPage: 76, Damage: "1d10", Steps: 1,
		}}}
}

func holdOnTheTable(t *testing.T) (Strike, *queueDouble) {
	t.Helper()
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aHold()}},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))
	d20 := 15
	if _, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
	}); err != nil {
		t.Fatalf("propor: %v", err)
	}
	if _, err := strike.Rule(context.Background(), app.Caller{ID: 1}, "gm", 1, live.RulingHit); err != nil {
		t.Fatalf("o mestre dizer acerto: %v", err)
	}
	return strike, table
}

func TestTheHitOffersOneDamageLinePerSituationWithTheNormalChosen(t *testing.T) {
	_, table := holdOnTheTable(t)
	toRoll := table.state.PendingAttack.DamageToRoll
	if toRoll == nil || len(toRoll.Lines) != 2 {
		t.Fatalf("o acerto ofereceu as linhas %+v; want o normal e a Chave", toRoll)
	}
	if toRoll.Line != "" || toRoll.Dice[0].Faces != 8 || toRoll.Bonus != 3 {
		t.Errorf("a linha marcada é %q, com d%d+%d; want o normal, d8+3", toRoll.Line, toRoll.Dice[0].Faces, toRoll.Bonus)
	}
	held := toRoll.Lines[1]
	if held.ID != "class.lutador.chave" || held.Label != "esmagando quem agarra" || held.Dice[0].Faces != 10 || held.Bonus != 3 {
		t.Errorf("a linha da Chave é %+v; want esmagando quem agarra, d10+3", held)
	}
}

func TestTheChosenDamageLineIsWhatTheDamageRolls(t *testing.T) {
	for _, c := range []struct {
		why  string
		hand []int
		want int
	}{
		{"na mão", []int{9}, 9 + 3},
		{"no app", nil, 4 + 3}, // o dado do caso é 4
	} {
		strike, table := holdOnTheTable(t)
		if _, err := strike.ChooseDamageLine(context.Background(), app.Caller{ID: 7}, "player", 1, "class.lutador.chave"); err != nil {
			t.Fatalf("%s: escolher a linha da Chave: %v", c.why, err)
		}
		if toRoll := table.state.PendingAttack.DamageToRoll; toRoll.Line != "class.lutador.chave" || toRoll.Dice[0].Faces != 10 {
			t.Errorf("%s: depois da escolha, o passo do dano pede %+v; want a Chave, d10", c.why, toRoll)
		}
		out, err := strike.RollDamage(context.Background(), app.Caller{ID: 7}, "player", 1, c.hand)
		if err != nil {
			t.Fatalf("%s: rolar o dano: %v", c.why, err)
		}
		if out.Faces != 10 || out.Damage != c.want {
			t.Errorf("%s: o dano rolou d%d e deu %d; want d10 e %d", c.why, out.Faces, out.Damage, c.want)
		}
	}
}

func TestTheDamageLineChoiceIsRefusedOutsideTheDamageStep(t *testing.T) {
	refused := func(why string, err error, says string) {
		t.Helper()
		if !errors.Is(err, app.ErrRefused) || !strings.Contains(err.Error(), says) {
			t.Errorf("%s: err = %v, want a recusa dizendo %q", why, err, says)
		}
	}
	strike, _ := holdOnTheTable(t)
	_, err := strike.ChooseDamageLine(context.Background(), app.Caller{ID: 7}, "player", 1, "class.lutador.arma-improvisada")
	refused("uma linha que a carta não tem", err, "não tem a linha")
	_, err = strike.ChooseDamageLine(context.Background(), app.Caller{ID: 8}, "player", 1, "class.lutador.chave")
	refused("quem não atacou", err, "quem atacou")
	if _, err := strike.RollDamage(context.Background(), app.Caller{ID: 7}, "player", 1, nil); err != nil {
		t.Fatalf("rolar o dano: %v", err)
	}
	_, err = strike.ChooseDamageLine(context.Background(), app.Caller{ID: 7}, "player", 1, "class.lutador.chave")
	refused("depois do dano rolado", err, "já foi rolado")
}
