package engine

import (
	"strings"
	"testing"
)

// O ATAQUE EM DOIS TEMPOS: o teste, a decisão do mestre, e só então o dano.
//
// O crítico muda QUANTOS dados se rolam (p231), e quem diz se houve crítico é o
// mestre (ver o `CLAUDE.md` da raiz). Por isso o dano não existe antes da
// decisão: rolado junto com o d20, ele sairia multiplicado pela sugestão do
// livro e não pela palavra de quem decide.

// machadoFlamejante é o machado de batalha (1d8, 20/×3) com o encanto
// Flamejante (+1d6 de fogo) e +4 de dano.
func machadoFlamejante() WeaponCard {
	card := aWeapon("1d8", 4, 20, 3, 8)
	card.Name = "Machado de batalha"
	card.ExtraDamage = []ExtraDamage{{Dice: "1d6", Type: "fogo"}}
	return card
}

func TestJudgingTheAttackRollsNoDamage(t *testing.T) {
	// 19 natural na espada longa (19/×2): o livro sugere crítico, e nenhum dado
	// de dano pode ser pedido ainda.
	out, err := JudgeAttack(aWeapon("1d8", 3, 19, 2, 8), AttackTarget{Defense: 13}, nil, D20(19), noDice(t))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Hit || !out.Critical {
		t.Errorf("19 natural +8 contra Defesa 13 na margem 19 é crítico pelo livro, e a sugestão saiu acerto=%v crítico=%v",
			out.Hit, out.Critical)
	}
	if out.Total != 27 || out.Defense != 13 {
		t.Errorf("a conta do teste saiu %d vs %d, e é 27 vs 13", out.Total, out.Defense)
	}
	if len(out.Dice) != 0 || out.Damage != 0 {
		t.Errorf("julgar o ataque já trouxe dano (%v, %d): ele é rolado depois da decisão do mestre", out.Dice, out.Damage)
	}
}

func TestTheGMsCriticalMultipliesTheDiceEvenWhenTheBookSaidHit(t *testing.T) {
	// 12 natural não está na margem 19: o livro diz acerto. O mestre diz crítico,
	// e o dano sai com 2d8 e o bônus de crítico.
	card := aWeapon("1d8", 3, 19, 2, 8)
	card.CriticalBonus = 10
	out, err := RollAttackDamage(card, AttackTarget{Defense: 13}, nil, D20(12), RulingCritical, fixedDice(t, 5, 4))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Critical || len(out.Dice) != 2 {
		t.Fatalf("o crítico do mestre rolou %v (crítico=%v), e o ×2 pede 2d8", out.Dice, out.Critical)
	}
	if out.Damage != 5+4+3+10 {
		t.Errorf("o dano saiu %d, e 2d8 (5, 4) + 3 + 10 do crítico = 22", out.Damage)
	}
}

func TestTheGMsHitRollsTheWeaponOnceEvenOnTheThreatRange(t *testing.T) {
	// 19 natural na margem 19: o livro sugere crítico. O mestre diz acerto, e o
	// dano é 1d8 só, sem o bônus de crítico.
	card := aWeapon("1d8", 3, 19, 2, 8)
	card.CriticalBonus = 10
	out, err := RollAttackDamage(card, AttackTarget{Defense: 13}, nil, D20(19), RulingHit, fixedDice(t, 6))
	if err != nil {
		t.Fatal(err)
	}
	if out.Critical || len(out.Dice) != 1 || out.Damage != 9 {
		t.Errorf("o acerto do mestre saiu crítico=%v com %v e %d de dano; o esperado é 1d8 (6) + 3 = 9",
			out.Critical, out.Dice, out.Damage)
	}
}

func TestTheDamageToRollFollowsTheWeaponAndTheRuling(t *testing.T) {
	// O ×3 é do machado, e o 1d6 de fogo fica fora dele (p231).
	critical, err := DamageToRoll(machadoFlamejante(), RulingCritical)
	if err != nil {
		t.Fatal(err)
	}
	want := DamageFormula{Dice: []DiceGroup{{Count: 3, Faces: 8}, {Count: 1, Faces: 6, Type: "fogo"}}, Bonus: 4}
	if !sameFormula(critical, want) {
		t.Errorf("o crítico do machado flamejante pede %+v, e o esperado é 3d8 + 1d6 de fogo + 4", critical)
	}
	hit, err := DamageToRoll(machadoFlamejante(), RulingHit)
	if err != nil {
		t.Fatal(err)
	}
	if len(hit.Dice) != 2 || hit.Dice[0].Count != 1 {
		t.Errorf("o acerto do machado flamejante pede %+v, e o esperado é 1d8 + 1d6 de fogo + 4", hit)
	}
}

func TestTheDiceRolledByHandFillTheFormulaInOrder(t *testing.T) {
	// Na mão, o jogador digita os dados na ordem da fórmula: os 3d8 do machado,
	// depois o 1d6 de fogo.
	hand := DiceRolledByHand([]int{8, 2, 5, 6})
	out, err := RollAttackDamage(machadoFlamejante(), AttackTarget{Defense: 13}, nil, D20(20), RulingCritical, hand.Roll)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Dice) != 3 || len(out.Extra) != 1 || out.Extra[0].Total != 6 {
		t.Fatalf("os dados da mão caíram em %v e %+v; o esperado é 3d8 (8, 2, 5) e 1d6 de fogo (6)", out.Dice, out.Extra)
	}
	if out.Damage != 8+2+5+6+4 {
		t.Errorf("o dano saiu %d, e 3d8 (8, 2, 5) + 1d6 (6) + 4 = 25", out.Damage)
	}
	if left := hand.Unused(); left != 0 {
		t.Errorf("sobraram %d dados da mão sem uso", left)
	}
}

func TestADieRolledByHandOutsideItsFacesIsRefused(t *testing.T) {
	hand := DiceRolledByHand([]int{9})
	_, err := RollAttackDamage(aWeapon("1d8", 3, 19, 2, 8), AttackTarget{Defense: 13}, nil, D20(12), RulingHit, hand.Roll)
	if err == nil || !strings.Contains(err.Error(), "9") || !strings.Contains(err.Error(), "d8") {
		t.Errorf("um d8 digitado como 9 passou, ou a recusa não disse o valor e o dado: %v", err)
	}
	short := DiceRolledByHand(nil)
	if _, err := RollAttackDamage(aWeapon("1d8", 3, 19, 2, 8), AttackTarget{Defense: 13}, nil, D20(12), RulingHit, short.Roll); err == nil {
		t.Error("o dano rolou sem nenhum dado digitado")
	}
}

func sameFormula(a, b DamageFormula) bool {
	if a.Bonus != b.Bonus || len(a.Dice) != len(b.Dice) {
		return false
	}
	for i := range a.Dice {
		if a.Dice[i] != b.Dice[i] {
			return false
		}
	}
	return true
}
