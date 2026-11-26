package live

import (
	"fmt"
	"slices"
	"strings"
)

// O TESTE ESTENDIDO (ver GLOSSARY, **teste estendido**, e o `CLAUDE.md`, "O que ele NÃO faz"): a tarefa da p222-223
// que se vence acumulando os sucessos exigidos antes de três falhas. Na mesa ele é um CONTADOR que o mestre abre e
// alimenta: o jogador rola a perícia como em qualquer teste, sem CD nem veredito, e o mestre marca cada resultado.
// O app soma e diz o que o livro prevê ao chegar lá — o exigido alcançado, ou a falha total —, e não compara a
// rolagem com CD nem aplica nada.

// ExtendedFailureLimit são as falhas que fazem a falha total (p222).
const ExtendedFailureLimit = 3

// As marcas que o mestre põe no contador: o sucesso, o sucesso que conta dois (o por 10 ou mais da avalanche,
// p320) e a falha.
const (
	ExtendedSuccess = "sucesso"
	ExtendedDouble  = "duplo"
	ExtendedFailure = "falha"
)

// Os desfechos que o livro prevê para um contador, vazios enquanto ele corre.
const (
	ExtendedReached      = "alcancado"
	ExtendedTotalFailure = "falha-total"
)

// ExtendedTest é o teste estendido aberto na mesa: o que é, os sucessos exigidos, de quem é a contagem e as marcas.
//
// Quem muda o teste troca o PONTEIRO, e não o que ele aponta: o `CloneState` é raso, e a marca posta no rascunho
// de uma gravação que falhou ficaria no estado de antes.
type ExtendedTest struct {
	Name     string `json:"name"`
	Required int    `json:"required"`
	// Who são as linhas da fila que têm contagem própria, como estavam ao abrir; vazio no teste do grupo, que
	// tem uma contagem só.
	Who []ExtendedWho `json:"who,omitempty"`
	// Marks são as marcas na ordem em que o mestre as pôs: a contagem sai delas, e desfazer é tirar a última.
	Marks []ExtendedMark `json:"marks,omitempty"`
}

// ExtendedWho é uma linha da fila com contagem própria: o id e o nome de quando o teste abriu.
type ExtendedWho struct {
	EntryID string `json:"entryId"`
	Label   string `json:"label"`
}

// ExtendedMark é uma marca do mestre: de quem (vazio no grupo) e qual.
type ExtendedMark struct {
	EntryID string `json:"entryId,omitempty"`
	Kind    string `json:"kind"`
}

// ExtendedTally é uma contagem: os sucessos, as falhas e o desfecho que o livro prevê, vazio enquanto corre.
type ExtendedTally struct {
	Successes int
	Failures  int
	Outcome   string
}

// TallyOf é a contagem de uma linha, ou a do grupo com o id vazio.
//
//	st.ExtendedTest.TallyOf("") // {Successes: 3, Failures: 1}
func (t ExtendedTest) TallyOf(entryID string) ExtendedTally {
	var tally ExtendedTally
	for _, m := range t.Marks {
		if m.EntryID != entryID {
			continue
		}
		switch m.Kind {
		case ExtendedSuccess:
			tally.Successes++
		case ExtendedDouble:
			tally.Successes += 2
		case ExtendedFailure:
			tally.Failures++
		}
	}
	switch {
	case tally.Successes >= t.Required:
		tally.Outcome = ExtendedReached
	case tally.Failures >= ExtendedFailureLimit:
		tally.Outcome = ExtendedTotalFailure
	}
	return tally
}

// OpenExtendedTest abre o teste estendido. `each` dá uma contagem a cada linha da fila de agora.
func OpenExtendedTest(st *SessionRuntimeState, name string, required int, each bool) error {
	if st.ExtendedTest != nil {
		return fmt.Errorf("o teste estendido %q está aberto: encerre-o antes de abrir outro", st.ExtendedTest.Name)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("diga o que é o teste estendido (\"Escalar um paredão\")")
	}
	if required < 1 {
		return fmt.Errorf("os sucessos exigidos são %d; o livro pede 3, 5 ou 7 (p223)", required)
	}
	test := &ExtendedTest{Name: name, Required: required}
	if each {
		if len(st.Initiative) == 0 {
			return fmt.Errorf("a contagem de cada um é por linha da fila, e a fila está vazia")
		}
		for _, e := range st.Initiative {
			test.Who = append(test.Who, ExtendedWho{EntryID: e.ID, Label: e.Label})
		}
	}
	st.ExtendedTest = test
	return nil
}

// MarkExtendedTest põe a marca do mestre na contagem da linha (ou na do grupo, com o id vazio). A contagem que já
// chegou a um desfecho não recebe mais marca: o que vem depois é do mestre.
func MarkExtendedTest(st *SessionRuntimeState, entryID, kind string) error {
	t := st.ExtendedTest
	if t == nil {
		return fmt.Errorf("não há teste estendido aberto")
	}
	if kind != ExtendedSuccess && kind != ExtendedDouble && kind != ExtendedFailure {
		return fmt.Errorf("a marca %q não existe: é %q, %q ou %q", kind, ExtendedSuccess, ExtendedDouble, ExtendedFailure)
	}
	whose := "do grupo"
	if len(t.Who) == 0 && entryID != "" {
		return fmt.Errorf("%s é do grupo, e a marca veio para a linha %q", t.Name, entryID)
	}
	if len(t.Who) > 0 {
		i := slices.IndexFunc(t.Who, func(w ExtendedWho) bool { return w.EntryID == entryID })
		if i < 0 {
			return fmt.Errorf("%s conta cada um da fila de quando abriu, e a linha %q não estava nela", t.Name, entryID)
		}
		whose = "de " + t.Who[i].Label
	}
	switch t.TallyOf(entryID).Outcome {
	case ExtendedReached:
		return fmt.Errorf("a contagem %s em %s já chegou ao exigido", whose, t.Name)
	case ExtendedTotalFailure:
		return fmt.Errorf("a contagem %s em %s já chegou à falha total", whose, t.Name)
	}
	next := *t
	next.Marks = append(slices.Clone(t.Marks), ExtendedMark{EntryID: entryID, Kind: kind})
	st.ExtendedTest = &next
	return nil
}

// UndoExtendedMark tira a última marca: o toque errado no meio da cena não vira um sucesso que ninguém tira.
func UndoExtendedMark(st *SessionRuntimeState) error {
	t := st.ExtendedTest
	if t == nil || len(t.Marks) == 0 {
		return fmt.Errorf("não há marca para desfazer")
	}
	next := *t
	next.Marks = slices.Clone(t.Marks[:len(t.Marks)-1])
	st.ExtendedTest = &next
	return nil
}

// CloseExtendedTest encerra o teste estendido.
func CloseExtendedTest(st *SessionRuntimeState) {
	st.ExtendedTest = nil
}
