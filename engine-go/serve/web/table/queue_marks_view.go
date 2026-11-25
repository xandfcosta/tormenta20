package table

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// A MARCA NA FILA vista pelo mestre (ver GLOSSARY, **marca na fila**): o crachá na linha marcada, o Marcar e o Tirar
// na folha dela, e os dados que se rolam na vez da linha, na barra da vez.

// rowMark é uma marca numa linha da fila do mestre: o nome da magia e o que ela rola, quando e até quando.
type rowMark struct {
	Spell  string
	Detail string
	// Chip é o que a ficha da marca diz na folha da linha: o nome e quantos turnos faltam.
	Chip string
}

// dueView é um dado que se rola na vez da linha, na barra do mestre: o Rolar e o "ou rolei" dele.
type dueView struct {
	// EntryID é a linha que está na vez, e Index a posição do dado entre os dela.
	EntryID string
	Index   int
	Name    string
	Detail  string
	// Aria é o nome do Rolar, que diz a magia e de quem é a vez.
	Aria string
	// HandSignal é o sinal do total da mão, e Faces o maior total que o dado tira.
	HandSignal string
	Highest    int
}

// dueHandAttrs são os atributos do "ou rolei" de um dado da vez: o sinal dele e a faixa que o dado tira.
func dueHandAttrs(due dueView) templ.Attributes {
	attrs := templ.Attributes{"min": "1", "inputmode": "numeric", "data-bind:" + due.HandSignal: true}
	if due.Highest > 0 {
		attrs["max"] = strconv.Itoa(due.Highest)
	}
	return attrs
}

// marksOf são as marcas da linha, escritas.
func marksOf(e live.InitiativeEntry) []rowMark {
	var marks []rowMark
	for _, m := range e.Marks {
		detail, chip := []string{partsWritten(m.Parts), live.MomentWritten(m.At)}, m.Spell
		if m.Counted {
			detail = append(detail, turnsLeftWritten(m.TurnsLeft))
			chip += " · " + turnsLeftWritten(m.TurnsLeft)
		}
		marks = append(marks, rowMark{Spell: m.Spell, Detail: strings.Join(detail, " · "), Chip: chip})
	}
	return marks
}

// dueViewsOf são os dados que se rolam na vez da linha que está na vez, com as condições que a mesa já desenha
// nela: as da ficha no PC, as da linha no NPC.
func dueViewsOf(v View, st *live.SessionRuntimeState) []dueView {
	if !st.CountsRounds() || st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) || st.TurnIndex >= len(v.Queue) {
		return nil
	}
	entry := st.Initiative[st.TurnIndex]
	var views []dueView
	for i, due := range live.DueRolls(entry, v.Queue[st.TurnIndex].Conditions) {
		// O MOMENTO sem o nome: a linha da vez já é a de quem está marcado.
		detail := []string{partsWritten(due.Parts), live.MomentWritten(due.At)}
		if due.Who != "" {
			detail = append(detail, "de "+due.Who)
		}
		if due.Save != "" {
			detail = append(detail, fmt.Sprintf("%s · CD %d", due.Save, due.DC))
		}
		if due.Last {
			detail = append(detail, "último turno")
		}
		// O NOME é o da magia, ou o da condição, que não é magia.
		name := due.Spell
		if due.Condition != "" {
			name = due.Condition
		}
		views = append(views, dueView{
			EntryID: entry.ID, Index: i, Name: name, Detail: strings.Join(detail, " · "),
			Aria:       "Rolar " + name + " na vez de " + entry.Label,
			HandSignal: dueHandSignal(i), Highest: highestOf(due.Parts),
		})
	}
	return views
}

// markableOf é a magia que o mestre pode marcar agora e quem a conjurou, ou vazios.
func markableOf(st *live.SessionRuntimeState) (spell, who string) {
	if st.Markable == nil {
		return "", ""
	}
	return st.Markable.Spell, st.Markable.Who
}

// dueSignals declara o "ou rolei" de cada dado da vez VAZIO: sem a declaração, o campo numérico nasce ligado a 0,
// e o Rolar mandaria um total que nenhum dado tira.
func dueSignals(dues []dueView) string {
	declared := make([]string, 0, len(dues))
	for _, due := range dues {
		declared = append(declared, due.HandSignal+": ''")
	}
	return "{" + strings.Join(declared, ", ") + "}"
}

// partsWritten é o dado da marca: "3d6 de ácido".
func partsWritten(parts []live.QueueMarkPart) string {
	written := make([]string, 0, len(parts))
	for _, p := range parts {
		one := p.Dice
		if p.Kind != "" {
			one += " de " + p.Kind
		}
		written = append(written, one)
	}
	return strings.Join(written, " + ")
}

// turnsLeftWritten é quantos turnos a marca contada ainda rola, contando o próximo.
func turnsLeftWritten(left int) string {
	switch {
	case left <= 0:
		return "último turno"
	case left == 1:
		return "mais 1 turno"
	}
	return fmt.Sprintf("mais %d turnos", left)
}

// highestOf é o maior total que o dado da vez tira, o teto do campo da mão; zero quando não se sabe.
func highestOf(parts []live.QueueMarkPart) int {
	if len(parts) != 1 {
		return 0
	}
	dice, err := engine.ParseDiceSum(parts[0].Dice)
	if err != nil {
		return 0
	}
	_, highest := dice.Range()
	return highest
}
