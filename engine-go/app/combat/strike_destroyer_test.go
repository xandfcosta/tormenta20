package combat

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O DESTRUIDOR NO DANO ROLADO (ALE-652, fatia D). Quem decide se a carta rola de novo os 1 e 2 é o motor
// (`destroyer_rules_test.go`); aqui se prende o que ESTA camada faz: a conta inteira é refeita com os dados da
// arma novos e os extras de antes, a partir da carta congelada, e a carta do mestre fica sabendo o que mudou.

// aGreatsword é um montante com o Destruidor e um dado de fogo a mais, que o Destruidor não alcança.
func aGreatsword() engine.WeaponCard {
	return engine.WeaponCard{Name: "Montante", Skill: "Luta", Damage: "2d6", DamageBonus: 3, Attack: 5,
		CritRange: 19, CritMult: 2, RerollsLowDice: true,
		ExtraDamage: []engine.ExtraDamage{{Dice: "1d6", Type: "fogo"}}}
}

// destroyerOnTheTable propõe o golpe do montante, o mestre diz acerto, e quem atacou rola o dano na mão.
func destroyerOnTheTable(t *testing.T, card engine.WeaponCard, damage []int) (Strike, *queueDouble) {
	t.Helper()
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{card}},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))
	d20 := 15
	out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
	})
	if err != nil {
		t.Fatalf("propor: %v", err)
	}
	if out.RerollsLowDice != card.RerollsLowDice {
		t.Errorf("o provisório saiu com RerollsLowDice %v, e a carta diz %v", out.RerollsLowDice, card.RerollsLowDice)
	}
	if _, err := strike.Rule(context.Background(), app.Caller{ID: 1}, "gm", 1, live.RulingHit); err != nil {
		t.Fatalf("o mestre dizer acerto: %v", err)
	}
	if _, err := strike.RollDamage(context.Background(), app.Caller{ID: 7}, "player", 1, damage); err != nil {
		t.Fatalf("rolar o dano na mão: %v", err)
	}
	return strike, table
}

func TestTheDestroyerRerollsTheWeaponOnesAndTwosAndRedoesTheDamage(t *testing.T) {
	// 2d6 (1, 6) + 1d6 de fogo (2) + 3. O 1 da arma se rola de novo; o 2 do fogo não, porque não é da arma.
	for _, c := range []struct {
		why  string
		hand []int
		want []int
	}{
		{"na mão", []int{5}, []int{5, 6}},
		{"no app", nil, []int{4, 6}}, // o dado do caso é 4
	} {
		strike, _ := destroyerOnTheTable(t, aGreatsword(), []int{1, 6, 2})
		out, err := strike.RerollLowDamage(context.Background(), app.Caller{ID: 7}, "player", 1, c.hand)
		if err != nil {
			t.Fatalf("%s: rolar de novo os 1 e 2: %v", c.why, err)
		}
		fire := 0
		for _, group := range out.ExtraDice {
			for _, value := range group.Dice {
				fire += value
			}
		}
		if !slices.Equal(out.Dice, c.want) || fire != 2 || !slices.Equal(out.DamageRerolledFrom, []int{1, 6}) {
			t.Errorf("%s: a arma ficou %v, o fogo %d e antes %v; want %v, o fogo 2 e antes [1 6]",
				c.why, out.Dice, fire, out.DamageRerolledFrom, c.want)
		}
		if want := c.want[0] + c.want[1] + 2 + 3; out.Damage != want {
			t.Errorf("%s: o dano refeito foi %d, want %d", c.why, out.Damage, want)
		}
	}
}

func TestTheDestroyerIsRefusedWhenItDoesNotApply(t *testing.T) {
	refused := func(why string, err error, says string) {
		t.Helper()
		if !errors.Is(err, app.ErrRefused) || !strings.Contains(err.Error(), says) {
			t.Errorf("%s: err = %v, want a recusa dizendo %q", why, err, says)
		}
	}
	strike, _ := destroyerOnTheTable(t, aGreatsword(), []int{1, 6, 2})
	_, err := strike.RerollLowDamage(context.Background(), app.Caller{ID: 7}, "player", 1, []int{5, 5})
	refused("dois dados digitados para um 1", err, "1 dado")
	if _, err := strike.RerollLowDamage(context.Background(), app.Caller{ID: 7}, "player", 1, nil); err != nil {
		t.Fatalf("rolar de novo: %v", err)
	}
	_, err = strike.RerollLowDamage(context.Background(), app.Caller{ID: 7}, "player", 1, nil)
	refused("a segunda vez", err, "já")

	strike, _ = destroyerOnTheTable(t, aGreatsword(), []int{3, 6, 1})
	_, err = strike.RerollLowDamage(context.Background(), app.Caller{ID: 7}, "player", 1, nil)
	refused("sem 1 nem 2 na arma (o 1 é do fogo)", err, "nenhum dado da arma")

	plain := aGreatsword()
	plain.RerollsLowDice = false
	strike, _ = destroyerOnTheTable(t, plain, []int{1, 6, 2})
	_, err = strike.RerollLowDamage(context.Background(), app.Caller{ID: 7}, "player", 1, nil)
	refused("a arma sem o Destruidor", err, "Destruidor")
}
