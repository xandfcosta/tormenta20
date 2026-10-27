package table

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/live"
)

// A FAIXA DO TESTE — irmã da do ataque, menos os verbos.
//
// Um teste não muda PV, condição nem turno: ele INFORMA. Não há o que confirmar
// nem o que cancelar, e desenhar os botões do ataque aqui daria à mesa dois
// controles que não fazem nada. É a única diferença entre as duas faixas, e ela
// é a razão inteira de este tipo existir em vez de reusar o `attackProposal`.

// skillTestBand é um teste rolado, do jeito que a mesa lê.
type skillTestBand struct {
	Who   string
	Skill string
	// Tally é a conta inteira em monoespaçado: "d20 14 + 7 = 21". A mesa
	// desconfia de número sem origem — "21" pede "de onde?".
	Tally string
	// Seal e Class são o selo e a cor dele. Sem CD não há passou/falhou, então o
	// selo diz só o que NÃO depende de CD: os naturais da p221.
	Seal  string
	Class string
	// ByHand diz que o d20 veio da mesa. Sem a marca ninguém sabe se confia no
	// app ou no dado que rolou na mão.
	ByHand bool
	// Paid são os gastos de PM do teste, escritos — "Audácia +5 · 2 PM"
	// (ALE-489). Eles já estão no modificador; aqui a mesa lê de onde vieram.
	Paid string
	// Rerolled diz os "rolar de novo" pagos e o dado que saiu (ALE-507).
	Rerolled string
	// Offers são os "rolar de novo" de QUEM ROLOU, e só ele os vê.
	Offers []rerollOffer
}

// skillTestBandOf traduz o teste do regime na faixa, ou nil quando não há.
func skillTestBandOf(st *live.SessionRuntimeState) *skillTestBand {
	if st == nil || st.LastSkillTest == nil {
		return nil
	}
	t := st.LastSkillTest
	band := &skillTestBand{
		Who: t.Who, Skill: t.Skill, ByHand: t.ByHand,
		Tally: fmt.Sprintf("d20 %s %s %d = %d", writtenDice(t), signOf(t.Modifier), abs(t.Modifier), t.Total),
		Seal:  "Teste", Class: "mesa-ataque-comum",
		Paid: writtenPaid(t.Spends), Rerolled: writtenRerolls(t),
	}
	// A TROCA aparece no nome: "Diplomacia com Enganação" — o número da conta é
	// o da outra perícia, e sem isso a mesa estranharia o total.
	if t.UsedSkill != "" {
		band.Skill = t.Skill + " com " + t.UsedSkill
	}
	// OS NATURAIS SÃO DO DADO (p221), e são o único veredicto que esta fatia
	// pode dar: "sempre é um sucesso" e "sempre é uma falha" não dependem de CD
	// nenhuma, e a CD é do mestre, dita em voz.
	switch {
	case t.Natural20:
		band.Seal, band.Class = "20 natural", "mesa-ataque-critico"
	case t.Natural1:
		band.Seal, band.Class = "1 natural", "mesa-ataque-erro"
	}
	return band
}

// writtenDice é o d20 da conta, e os DOIS quando se rolaram dois: "14 (de 14 e
// 6)" — o melhor vale, e a mesa vê o que ficou de fora (p45).
func writtenDice(t *live.SkillTestRoll) string {
	if len(t.Rolls) < 2 {
		return strconv.Itoa(t.Roll)
	}
	return fmt.Sprintf("%d (de %d e %d)", t.Roll, t.Rolls[0], t.Rolls[1])
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

// signOf e abs escrevem o modificador como a mesa o lê — "d20 14 − 3 = 11" e
// não "d20 14 + -3 = 11". O sinal é do LIVRO, que escreve "+3" e "−1" nas
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
