package table

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/live"
	"t20engine/serve/web/ds"
)

// O CARTÃO DO GOLPE EM DOIS TEMPOS (ALE-536): o que cada tempo desenha, e em
// que NOTAÇÃO. A notação da mesa põe entre parênteses o que o dado tirou —
// `d20 (19) + 8 = 27`, `2d8 (5, 3) + 3 = 11` —, para o número sorteado nunca se
// confundir com o bônus que a ficha soma.

// Os tempos do cartão. O golpe passa por julgar → esperar o dano → confirmar,
// ou acaba no erro; a manobra tem um tempo só, porque não rola dano.
const (
	stageJudge    = "judge"
	stageDamage   = "damage"
	stageConfirm  = "confirm"
	stageMiss     = "miss"
	stageManeuver = "maneuver"
)

// stageOf diz em que tempo o provisório está.
func stageOf(pa live.PendingAttack) string {
	switch {
	case pa.Maneuver != nil:
		return stageManeuver
	case pa.Ruling == "":
		return stageJudge
	case pa.Ruling == live.RulingMiss:
		return stageMiss
	case !pa.DamageRolled:
		return stageDamage
	}
	return stageConfirm
}

// duelView é a cabeça do cartão do mestre: a rolagem à esquerda, o que ela
// enfrenta à direita, e o selo da decisão no meio.
type duelView struct {
	RollLabel    string
	Roll         int
	RollSum      string
	AgainstLabel string
	Against      int
	AgainstSum   string
	Seal         string
	Tone         string
}

// verdictButton é um dos três botões do mestre no tempo de julgar. O que o
// livro sugere é o botão cheio.
type verdictButton struct {
	Label     string
	Path      string
	Suggested bool
}

// duelRow é uma linha de conta abaixo do duelo. Text é a linha que é FRASE e
// não número — a consequência da manobra —, e ocupa a largura toda.
type duelRow struct {
	Label string
	Sum   string
	Value string
	Final bool
	Text  bool
}

// handGroup é um grupo de dados da fórmula na entrada da mão: "2d8 ( _ _ )".
type handGroup struct {
	Label   string
	Type    string
	Faces   int
	Signals []string
}

// rollNotation é a rolagem de um d20 com o bônus: "d20 (19) + 8".
func rollNotation(roll, total int) string {
	return "d20 (" + strconv.Itoa(roll) + ")" + signedTerm(total-roll)
}

// signedTerm é o termo somado, com o sinal separado: " + 8", " − 1", ou nada.
func signedTerm(n int) string {
	switch {
	case n > 0:
		return " + " + strconv.Itoa(n)
	case n < 0:
		return " − " + strconv.Itoa(-n)
	}
	return ""
}

// diceNotation é um grupo rolado: "2d8 (5, 3)", ou "1d6 fogo (4)".
func diceNotation(dice []int, faces int, kind string) string {
	rolled := make([]string, 0, len(dice))
	for _, d := range dice {
		rolled = append(rolled, strconv.Itoa(d))
	}
	group := fmt.Sprintf("%dd%d", len(dice), faces)
	if kind != "" {
		group += " " + kind
	}
	return group + " (" + strings.Join(rolled, ", ") + ")"
}

// damageSum é o dano rolado na notação, sem o total: "2d8 (5, 3) + 1d6 fogo (4) + 3".
func damageSum(pa live.PendingAttack) string {
	rolled := 0
	for _, d := range pa.Dice {
		rolled += d
	}
	parts := []string{diceNotation(pa.Dice, pa.Faces, "")}
	for _, extra := range pa.ExtraDice {
		for _, d := range extra.Dice {
			rolled += d
		}
		parts = append(parts, diceNotation(extra.Dice, extra.Faces, extra.Type))
	}
	return strings.Join(parts, " + ") + signedTerm(pa.RawDamage-rolled)
}

// toRollLine é o dano a rolar depois da decisão: "3d8 + 1d6 + 4".
func toRollLine(toRoll live.DamageToRoll) string {
	parts := make([]string, 0, len(toRoll.Dice))
	for _, group := range toRoll.Dice {
		parts = append(parts, fmt.Sprintf("%dd%d", group.Count, group.Faces))
	}
	return strings.Join(parts, " + ") + signedTerm(toRoll.Bonus)
}

