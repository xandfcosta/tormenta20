package live

import (
	"fmt"
	"slices"
)

// A MARCA NA FILA (ver GLOSSARY, **marca na fila**): o que a magia deixa na criatura atingida e rola na vez DELA. O
// mestre diz quem falhou na resistência, então é ele que marca a linha; a mesa guarda a marca, conta os turnos e
// diz o que se rola na vez da linha marcada. Quem rola e aplica é o mestre.

// ConditionBurning é a condição Em Chamas do livro (p394). O dado dela é do motor; o regime a conhece pelo id
// porque é aqui que a marca que SÃO as chamas toma o lugar dele.
const ConditionBurning = "em-chamas"

// ConditionDue é o dado que uma condição rola no início da vez, como o motor o diz (`engine.ConditionTurnRolls`):
// a regra de qual condição rola o quê mora lá, e o regime só a põe na fila.
type ConditionDue struct {
	ID, Name, Dice, Kind string
}

// QueueMark é a marca de uma magia ou de um perigo numa linha da fila: o dado já com os aumentos da conjuração, o
// momento e a duração.
type QueueMark struct {
	// Source é a ORIGEM da marca — o nome da magia ou o do perigo —, e Who o de quem conjurou, vazio no perigo,
	// que não tem conjurador. A chave JSON continua "spell" para a sessão gravada antes do perigo seguir lendo.
	Source string `json:"spell"`
	Who    string `json:"who"`
	// At é "inicio" ou "fim" do turno de quem está marcado.
	At    string          `json:"at"`
	Parts []QueueMarkPart `json:"parts"`
	// Bonuses são as parcelas fixas que os poderes de quem conjurou somam ao dado da marca (o Aspecto do Inverno).
	Bonuses []SpellCastBonus `json:"bonuses,omitempty"`
	// Save e DC são a resistência que quem está marcado repete antes do dado (a Vontade da Ilusão Lacerante),
	// vazios no resto.
	Save string `json:"save,omitempty"`
	DC   int    `json:"dc,omitempty"`
	// Counted diz que a marca dura turnos contados, e TurnsLeft quantos ainda vêm: os dois da Flecha Ácida, as 5
	// rodadas do Círculo. Sem contagem, ela fica até o mestre a tirar.
	Counted   bool `json:"counted,omitempty"`
	TurnsLeft int  `json:"turnsLeft,omitempty"`
	// CountedAt é o `TurnsTaken` em que a marca gastou o último turno: voltar a vez e passar de novo entra no
	// MESMO turno, com o mesmo número, e ele não se gasta duas vezes.
	CountedAt int `json:"countedAt,omitempty"`
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
	// Source é a origem da marca (a magia ou o perigo), e Who quem conjurou, vazio no perigo.
	Source string
	Who    string
	// At é "inicio" ou "fim" do turno, e Target de quem é a vez. Condition é o nome da condição que rola (Em
	// Chamas) e ConditionID o id dela, e nela `Source` e `Who` ficam vazios: condição não é magia.
	At          string
	Target      string
	Condition   string
	ConditionID string
	Parts       []QueueMarkPart
	Bonuses     []SpellCastBonus
	Save        string
	DC          int
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
	entry.Marks = slices.DeleteFunc(entry.Marks, func(m QueueMark) bool { return m.Source == mark.Source })
	entry.Marks = append(entry.Marks, mark)
	return nil
}

// UnmarkEntry tira da linha a marca que veio desta origem.
func UnmarkEntry(st *SessionRuntimeState, entryID, source string) error {
	entry := findEntry(st, entryID)
	if entry == nil {
		return fmt.Errorf("a linha %q não está na fila", entryID)
	}
	before := len(entry.Marks)
	entry.Marks = slices.DeleteFunc(entry.Marks, func(m QueueMark) bool { return m.Source == source })
	if len(entry.Marks) == before {
		return fmt.Errorf("%s não está marcado com %s", entry.Label, source)
	}
	return nil
}

// countTheMarksOfTheTurn gasta um turno de cada marca contada de quem acabou de entrar na vez, e tira a que já
// tinha gastado o último: a Flecha marcada com dois rola na primeira e na segunda vez do alvo, e sai na terceira.
// O turno entrado de novo depois do "Turno anterior" não se gasta outra vez (`CountedAt`).
func countTheMarksOfTheTurn(st *SessionRuntimeState) {
	if st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return
	}
	entry := &st.Initiative[st.TurnIndex]
	again := func(m QueueMark) bool { return m.CountedAt == st.TurnsTaken }
	entry.Marks = slices.DeleteFunc(entry.Marks, func(m QueueMark) bool { return m.Counted && m.TurnsLeft <= 0 && !again(m) })
	for i := range entry.Marks {
		if entry.Marks[i].Counted && !again(entry.Marks[i]) {
			entry.Marks[i].TurnsLeft--
			entry.Marks[i].CountedAt = st.TurnsTaken
		}
	}
	if len(entry.Marks) == 0 {
		entry.Marks = nil
	}
}

// DueRolls são os dados que se rolam na vez da linha: o de cada marca, no início ou no fim do turno, e o de cada
// condição, no início — a menos que uma marca SEJA as chamas, e aí o 1d6 do Em Chamas dá lugar ao dado dela (p196).
// Os dados das condições vêm de quem pergunta, porque a do PC mora na ficha e a do NPC na linha, e o que cada uma
// rola é do motor.
//
//	live.DueRolls(entry, nil) // [{Source: "Flecha Ácida", At: "inicio", Target: "Ogro", …}]
func DueRolls(entry InitiativeEntry, conditions []ConditionDue) []DueRoll {
	var due []DueRoll
	burning := false
	for _, m := range entry.Marks {
		burning = burning || m.Burning
		due = append(due, DueRoll{
			Source: m.Source, Who: m.Who, At: m.At, Target: entry.Label, Bonuses: m.Bonuses,
			Parts: m.Parts, Save: m.Save, DC: m.DC, Last: m.Counted && m.TurnsLeft == 0,
		})
	}
	for _, c := range conditions {
		if burning && c.ID == ConditionBurning {
			continue
		}
		due = append(due, DueRoll{
			Condition: c.Name, ConditionID: c.ID, At: "inicio", Target: entry.Label,
			Parts: []QueueMarkPart{{Dice: c.Dice, Kind: c.Kind}},
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
