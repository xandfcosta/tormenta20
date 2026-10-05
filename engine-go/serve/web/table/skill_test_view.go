package table

import (
	"fmt"

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
}

// skillTestBandOf traduz o teste do regime na faixa, ou nil quando não há.
func skillTestBandOf(st *live.SessionRuntimeState) *skillTestBand {
	if st == nil || st.LastSkillTest == nil {
		return nil
	}
	t := st.LastSkillTest
	band := &skillTestBand{
		Who: t.Who, Skill: t.Skill, ByHand: t.ByHand,
		Tally: fmt.Sprintf("d20 %d %s %d = %d", t.Roll, signOf(t.Modifier), abs(t.Modifier), t.Total),
		Seal:  "Teste", Class: "mesa-teste-selo-comum",
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
