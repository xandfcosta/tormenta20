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

// A FAIXA DA MANOBRA diz o que a confirmação vai fazer, ANTES do clique.
//
// O mestre decide com a consequência à vista: descobrir depois o que o botão
// fez é exatamente o que uma faixa que mostra a conta inteira existe para
// evitar. Nem toda manobra deixa condição, e a linha só a escreve quando há.
func TestTheManeuverLineSaysWhatTheConfirmWillDo(t *testing.T) {
	derrubou := maneuverLine(live.PendingAttack{
		Roll: 15, Total: 19,
		Maneuver: &live.ManeuverRoll{Kind: "derrubar", Opposed: 10, Margin: 9, Won: true, Imposes: "caido"},
	})
	for _, pedaco := range []string{"derrubar", "19 vs 10", "por 9", "fica caído"} {
		if !strings.Contains(derrubou, pedaco) {
			t.Errorf("a linha %q não diz %q", derrubou, pedaco)
		}
	}

	// A MARGEM PEQUENA não vira texto: "por 2" é número sem consequência no meio
	// do turno, e a condição continua sendo dita porque ela É a consequência.
	apertado := maneuverLine(live.PendingAttack{
		Roll: 12, Total: 15,
		Maneuver: &live.ManeuverRoll{Kind: "derrubar", Opposed: 13, Margin: 2, Won: true, Imposes: "caido"},
	})
	if strings.Contains(apertado, "por 2") {
		t.Errorf("a linha %q escreveu uma margem que não dá efeito extra", apertado)
	}
	// E A LINHA NÃO EXPLICA A REGRA: ela é telegráfica como a do golpe, e a
	// explicação por extenso quebrava a faixa em duas a 390px empurrando a
	// consequência para o fim.
	if strings.Contains(derrubou, "efeito extra") {
		t.Errorf("a linha %q explica a regra por extenso — a conta é aritmética, e o "+
			"efeito extra do derrubar (empurrar um quadrado) o app nem aplica", derrubou)
	}
	if !strings.Contains(apertado, "fica caído") {
		t.Errorf("a linha %q não diz a condição, que é o que a confirmação vai deixar", apertado)
	}

	// A MANOBRA PERDIDA não promete condição nenhuma.
	perdeu := maneuverLine(live.PendingAttack{
		Roll: 3, Total: 7,
		Maneuver: &live.ManeuverRoll{Kind: "derrubar", Opposed: 18, Margin: -11},
	})
	if strings.Contains(perdeu, "fica") {
		t.Errorf("a linha da manobra PERDIDA promete condição: %q", perdeu)
	}

	// O EMPATE diz o que a regra pede, e não um número: a p234 manda rolar de
	// novo, e anunciar uma margem zero faria a mesa procurar o vencedor.
	empate := maneuverLine(live.PendingAttack{
		Roll: 12, Total: 15,
		Maneuver: &live.ManeuverRoll{Kind: "agarrar", Opposed: 15, AnotherRoll: true},
	})
	if !strings.Contains(empate, "role de novo") {
		t.Errorf("a linha do empate não manda rolar de novo: %q", empate)
	}
	if strings.Contains(empate, "fica") {
		t.Errorf("o empate prometeu condição: %q", empate)
	}
}

// A DEFESA QUE NÃO É A DA FICHA DIZ POR QUÊ (ALE-423).
//
// É a mesma razão do "20 natural acerta" que esta linha já nomeia: um número
// que DESMENTE o que a mesa sabe tem de vir com o motivo. O Goblin tem Defesa
// 13 na ficha; se a faixa escreve "14 vs 18" e cala, a mesa procura o erro.
func TestTheAttackLineNamesWhyTheDefenseChanged(t *testing.T) {
	pa := live.PendingAttack{
		Roll: 9, Total: 14, Defense: 18, Situations: []string{"cobertura leve"},
	}
	// A CONTA fica só com a aritmética — ela é monoespaçada, e prosa dentro
	// dela parte a linha a 390px.
	if line := attackLine(pa); line != "14 vs 18" {
		t.Errorf("a conta = %q, e a prosa não entra nela", line)
	}
	if motivo := writtenSituations(pa); motivo != "cobertura leve" {
		t.Errorf("o motivo = %q, e a Defesa 18 não é a da ficha — a mesa procura o erro "+
			"se nada disser de onde ela veio", motivo)
	}
}

// E A CAMUFLAGEM É O CASO QUE MAIS PRECISA, porque ela desmente a COMPARAÇÃO:
// o total bateu a Defesa e o ataque errou mesmo assim (p238). Sem o motivo, a
// faixa escreve "18 vs 13 · ERROU" e parece defeito.
func TestTheAttackLineNamesTheConcealmentThatUndidTheHit(t *testing.T) {
	motivo := writtenSituations(live.PendingAttack{
		Roll: 18, Total: 18, Defense: 13, Situations: []string{"camuflagem leve"},
	})
	if !strings.Contains(motivo, "camuflagem leve") {
		t.Errorf("o motivo = %q: o ataque bateu a Defesa e errou, e nada diz por quê", motivo)
	}
}

// SEM SITUAÇÃO não há prosa: o separador só existe quando há o que dizer.
func TestTheAttackLineStaysTheSameWithoutSituations(t *testing.T) {
	pa := live.PendingAttack{Roll: 9, Total: 14, Defense: 17}
	if line := attackLine(pa); line != "14 vs 17" {
		t.Errorf("a linha = %q, e sem situação ela é a de sempre", line)
	}
	if motivo := writtenSituations(pa); motivo != "" {
		t.Errorf("sem situação a prosa veio %q, e ela tem de não existir", motivo)
	}
}
