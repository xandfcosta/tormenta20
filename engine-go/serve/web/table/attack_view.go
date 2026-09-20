package table

import (
	"fmt"
	"strings"

	"t20engine/domain/live"
)

// O ATAQUE PROPOSTO, como a mesa o lê.
//
// A faixa é irmã da do movimento (`moveProposed`) e o desenho é o mesmo: o que
// DECIDE o clique primeiro, os dois verbos à direita, e o selo "Rascunho" do
// lado de quem não confirma. O que muda é o que decide — no movimento é o
// custo, aqui é o VEREDICTO: acertar ou não muda o que a mesa faz em seguida, e
// o dano só importa depois dessa resposta.
type attackProposal struct {
	// Veredicto é a palavra do crachá: CRÍTICO, ACERTOU ou ERROU.
	Verdict string
	// Classe é a tinta do crachá, e ela segue o veredicto.
	Class         string
	Attacker      string
	Target        string
	TargetEntryID string
	Weapon        string
	// Conta é a aritmética inteira numa linha. Ela existe porque a mesa
	// desconfia de número sem origem — "11 de dano" pede "de onde?", e
	// "24 vs 17 · 2d8+3 (8+5) · RD 5 → 11" não pede nada.
	Tally string
	// Meu diz se quem olha foi quem rolou: só ele cancela o que é dele. O mestre
	// cancela por qualquer um, e isso é decidido na tela pelo `Mestre` da view.
	Mine bool
}

// attackProposalOf traduz o provisório do regime na faixa, ou nil quando não há
// ataque pendurado.
//
// Os NOMES vêm da fila e não do provisório: a linha pode ter sido renomeada
// entre a rolagem e o desenho, e o nome que a mesa lê é o de agora.
func attackProposalOf(st *live.SessionRuntimeState, userID int64) *attackProposal {
	if st == nil || st.PendingAttack == nil {
		return nil
	}
	pa := st.PendingAttack
	out := &attackProposal{
		Attacker: entryLabel(st, pa.AttackerEntryID), Target: entryLabel(st, pa.TargetEntryID),
		TargetEntryID: pa.TargetEntryID, Weapon: pa.Weapon,
		Tally: attackLine(*pa), Mine: pa.ByUserID == userID,
	}
	switch {
	case pa.Critical:
		out.Verdict, out.Class = "Crítico", "mesa-ataque-critico"
	case pa.Hit:
		out.Verdict, out.Class = "Acertou", "mesa-ataque-acerto"
	default:
		out.Verdict, out.Class = "Errou", "mesa-ataque-erro"
	}
	return out
}

// attackLine escreve a conta inteira do ataque.
//
// Um ataque que ERRA para na comparação: não houve dano, e escrever "0 de dano"
// convidaria a procurar de onde o zero saiu. A RD só aparece quando ela agiu —
// "RD 0" é ruído numa linha que a mesa lê no meio do turno.
//
//	attackLine(...) // "24 vs 17 · 2d8+3 (8+5) · RD 5 → 11"
func attackLine(pa live.PendingAttack) string {
	line := fmt.Sprintf("%d vs %d", pa.Total, pa.Defense)
	if !pa.Hit {
		return line
	}
	sum := 0
	rolled := make([]string, 0, len(pa.Dice))
	for _, d := range pa.Dice {
		sum += d
		rolled = append(rolled, fmt.Sprint(d))
	}
	damage := fmt.Sprintf("%dd%d", len(pa.Dice), pa.Faces)
	if bonus := pa.RawDamage - sum; bonus != 0 {
		damage += fmt.Sprintf("%+d", bonus)
	}
	line += fmt.Sprintf(" · %s (%s)", damage, strings.Join(rolled, "+"))
	// SEM RD AGINDO o total É o dano, e escrever os dois repete o número: a
	// primeira versão desta linha dizia "1d4 (4) = 4 → 4 de dano", e foi olhar a
	// tela que a pegou. A seta só existe quando há de onde para onde.
	if pa.Absorbed == 0 {
		return line + fmt.Sprintf(" = %d de dano", pa.Damage)
	}
	return line + fmt.Sprintf(" = %d · RD %d → %d de dano", pa.RawDamage, pa.Absorbed, pa.Damage)
}

func entryLabel(st *live.SessionRuntimeState, entryID string) string {
	if i := live.FindEntryIndex(st, entryID); i >= 0 {
		return st.Initiative[i].Label
	}
	return "quem saiu da fila"
}
