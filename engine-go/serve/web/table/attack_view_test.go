package table

import (
	"strings"
	"testing"

	"t20engine/domain/live"
)

// A LINHA DA CONTA tem três ramos, e cada um é uma decisão sobre o que a mesa
// precisa ler — não uma formatação a mais.
func TestTheAttackLineStopsAtTheComparisonWhenItMisses(t *testing.T) {
	line := attackLine(live.PendingAttack{Roll: 9, Total: 14, Defense: 17})
	if line != "14 vs 17" {
		t.Errorf("a linha = %q, e um ataque que erra para na comparação", line)
	}
	if strings.Contains(line, "dano") {
		t.Errorf("a linha = %q: escrever dano 0 convida a procurar de onde o zero saiu", line)
	}
}

// O CRÍTICO do exemplo do livro (p142): 1d8+3 com x2 vira 2d8+3.
func TestTheAttackLineWritesTheNotationTheTableReads(t *testing.T) {
	line := attackLine(live.PendingAttack{
		Roll: 19, Total: 24, Defense: 17, Hit: true, Critical: true,
		Dice: []int{8, 5}, Faces: 8, RawDamage: 16, Damage: 16,
	})
	if line != "24 vs 17 · 2d8+3 (8+5) = 16 de dano" {
		t.Errorf("a linha = %q", line)
	}
}

// A RD SÓ APARECE QUANDO AGIU: "RD 0" é ruído numa linha lida no meio do turno.
func TestTheAttackLineNamesTheReductionOnlyWhenItBit(t *testing.T) {
	withRD := attackLine(live.PendingAttack{
		Roll: 14, Total: 19, Defense: 17, Hit: true,
		Dice: []int{8}, Faces: 8, RawDamage: 8, Absorbed: 5, Damage: 3,
	})
	if withRD != "19 vs 17 · 1d8 (8) = 8 · RD 5 → 3 de dano" {
		t.Errorf("a linha = %q, e a p229 manda 8 contra RD 5 virar 3", withRD)
	}
	withoutRD := attackLine(live.PendingAttack{
		Roll: 14, Total: 19, Defense: 17, Hit: true,
		Dice: []int{8}, Faces: 8, RawDamage: 8, Damage: 8,
	})
	// SEM RD o total É o dano: repetir o número ("= 8 → 8 de dano") foi o que
	// olhar a tela pegou, e nenhum limiar pegaria.
	if withoutRD != "19 vs 17 · 1d8 (8) = 8 de dano" {
		t.Errorf("a linha = %q: sem RD, o número aparece UMA vez", withoutRD)
	}
}

// O DADO NATURAL que desmente a comparação é NOMEADO (p221). Sem isso a faixa
// diz "ERROU · 14 vs 13" — a conta certa com cara de regra quebrada, que foi
// o que olhar a tela pegou.
func TestTheAttackLineNamesTheNaturalRollThatOverridesTheComparison(t *testing.T) {
	missOnOne := attackLine(live.PendingAttack{Roll: 1, Total: 14, Defense: 13})
	if missOnOne != "14 vs 13 · 1 natural erra" {
		t.Errorf("a linha = %q, e o 1 natural que erra contra Defesa menor tem de aparecer", missOnOne)
	}
	hitOnTwenty := attackLine(live.PendingAttack{
		Roll: 20, Total: 22, Defense: 25, Hit: true, Critical: true,
		Dice: []int{6, 3}, Faces: 8, RawDamage: 9, Damage: 9,
	})
	if hitOnTwenty != "22 vs 25 · 20 natural acerta · 2d8 (6+3) = 9 de dano" {
		t.Errorf("a linha = %q, e o 20 natural que acerta contra Defesa maior tem de aparecer", hitOnTwenty)
	}
	// Quando a comparação já conta a história, o natural é ruído.
	plainTwenty := attackLine(live.PendingAttack{
		Roll: 20, Total: 30, Defense: 25, Hit: true, Critical: true,
		Dice: []int{6, 3}, Faces: 8, RawDamage: 9, Damage: 9,
	})
	if plainTwenty != "30 vs 25 · 2d8 (6+3) = 9 de dano" {
		t.Errorf("a linha = %q: o 20 que acertaria de qualquer jeito não precisa de nota", plainTwenty)
	}
}

// O VEREDICTO é uma das três palavras, e a tinta segue a palavra.
func TestTheVerdictIsOneOfThreeWords(t *testing.T) {
	st := live.EmptyRuntimeState()
	st.Initiative = []live.InitiativeEntry{
		{ID: "a", Label: "Arwen"}, {ID: "b", Label: "Ogro"},
	}
	cases := []struct {
		hit, crit bool
		want      string
	}{{false, false, "Errou"}, {true, false, "Acertou"}, {true, true, "Crítico"}}
	for _, tc := range cases {
		st.PendingAttack = &live.PendingAttack{
			AttackerEntryID: "a", TargetEntryID: "b", Hit: tc.hit, Critical: tc.crit, ByUserID: 7,
		}
		band := attackProposalOf(st, 7)
		if band == nil || band.Verdict != tc.want {
			t.Errorf("hit=%v crit=%v deu %v, queria %q", tc.hit, tc.crit, band, tc.want)
		}
		if band.Attacker != "Arwen" || band.Target != "Ogro" {
			t.Errorf("os nomes vêm da FILA, e vieram %q → %q", band.Attacker, band.Target)
		}
		if !band.Mine {
			t.Errorf("quem rolou (7) é quem olha (7): a faixa tem de deixar ele cancelar")
		}
	}
	if band := attackProposalOf(st, 99); band.Mine {
		t.Errorf("quem NÃO rolou não cancela o que não é dele")
	}
}
