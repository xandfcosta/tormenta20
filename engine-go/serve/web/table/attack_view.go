package table

import (
	"fmt"
	"strings"

	"t20engine/domain/book"
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
	if pa.Maneuver != nil {
		out.Verdict, out.Class = maneuverVerdict(*pa.Maneuver)
		out.Tally = maneuverLine(*pa)
		return out
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

// maneuverNames são as cinco da p234 como a mesa as lê. O crachá diz a MANOBRA e
// não "venceu": "Derrubou" responde o que aconteceu, e "Venceu" faria a mesa
// perguntar o quê.
var maneuverNames = map[string][2]string{
	"agarrar":  {"Agarrou", "não agarrou"},
	"derrubar": {"Derrubou", "não derrubou"},
	"desarmar": {"Desarmou", "não desarmou"},
	"empurrar": {"Empurrou", "não empurrou"},
	"quebrar":  {"Quebrou", "não quebrou"},
}

// maneuverVerdict é a palavra e a tinta do crachá de uma manobra.
//
// O EMPATE DE BÔNUS IGUAIS tem crachá próprio, e não o de derrota: a p234 manda
// rolar de novo, e anunciar "não derrubou" seria dar por perdida uma manobra que
// a regra não decidiu. A tinta é a do erro porque nada aconteceu ainda, e a
// palavra é que diz o que falta.
func maneuverVerdict(m live.ManeuverRoll) (string, string) {
	if m.AnotherRoll {
		return "Empate", "mesa-ataque-erro"
	}
	nomes, conhecida := maneuverNames[m.Kind]
	if !conhecida {
		nomes = [2]string{"Venceu", "perdeu"}
	}
	if m.Won {
		return nomes[0], "mesa-ataque-acerto"
	}
	return capitalize(nomes[1]), "mesa-ataque-erro"
}

// maneuverLine escreve a conta do teste OPOSTO.
//
// A MARGEM aparece só quando ela faz diferença: cinco pontos ou mais dão efeito
// extra ao derrubar e ao desarmar (p234), e escrever "por 2" numa vitória
// apertada seria número sem consequência no meio do turno. O empate escreve o
// que a regra pede em vez de um número.
//
//	maneuverLine(...) // "derrubar · 19 vs 10 · por 9"
func maneuverLine(pa live.PendingAttack) string {
	m := *pa.Maneuver
	line := fmt.Sprintf("%s · %d vs %d", m.Kind, pa.Total, m.Opposed)
	switch {
	case m.AnotherRoll:
		return line + " · bônus iguais, role de novo (p234)"
	case m.Won && m.Margin >= 5:
		line += fmt.Sprintf(" · por %d, e cinco ou mais dão efeito extra", m.Margin)
	}
	// A CONDIÇÃO que a confirmação vai deixar, e ela é dita ANTES: o mestre
	// decide com ela à vista, e descobrir depois o que o clique fez é o que a
	// faixa existe para evitar. Nem toda manobra deixa uma — o desarmar derruba
	// um item —, e por isso ela só aparece quando há.
	if m.Won && m.Imposes != "" {
		// O NOME ESCRITO e não o id: "fica caido" é o identificador vazando para
		// a tela. Quem o traduz é o catálogo, que é onde o nome da condição é
		// autorado — o `GLOSSARY.md` manda o id em inglês no código e o texto em
		// português na tela, e uma condição não é exceção.
		line += " · fica " + strings.ToLower(book.ConditionName(m.Imposes))
	}
	return line
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
	// O NATURAL QUE DESMENTE A COMPARAÇÃO é nomeado (p221): "ERROU · 14 vs 13"
	// é a conta certa com cara de regra quebrada. Quando a comparação já conta a
	// história, o natural não entra — seria ruído no meio do turno.
	switch {
	case pa.Roll == 1 && pa.Total >= pa.Defense:
		line += " · 1 natural erra"
	case pa.Roll == 20 && pa.Total < pa.Defense:
		line += " · 20 natural acerta"
	}
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
