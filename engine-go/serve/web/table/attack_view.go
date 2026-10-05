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
	// Situations são os motivos da Tabela 5-3, em prosa e FORA da conta.
	Situations string
	Tally      string
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
		Attacker: entryLabel(st, pa.AttackerEntryID), Target: targetLabel(*pa, st),
		TargetEntryID: pa.TargetEntryID, Weapon: pa.Weapon,
		Tally: attackLine(*pa), Situations: writtenSituations(*pa),
		Mine: pa.ByUserID == userID,
	}
	if pa.Maneuver != nil {
		// A TINTA DO CRACHÁ É NEUTRA, e é a única que pode ser: as outras três
		// desta faixa — crítico, acerto, erro — são cor de DESFECHO, e a manobra
		// não tem um para colorir.
		out.Verdict, out.Class = maneuverSeal(*pa.Maneuver), "mesa-ataque-comum"
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

// maneuverSeal é o crachá de uma manobra, e ele diz QUAL manobra — nunca quem
// venceu.
//
// Ele já disse "Derrubou" e "não derrubou", lendo um `Won` que o motor calculava.
// Isso saiu: o sistema INFORMA, o mestre DECIDE (ver o `CLAUDE.md` da raiz), e um
// crachá que anuncia o resultado de um embate é exatamente a forma recusada.
//
// O que sobra é o NOME da manobra em curso, que é o que a mesa precisa para ler
// a linha de baixo — "Derrubar · 19 vs 10" se lê sozinho, e "Manobra · 19 vs 10"
// faria perguntar qual.
func maneuverSeal(m live.ManeuverRoll) string {
	if nome, conhecida := maneuverNames[m.Kind]; conhecida {
		return nome
	}
	return "Manobra"
}

// maneuverNames são as cinco da p234 como a mesa as lê, no INFINITIVO.
//
// Eram pares — "Derrubou" / "não derrubou" —, e o passado afirmava um desfecho
// que o sistema não tem mais como saber. O infinitivo diz o que foi TENTADO, que
// é o que de fato aconteceu.
var maneuverNames = map[string]string{
	"agarrar":  "Agarrar",
	"derrubar": "Derrubar",
	"desarmar": "Desarmar",
	"empurrar": "Empurrar",
	"quebrar":  "Quebrar",
}

// maneuverLine escreve a conta do teste OPOSTO, e ela é a resposta inteira: dois
// totais e a diferença entre eles.
//
// SEM VEREDICTO. Quem compara os dois números é o mestre — e a diferença aparece
// porque é a subtração que ele faria de cabeça, não porque alguém venceu.
//
//	maneuverLine(...) // "19 vs 10 · diferença +9 · vitória deixa caído"
func maneuverLine(pa live.PendingAttack) string {
	m := *pa.Maneuver
	// O NOME DA MANOBRA NÃO ENTRA AQUI: o crachá ao lado já o diz, e repeti-lo
	// gastaria o começo da linha — que a 390px é o pedaço mais caro dela.
	line := fmt.Sprintf("%d vs %d · diferença %+d", pa.Total, m.Opposed, m.Margin)
	// O QUE O LIVRO PREVÊ, dito SEMPRE e não só na vitória: o sistema não sabe
	// quem venceu, e esta linha existe justamente para o mestre decidir com a
	// consequência à vista. Nem toda manobra deixa condição — o desarmar derruba
	// um item —, e por isso ela só aparece quando há.
	//
	// O NOME ESCRITO e não o id: "fica caido" é o identificador vazando para a
	// tela. Quem o traduz é o catálogo, que é onde o nome é autorado.
	if m.ConditionOnAWin != "" {
		line += " · vitória deixa " + strings.ToLower(book.ConditionName(m.ConditionOnAWin))
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
		return line + fmt.Sprintf(" = %d de dano%s", pa.Damage, writtenNonLethal(pa))
	}
	return line + fmt.Sprintf(" = %d · RD %d → %d de dano%s",
		pa.RawDamage, pa.Absorbed, pa.Damage, writtenNonLethal(pa))
}

// targetLabel é o nome do que foi atacado, e ele tem DUAS procedências.
//
// A criatura vem da FILA e não do provisório, porque a linha pode ter sido
// renomeada entre a rolagem e o desenho — o nome que a mesa lê é o de agora. O
// OBJETO não tem linha, então o nome dele viaja no provisório; é a única coisa
// que a faixa recebe sobre ele, e sem isso ela diria "acertou" sem dizer o quê.
func targetLabel(pa live.PendingAttack, st *live.SessionRuntimeState) string {
	if pa.TargetTokenID != "" {
		return pa.TargetLabel
	}
	return entryLabel(st, pa.TargetEntryID)
}

func entryLabel(st *live.SessionRuntimeState, entryID string) string {
	if i := live.FindEntryIndex(st, entryID); i >= 0 {
		return st.Initiative[i].Label
	}
	return "quem saiu da fila"
}

// writtenSituations são as linhas da Tabela 5-3 que valeram, em prosa.
//
// FORA DA CONTA, e isso foi o OLHO a 390px que decidiu. Dentro dela o parêntese
// ficava em monoespaçado no meio da aritmética e, com três situações, partia a
// linha em três — empurrando o dano, que é o que a mesa veio ler, para o fim da
// terceira. Em prosa ele é ~30% mais estreito, e a quebra cai no lugar natural:
// entre o número e o motivo.
//
// Elas existem pela mesma razão do "20 natural acerta" que a `attackLine`
// nomeia: um número que DESMENTE o que a mesa sabe vem com o motivo. A Defesa
// não é a da ficha quando o alvo está coberto, e a camuflagem desmente a
// própria comparação — o total bateu a Defesa e o ataque errou (p238).
func writtenSituations(pa live.PendingAttack) string {
	return strings.Join(pa.Situations, ", ")
}

// writtenNonLethal qualifica o dano quando ele não mata (p236).
//
// COLADO NO DANO e não na prosa das situações, ao contrário da Tabela 5-3: a
// situação explica a COMPARAÇÃO e esta qualifica o NÚMERO — "9 de dano" e "9 de
// dano não letal" levam a mesas diferentes, e o mestre que não souber vai pedir
// um teste de Constituição que a regra não manda fazer.
//
// Só quando é TUDO: o livro não dá meio-termo — ou a arma é Piedosa e todo o
// dano dela é não letal, ou não é.
func writtenNonLethal(pa live.PendingAttack) string {
	if pa.NonLethal > 0 && pa.NonLethal >= pa.Damage {
		return " " + nonLethalWords
	}
	return ""
}

// nonLethalWords carrega um espaço INQUEBRÁVEL entre as duas palavras, e o olho
// a 390px foi quem pediu: numa linha com RD e situação, a conta quebrava em
// "→ 11 de dano não" / "letal", e "dano não" sozinho no fim de uma linha diz o
// contrário do que a regra diz.
const nonLethalWords = "não letal"
