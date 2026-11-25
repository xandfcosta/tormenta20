package live

import (
	"fmt"
	"slices"
)

// A MARCA NA FILA (ver GLOSSARY, **marca na fila**): o que a magia deixa na criatura atingida e rola na vez DELA. O
// mestre diz quem falhou na resistência, então é ele que marca a linha; a mesa guarda a marca, conta os turnos e
// diz o que se rola na vez da linha marcada. Quem rola e aplica é o mestre.

// ConditionBurning é a condição Em Chamas do livro (p394), que rola 1d6 de fogo no início da vez de quem a tem.
const ConditionBurning = "em-chamas"

// QueueMark é a marca de uma magia numa linha da fila: o dado já com os aumentos da conjuração, o momento e a
// duração.
type QueueMark struct {
	// Spell é o nome da magia, e Who o de quem a conjurou.
	Spell string `json:"spell"`
	Who   string `json:"who"`
	// At é "inicio" ou "fim" do turno de quem está marcado.
	At    string          `json:"at"`
	Parts []QueueMarkPart `json:"parts"`
	// Save e DC são a resistência que quem está marcado repete antes do dado (a Vontade da Ilusão Lacerante),
	// vazios no resto.
	Save string `json:"save,omitempty"`
	DC   int    `json:"dc,omitempty"`
	// Counted diz que a marca dura turnos contados, e TurnsLeft quantos ainda vêm: os dois da Flecha Ácida, as 5
	// rodadas do Círculo. Sem contagem, ela fica até o mestre a tirar.
	Counted   bool `json:"counted,omitempty"`
	TurnsLeft int  `json:"turnsLeft,omitempty"`
	// Burning diz que a marca SÃO as chamas da condição Em Chamas, com o dado dela no lugar do 1d6 (p196).
	Burning bool `json:"burning,omitempty"`
}

// QueueMarkPart é uma parcela do dado da marca, ainda não rolada.
type QueueMarkPart struct {
	Dice string `json:"dice"`
	Kind string `json:"kind,omitempty"`
	Note string `json:"note,omitempty"`
}

// DueRoll é um dado que se rola na vez de uma linha: o de uma marca, ou o 1d6 da condição Em Chamas. O gesto do
// mestre o pede pela posição na lista.
type DueRoll struct {
	Spell string
	Who   string
	// When é o momento escrito para a cabeça da faixa: "início do turno de Ogro".
	When  string
	Parts []QueueMarkPart
	Save  string
	DC    int
	// Last diz que é o último turno da marca contada.
	Last bool
}

// MarkEntry põe na linha a marca da última magia conjurada que deixa marca (`Markable`). A mesma magia marcada de
// novo troca a marca antiga: é a conjuração nova, com os aumentos dela.
func MarkEntry(st *SessionRuntimeState, entryID string) error {
	if st.Markable == nil {
		return fmt.Errorf("nenhuma magia conjurada nesta sessão deixa marca na criatura atingida")
	}
	entry := findEntry(st, entryID)
	if entry == nil {
		return fmt.Errorf("a linha %q não está na fila", entryID)
	}
	mark := *st.Markable
	mark.Parts = slices.Clone(mark.Parts)
	entry.Marks = slices.DeleteFunc(entry.Marks, func(m QueueMark) bool { return m.Spell == mark.Spell })
	entry.Marks = append(entry.Marks, mark)
	return nil
}

// UnmarkEntry tira da linha a marca da magia.
func UnmarkEntry(st *SessionRuntimeState, entryID, spell string) error {
	entry := findEntry(st, entryID)
	if entry == nil {
		return fmt.Errorf("a linha %q não está na fila", entryID)
	}
	before := len(entry.Marks)
	entry.Marks = slices.DeleteFunc(entry.Marks, func(m QueueMark) bool { return m.Spell == spell })
	if len(entry.Marks) == before {
		return fmt.Errorf("%s não está marcado com %s", entry.Label, spell)
	}
	return nil
}

// countTheMarksOfTheTurn gasta um turno de cada marca contada de quem acabou de entrar na vez, e tira a que já
// tinha gastado o último: a Flecha marcada com dois rola na primeira e na segunda vez do alvo, e sai na terceira.
func countTheMarksOfTheTurn(st *SessionRuntimeState) {
	if st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return
	}
	entry := &st.Initiative[st.TurnIndex]
	entry.Marks = slices.DeleteFunc(entry.Marks, func(m QueueMark) bool { return m.Counted && m.TurnsLeft <= 0 })
	for i := range entry.Marks {
		if entry.Marks[i].Counted {
			entry.Marks[i].TurnsLeft--
		}
	}
	if len(entry.Marks) == 0 {
		entry.Marks = nil
	}
}

// DueRolls são os dados que se rolam na vez da linha: o de cada marca, no início ou no fim do turno, e o 1d6 de
// fogo da condição Em Chamas (p394) — a menos que uma marca SEJA as chamas, e aí o dado é o dela. As condições vêm
// de quem pergunta, porque a do PC mora na ficha e a do NPC na linha.
//
//	live.DueRolls(entry, entry.Conditions) // [{Spell: "Flecha Ácida", When: "início do turno de Ogro", …}]
func DueRolls(entry InitiativeEntry, conditions []string) []DueRoll {
	var due []DueRoll
	burning := false
	for _, m := range entry.Marks {
		burning = burning || m.Burning
		due = append(due, DueRoll{
			Spell: m.Spell, Who: m.Who, When: MomentWritten(m.At) + " de " + entry.Label,
			Parts: m.Parts, Save: m.Save, DC: m.DC, Last: m.Counted && m.TurnsLeft == 0,
		})
	}
	if !burning && slices.Contains(conditions, ConditionBurning) {
		due = append(due, DueRoll{
			Spell: "Em Chamas", Who: entry.Label, When: MomentWritten("inicio"),
			Parts: []QueueMarkPart{{Dice: "1d6", Kind: "fogo"}},
		})
	}
	return due
}

// MomentWritten é o momento como a mesa o escreve: "início do turno" ou "fim do turno".
func MomentWritten(at string) string {
	if at == "fim" {
		return "fim do turno"
	}
	return "início do turno"
}

func findEntry(st *SessionRuntimeState, entryID string) *InitiativeEntry {
	for i := range st.Initiative {
		if st.Initiative[i].ID == entryID {
			return &st.Initiative[i]
		}
	}
	return nil
}