// handGroupsOf são os campos da mão, um por dado, numerados na ordem da
// fórmula — a mesma em que o servidor os lê (`handDamageSignal`).
func handGroupsOf(toRoll live.DamageToRoll) []handGroup {
	groups := make([]handGroup, 0, len(toRoll.Dice))
	next := 1
	for _, group := range toRoll.Dice {
		hand := handGroup{Label: fmt.Sprintf("%dd%d", group.Count, group.Faces), Type: group.Type, Faces: group.Faces}
		for range group.Count {
			hand.Signals = append(hand.Signals, handDamageSignal(next))
			next++
		}
		groups = append(groups, hand)
	}
	return groups
}

// critLabel é o crítico DA ARMA usada: "Crítico ×3".
func critLabel(pa live.PendingAttack) string {
	multiplier := pa.CritMult
	if multiplier <= 0 {
		multiplier = 2
	}
	return "Crítico ×" + strconv.Itoa(multiplier)
}

// rulingSaid é a decisão do mestre como o jogador a lê: "crítico ×2".
func rulingSaid(pa live.PendingAttack) string {
	switch pa.Ruling {
	case live.RulingMiss:
		return "errou"
	case live.RulingHit:
		return "acertou"
	case live.RulingCritical:
		return strings.ToLower(critLabel(pa))
	}
	return ""
}

// verdictsOf são os três botões do mestre, com a sugestão do livro cheia.
func verdictsOf(pa live.PendingAttack) []verdictButton {
	return []verdictButton{
		{Label: "Errou", Path: "julgar/errou", Suggested: !pa.Hit},
		{Label: "Acertou", Path: "julgar/acertou", Suggested: pa.Hit && !pa.Critical},
		{Label: critLabel(pa), Path: "julgar/critico", Suggested: pa.Critical},
	}
}

// suggestionHint diz POR QUE o livro sugere o que sugere, com o número que
// decide: a margem da arma, o natural das pontas (p221), a camuflagem (p239).
func suggestionHint(pa live.PendingAttack) string {
	switch {
	case pa.Critical:
		margin := "na margem 20"
		if pa.CritRange > 0 && pa.CritRange < 20 {
			margin = fmt.Sprintf("na margem %d–20", pa.CritRange)
		}
		return fmt.Sprintf("O livro sugere crítico: %d natural, %s da arma.", pa.Roll, margin)
	case pa.Hit && pa.Roll == 20 && pa.Total < pa.Defense:
		return "O livro sugere acerto: 20 natural sempre acerta (p221)."
	case pa.Hit:
		return fmt.Sprintf("O livro sugere acerto: %d contra Defesa %d.", pa.Total, pa.Defense)
	case pa.Roll == 1 && pa.Total >= pa.Defense:
		return "O livro sugere erro: 1 natural sempre erra (p221)."
	case pa.Total >= pa.Defense:
		return "O livro sugere erro: a camuflagem desfez o acerto (p239)."
	}
	return fmt.Sprintf("O livro sugere erro: %d contra Defesa %d.", pa.Total, pa.Defense)
}

// strikeDuel é o duelo do golpe: a rolagem contra a Defesa, e o selo depois da
// decisão.
func strikeDuel(pa live.PendingAttack) duelView {
	duel := duelView{
		RollLabel: "Ataque", Roll: pa.Total, RollSum: rollNotation(pa.Roll, pa.Total),
		AgainstLabel: "Defesa", Against: pa.Defense,
	}
	switch pa.Ruling {
	case live.RulingHit:
		duel.Seal, duel.Tone = "Acertou", "good"
	case live.RulingCritical:
		duel.Seal, duel.Tone = critLabel(pa), "good"
	}
	return duel
}

