package table

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/live"
)

// A FAIXA DO TESTE (p220-221): a conta e os naturais, sem passou/falhou, que dependem da CD.

// skillTestBandOf traduz o teste do regime na faixa, ou nil quando não há.
func skillTestBandOf(st *live.SessionRuntimeState) *rollBand {
	if st == nil || st.LastSkillTest == nil {
		return nil
	}
	t := st.LastSkillTest
	band := &rollBand{Badge: "Teste", Who: t.Who + " · ", What: t.Skill, HandReroll: t.ByHand}
	// A TROCA aparece no nome: "Diplomacia com Enganação" — o número da conta é
	// o da outra perícia, e sem isso a mesa estranharia o total.
	if t.UsedSkill != "" {
		band.What = t.Skill + " com " + t.UsedSkill
	}
	// OS NATURAIS SÃO DO DADO (p221), e são o único veredicto que esta fatia
	// pode dar: "sempre é um sucesso" e "sempre é uma falha" não dependem de CD
	// nenhuma, e a CD é do mestre, dita em voz.
	switch {
	case t.Natural20:
		band.Seal, band.Tone = "20 natural", "good"
	case t.Natural1:
		band.Seal, band.Tone = "1 natural", "danger"
	}
	// A CONTA ao lado do total: a soma, o que foi pago, de onde veio o dado e os "rolar de novo" pagos. A mesa
	// desconfia de número sem origem — "21" pede "de onde?".
	tally := []string{"d20 (" + writtenDice(t) + ")" + signedTerm(t.Modifier)}
	if paid := writtenPaid(t.Spends); paid != "" {
		tally = append(tally, paid)
	}
	if t.ByHand {
		tally = append(tally, "dado rolado na mesa")
	}
	if rerolled := writtenRerolls(t.Rerolls, t.Roll); rerolled != "" {
		tally = append(tally, rerolled)
	}
	band.Lines = []rollLine{{Total: t.Total, Tally: strings.Join(tally, " · ")}}
	return band
}

// writtenDice é o que o d20 tirou, na notação da mesa — entre parênteses, e os
// DOIS quando se rolaram dois: "d20 (14, 6)". O melhor vale (p45), e a mesa vê
// o que ficou de fora.
func writtenDice(t *live.SkillTestRoll) string {
	if len(t.Rolls) < 2 {
		return strconv.Itoa(t.Roll)
	}
	return fmt.Sprintf("%d, %d", t.Rolls[0], t.Rolls[1])
}

// writtenPaid escreve os gastos do teste: "Audácia +5 · 2 PM, Orgulho +4 ·
// 2 PM". A troca grátis sai sem PM.
func writtenPaid(spends []live.SkillTestPaid) string {
	parts := make([]string, 0, len(spends))
	for _, s := range spends {
		part := s.Name
		if s.Bonus != 0 {
			part += fmt.Sprintf(" %s%d", signOf(s.Bonus), abs(s.Bonus))
		}
		if s.PM > 0 {
			part += fmt.Sprintf(" · %d PM", s.PM)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

// signOf e abs escrevem o bônus de um gasto como a mesa o lê — "Audácia +5",
// "Orgulho −1", e não "+-1". O sinal é do LIVRO, que escreve "+3" e "−1" nas
// fichas, e um menos ASCII colado num número negativo lê como dois sinais.
func signOf(n int) string {
	if n < 0 {
		return "\u2212"
	}
	return "+"
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
