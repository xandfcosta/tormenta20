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

// A SUGESTÃO DO LIVRO é o botão cheio, e o crítico diz o multiplicador da arma.
func TestTheSuggestedVerdictIsTheFilledButton(t *testing.T) {
	st := live.EmptyRuntimeState()
	st.Initiative = []live.InitiativeEntry{
		{ID: "a", Label: "Arwen"}, {ID: "b", Label: "Ogro"},
	}
	cases := []struct {
		hit, crit bool
		mult      int
		want      string
	}{{false, false, 2, "Errou"}, {true, false, 2, "Acertou"}, {true, true, 3, "Crítico ×3"}}
	for _, tc := range cases {
		st.PendingAttack = &live.PendingAttack{
			AttackerEntryID: "a", TargetEntryID: "b", Hit: tc.hit, Critical: tc.crit, CritMult: tc.mult, ByUserID: 7,
		}
		band := attackProposalOf(st, 7)
		var filled []string
		for _, verdict := range band.Verdicts {
			if verdict.Suggested {
				filled = append(filled, verdict.Label)
			}
		}
		if len(band.Verdicts) != 3 || len(filled) != 1 || filled[0] != tc.want {
			t.Errorf("hit=%v crit=%v ×%d encheu %v, e o esperado é só %q", tc.hit, tc.crit, tc.mult, filled, tc.want)
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

// A FAIXA DA MANOBRA entrega O EMBATE INTEIRO e NÃO o desfecho dele.
//
// Este caso afirmava o contrário — "a faixa diz o que a confirmação vai fazer" —,
// e por baixo dele a faixa escrevia "Derrubou" lendo um vencedor que o motor
// calculava. A regra que o derrubou é a seção "O sistema INFORMA; o mestre
// DECIDE" do `CLAUDE.md` da raiz.
//
// O que a linha tem de carregar são os três números que o mestre juntaria de
// cabeça (os dois totais e a subtração) mais a consequência que o LIVRO prevê —
// que é o que poupa a ida à p234 no meio do turno.
func TestTheManeuverBandGivesTheWholeContestAndNoVerdict(t *testing.T) {
	derrubou := live.ManeuverRoll{
		Kind: "derrubar", Opposed: 10, Margin: 9, ConditionOnAWin: "caido",
	}
	linha := maneuverLine(live.PendingAttack{Roll: 15, Total: 19, Maneuver: &derrubou})
	for _, pedaco := range []string{"19 vs 10", "diferença +9", "deixa caído"} {
		if !strings.Contains(linha, pedaco) {
			t.Errorf("a linha %q não diz %q", linha, pedaco)
		}
	}
	// O CRACHÁ DIZ QUAL MANOBRA, no infinitivo: ele já disse "Derrubou", e o
	// passado afirma um desfecho.
	if selo := maneuverSeal(derrubou); selo != "Derrubar" {
		t.Errorf("o crachá da manobra diz %q, e o infinitivo é o que não julga", selo)
	}

	// A DIFERENÇA NEGATIVA SAI COM SINAL e a condição prevista CONTINUA DITA: o
	// que a linha escreve é o que o livro reserva à vitória, não o que aconteceu.
	// Uma condição que sumisse no negativo obrigaria a faixa a saber quem venceu.
	perdeu := maneuverLine(live.PendingAttack{
		Roll: 3, Total: 7,
		Maneuver: &live.ManeuverRoll{
			Kind: "derrubar", Opposed: 18, Margin: -11, ConditionOnAWin: "caido",
		},
	})
	if !strings.Contains(perdeu, "diferença -11") {
		t.Errorf("a linha %q não diz a diferença negativa com sinal", perdeu)
	}
	if !strings.Contains(perdeu, "deixa caído") {
		t.Errorf("a linha %q escondeu a condição porque a diferença é negativa — "+
			"decidir se a manobra falhou é do mestre, e ele decide com ela à vista", perdeu)
	}

	// A MANOBRA SEM CONDIÇÃO não inventa uma: o desarmar derruba um item.
	desarmou := maneuverLine(live.PendingAttack{
		Roll: 15, Total: 19,
		Maneuver: &live.ManeuverRoll{Kind: "desarmar", Opposed: 10, Margin: 9},
	})
	if strings.Contains(desarmou, "deixa") {
		t.Errorf("a linha do desarmar promete condição: %q", desarmou)
	}
}

// NENHUMA PALAVRA DE DESFECHO CHEGA AO JOGADOR, e este caso varre o TEXTO da
// faixa recortada (`forPlayer`), porque é o texto que ele lê.
//
// O desfecho é do mestre (ALE-532): a faixa DELE diz "sucesso" ou "falha" para
// facilitar a leitura; a do jogador, nunca. Uma palavra reintroduzida na linha
// do jogador passa verde por toda a suíte — é a metade que o guarda de forma do
// motor não alcança.
func TestNoManeuverTextTheTablePlayerSeesDeclaresAWinner(t *testing.T) {
	julgamentos := []string{
		"venceu", "vence", "perdeu", "perde", "falhou", "falha",
		"derrubou", "agarrou", "desarmou", "empurrou", "quebrou",
		"sucesso", "acertou", "errou", "role de novo", "vs",
	}
	medidas := 0
	for _, kind := range []string{"derrubar", "agarrar", "desarmar", "empurrar", "quebrar"} {
		for _, margin := range []int{9, 0, -11} {
			for _, mine := range []bool{true, false} {
				m := live.ManeuverRoll{Kind: kind, OpposedRoll: 4, Opposed: 10, Margin: margin, ConditionOnAWin: "caido"}
				pa := live.PendingAttack{Roll: 15, Total: 10 + margin, Maneuver: &m}
				gm := attackProposal{Maneuver: true, Mine: mine, Action: maneuverSeal(m),
					Duel: duelView{Seal: "Sucesso"}, Tally: maneuverLine(pa), Said: rulingSaid(pa)}
				band := gm.forPlayer(pa)
				texto := strings.ToLower(band.Action + " " + band.Duel.Seal + " " + band.Said + " " + band.Tally + " " +
					strings.Join(attackNotes(band), " "))
				medidas++
				for _, palavra := range julgamentos {
					if strings.Contains(texto, palavra) {
						t.Errorf("a faixa do jogador (dele: %v) de %q com diferença %+d escreveu %q:\n  %s\n"+
							"Quem julga o embate é o mestre — ver \"O veredito é do mestre, e só ele o "+
							"vê\" no CLAUDE.md da raiz.", mine, kind, margin, palavra, texto)
					}
				}
			}
		}
	}
	// O DENOMINADOR: uma varredura que não montasse faixa nenhuma daria verde.
	if medidas != 30 {
		t.Fatalf("a varredura leu %d faixas e os casos são 5 × 3 × 2 = 30", medidas)
	}
}

// A FAIXA DO MESTRE lê o embate pela p234 e diz o efeito extra da margem de 5.
func TestTheGMManeuverBandReadsTheContest(t *testing.T) {
	cases := []struct {
		kind                      string
		roll, total               int
		oppRoll, opposed          int
		margin                    int
		want, wantTone, wantExtra string
	}{
		{"derrubar", 15, 19, 4, 10, 9, "Sucesso por 5+", "good", "por 5+: também empurra 1 quadrado"},
		{"desarmar", 12, 14, 8, 12, 2, "Sucesso", "good", ""},
		{"empurrar", 18, 22, 2, 12, 10, "Sucesso por 5+", "good", "por 5+: empurra 4,5m"},
		{"agarrar", 4, 18, 13, 18, 0, "Sucesso", "good", ""},
		{"agarrar", 3, 8, 14, 19, -11, "Falha", "danger", ""},
		{"quebrar", 7, 12, 7, 12, 0, "Empate", "", "bônus iguais: outro teste (p234)"},
	}
	for _, c := range cases {
		pa := live.PendingAttack{Roll: c.roll, Total: c.total, Maneuver: &live.ManeuverRoll{
			Kind: c.kind, OpposedRoll: c.oppRoll, Opposed: c.opposed, Margin: c.margin,
		}}
		if got, tone, extra := maneuverVerdict(pa); got != c.want || tone != c.wantTone || extra != c.wantExtra {
			t.Errorf("%s %d contra %d: veredito %q (%q), extra %q; queria %q (%q), extra %q",
				c.kind, c.total, c.opposed, got, tone, extra, c.want, c.wantTone, c.wantExtra)
		}
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

// O DANO NÃO LETAL É DITO, porque ele muda o que a mesa faz com o número.
//
// "9 de dano" e "9 de dano não letal" levam a mesas diferentes: o segundo
// derruba e não mata (p236), e o mestre que não souber vai pedir um teste de
// Constituição que a regra não manda fazer.
//
// Ele entra na CONTA e não na prosa das situações, ao contrário da Tabela 5-3:
// é uma qualificação do DANO, e fica colado nele.
func TestTheAttackLineSaysWhenTheDamageIsNonLethal(t *testing.T) {
	line := attackLine(live.PendingAttack{
		Roll: 15, Total: 20, Defense: 13, Hit: true,
		Dice: []int{6}, Faces: 8, RawDamage: 9, Damage: 9, NonLethal: 9,
	})
	// O ESPERADO USA A MESMA CONSTANTE porque o espaço entre as palavras é
	// INQUEBRÁVEL — escrevê-lo à mão aqui daria um teste que passa com o espaço
	// comum, que é exatamente o que o olho reprovou a 390px.
	if line != "20 vs 13 · 1d8+3 (6) = 9 de dano "+nonLethalWords {
		t.Errorf("a linha = %q, e a p236 faz do não letal outra coisa", line)
	}
	// E O DANO COMUM não ganha palavra nenhuma: "não letal" só aparece onde há
	// regra, senão a mesa procura o que mudou.
	comum := attackLine(live.PendingAttack{
		Roll: 15, Total: 20, Defense: 13, Hit: true,
		Dice: []int{6}, Faces: 8, RawDamage: 9, Damage: 9,
	})
	if comum != "20 vs 13 · 1d8+3 (6) = 9 de dano" {
		t.Errorf("a linha do dano comum = %q", comum)
	}
}

// A CHANCE DE IGNORAR O EXTRA DO CRÍTICO É DITA À MESA, com o número que ela
// tira se a rolagem mandar (p166, p338, ALE-436).
//
// O app INFORMA e a mesa ROLA: a linha traz a chance e o quanto o crítico
// acrescentou, que é exatamente o par que o mestre precisa.
func TestTheAttackLineTellsTheChanceToIgnoreTheCriticalExtra(t *testing.T) {
	line := attackLine(live.PendingAttack{
		Roll: 19, Total: 24, Defense: 17, Hit: true, Critical: true,
		Dice: []int{8, 5}, Faces: 8, RawDamage: 16, Damage: 16,
		CriticalExtra: 5, CritExtraIgnoreChance: 50,
	})
	if line != "24 vs 17 · 2d8+3 (8+5) = 16 de dano · 50% de ignorar os 5 do crítico" {
		t.Errorf("a linha = %q", line)
	}
}

// E ELA SÓ APARECE QUANDO HÁ O QUE IGNORAR, pela mesma razão da RD: anunciar
// uma chance num acerto normal manda o mestre rolar por nada.
func TestTheAttackLineKeepsQuietWhenThereIsNoCriticalExtra(t *testing.T) {
	line := attackLine(live.PendingAttack{
		Roll: 14, Total: 19, Defense: 17, Hit: true,
		Dice: []int{8}, Faces: 8, RawDamage: 8, Damage: 8,
		CritExtraIgnoreChance: 50,
	})
	if line != "19 vs 17 · 1d8 (8) = 8 de dano" {
		t.Errorf("a linha = %q: sem crítico não há extra, e sem extra não há o que rolar", line)
	}
}

// A LINHA DIZ O QUE O LIVRO RESERVA AO GOLPE, e nada sobre o desfecho (ALE-441).
//
// O mesmo desenho da manobra: o app informa a condição PREVISTA e quem a aplica
// é o mestre, no gesto da fila que já existe.
func TestTheAttackLineSaysTheConditionTheBookForesees(t *testing.T) {
	line := attackLine(live.PendingAttack{
		Roll: 14, Total: 19, Defense: 17, Hit: true,
		Dice: []int{8}, Faces: 8, RawDamage: 8, Damage: 8,
		Announces: []live.ForeseenCondition{
			{Condition: "sangrando", Note: "a perda de PV desta arma é cumulativa"},
		},
	})
	if line != "19 vs 17 · 1d8 (8) = 8 de dano · o livro prevê: Sangrando "+
		"(a perda de PV desta arma é cumulativa)" {
		t.Errorf("a linha = %q", line)
	}
}

// DUAS CONDIÇÕES saem na mesma linha, e o NOME vai traduzido: o id é do código
// e a mesa lê português (a faixa da manobra já fazia isso).
func TestTheAttackLineNamesEveryForeseenConditionInPortuguese(t *testing.T) {
	line := attackLine(live.PendingAttack{
		Roll: 19, Total: 24, Defense: 17, Hit: true, Critical: true,
		Dice: []int{8, 5}, Faces: 8, RawDamage: 16, Damage: 16,
		Announces: []live.ForeseenCondition{
			{Condition: "fraco", Note: "em criatura viva"},
			{Condition: "atordoado", Note: "por uma rodada; Fort evita"},
		},
	})
	for _, escrito := range []string{"Fraco (em criatura viva)", "Atordoado (por uma rodada; Fort evita)"} {
		if !strings.Contains(line, escrito) {
			t.Errorf("a linha não diz %q: %s", escrito, line)
		}
	}
	// E NENHUMA PALAVRA DE DESFECHO: "fica", "sofre", "aplicado" diriam que
	// aconteceu, e quem diz isso é o mestre.
	for _, proibida := range []string{"fica ", "sofre ", "aplicado", "atordoou"} {
		if strings.Contains(line, proibida) {
			t.Errorf("a linha diz %q, e isso é veredicto: %s", proibida, line)
		}
	}
}