// maneuverDuel é o teste OPOSTO da manobra (p234): os dois lados e a leitura
// que os números sugerem.
func maneuverDuel(pa live.PendingAttack) (duelView, string) {
	m := *pa.Maneuver
	seal, tone, note := maneuverVerdict(pa)
	return duelView{
		RollLabel: maneuverSeal(m), Roll: pa.Total, RollSum: rollNotation(pa.Roll, pa.Total),
		AgainstLabel: "Resiste", Against: m.Opposed, AgainstSum: rollNotation(m.OpposedRoll, m.Opposed),
		Seal: seal, Tone: tone,
	}, note
}

// maneuverConsequence é o que o livro prevê para a vitória, em frase: a
// condição e o efeito extra da margem de 5 (p234).
func maneuverConsequence(pa live.PendingAttack) []duelRow {
	m := *pa.Maneuver
	var said []string
	if m.ConditionOnAWin != "" {
		said = append(said, "deixa "+strings.ToLower(book.ConditionName(m.ConditionOnAWin)))
	}
	if extra := maneuverExtra(m.Kind, m.Margin); extra != "" {
		said = append(said, "por 5+, "+extra)
	}
	if len(said) == 0 {
		return nil
	}
	return []duelRow{{Label: "Consequência", Value: strings.Join(said, "; "), Text: true}}
}

// damageRows são as linhas de conta do dano rolado: o dano, a RD do alvo e o
// final. Sem RD agindo, o dano É o final, e a linha não se repete.
func damageRows(pa live.PendingAttack) []duelRow {
	rows := []duelRow{{Label: "Dano", Sum: damageSum(pa), Value: strconv.Itoa(pa.RawDamage)}}
	if pa.Absorbed == 0 {
		return rows
	}
	return append(rows,
		duelRow{Label: "RD", Value: "−" + strconv.Itoa(pa.Absorbed)},
		duelRow{Label: "Dano final", Value: strconv.Itoa(pa.Damage), Final: true})
}

// damageNotes são o que qualifica o dano para o mestre: não letal (p236), a
// chance de ignorar o crítico (p166, p338) e o que o livro reserva ao golpe.
func damageNotes(pa live.PendingAttack) []string {
	var notes []string
	// O NÃO LETAL diz de quê: "Não letal" sozinho embaixo das linhas do dano
	// não diz o que deixa de matar.
	if writtenNonLethal(pa) != "" {
		notes = append(notes, "Dano não letal (p236)")
	}
	for _, note := range []string{writtenCritChance(pa), writtenForeseen(pa)} {
		if note = strings.TrimPrefix(strings.TrimSpace(note), "| "); note != "" {
			notes = append(notes, capitalize(note))
		}
	}
	return notes
}

// attackSheet é o diálogo do corpo do cartão, que a cabeça abre no celular.
const attackSheet = "attack-sheet"

// gmAttackBadge é o crachá do cartão do mestre, que diz o que o tempo pede dele.
func gmAttackBadge(stage string) string {
	switch stage {
	case stageJudge:
		return "Para julgar"
	case stageDamage:
		return "Esperando o dano"
	}
	return "Para confirmar"
}

// playerAttackBadge é o crachá do cartão do jogador.
func playerAttackBadge(a *attackProposal) string {
	if !a.Mine {
		return "Ataque"
	}
	switch a.Stage {
	case stageDamage:
		return "Role o dano"
	case stageConfirm:
		return "Dano rolado"
	case stageMiss:
		return "Errou"
	}
	return "Rascunho"
}

// verdictVariant enche o botão que o livro sugere.
func verdictVariant(b verdictButton) ds.ButtonVariant {
	if b.Suggested {
		return ds.ButtonPrimary
	}
	return ds.ButtonSecondary
}

// handSignalsDeclaration declara os campos da mão vazios: `{damage_die_1: ”}`.
func handSignalsDeclaration(hand []handGroup) string {
	var decl []string
	for _, group := range hand {
		for _, signal := range group.Signals {
			decl = append(decl, signal+": ''")
		}
	}
	return "{" + strings.Join(decl, ", ") + "}"
}

// handDieLabel é o nome acessível de um campo da mão: "d8 2 de 3".
func handDieLabel(group handGroup, n int) string {
	label := fmt.Sprintf("d%d %d de %d", group.Faces, n+1, len(group.Signals))
	if group.Type != "" {
		label += " de " + group.Type
	}
	return label
}
