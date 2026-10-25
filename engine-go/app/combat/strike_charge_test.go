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

// A INVESTIDA NO GOLPE (p235, p42, ALE-499). O que ESTA camada decide: o +2 no
// teste, a recusa da arma à distância, a Imprudente só para quem tem o poder e
// com +1d12 no dano, e o provisório dizendo qual investida foi — é dele que a
// confirmação tira a penalidade na Defesa e o custo de ação completa.

func chargeOn(t *testing.T, hero Combatant, charge string) (live.PendingAttack, error) {
	t.Helper()
	strike := NewStrike(sheetDouble{
		"heroi": hero,
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, aTable(t), semTabuleiro{}, fixedDie(4))
	d20 := 10
	return strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Charge: charge,
	})
}

func aSwordsman() Combatant {
	return Combatant{EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}}
}

func TestTheChargeAddsTwoAndTheProvisionalSaysSo(t *testing.T) {
	out, err := chargeOn(t, aSwordsman(), ChargePlain)
	if err != nil {
		t.Fatalf("investida: %v", err)
	}
	if out.Total != 10+5+2 {
		t.Errorf("total %d, want 17 — +2 no teste de ataque (p235)", out.Total)
	}
	if out.Charge != ChargePlain {
		t.Errorf("o provisório saiu com investida %q, want %q", out.Charge, ChargePlain)
	}
}

func TestTheChargeIsAMeleeAttack(t *testing.T) {
	archer := aSwordsman()
	archer.Weapons = []engine.WeaponCard{{Name: "Arco curto", Skill: "Pontaria", Damage: "1d6", Attack: 5, CritRange: 20, CritMult: 3}}
	if _, err := chargeOn(t, archer, ChargePlain); !errors.Is(err, app.ErrRefused) || !strings.Contains(err.Error(), "corpo a corpo") {
		t.Errorf("investida de arco: err = %v, want recusa que diz corpo a corpo", err)
	}
}

func TestTheRecklessChargeNeedsThePowerAndAddsOneD12(t *testing.T) {
	if _, err := chargeOn(t, aSwordsman(), ChargeReckless); !errors.Is(err, app.ErrRefused) || !strings.Contains(err.Error(), "Investida Imprudente") {
		t.Errorf("Imprudente sem o poder: err = %v, want recusa pelo nome", err)
	}
	barbarian := aSwordsman()
	barbarian.ChargesRecklessly = true
	plain, err := chargeOn(t, barbarian, ChargePlain)
	if err != nil {
		t.Fatalf("investida: %v", err)
	}
	reckless, err := chargeOn(t, barbarian, ChargeReckless)
	if err != nil {
		t.Fatalf("Imprudente: %v", err)
	}
	// O dado fixo é 4: o +1d12 rola 4.
	if reckless.RawDamage-plain.RawDamage != 4 {
		t.Errorf("dano bruto %d na Imprudente e %d na investida; want +4 (o 1d12 do dado fixo, p42)",
			reckless.RawDamage, plain.RawDamage)
	}
	if reckless.Total != plain.Total {
		t.Errorf("a Imprudente mexeu no ataque (%d × %d); ela só troca Defesa por dano", reckless.Total, plain.Total)
	}
}
