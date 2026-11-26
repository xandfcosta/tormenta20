package table

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/domain/live"
)

// O TESTE ESTENDIDO na mesa (ver GLOSSARY, **teste estendido**): o diálogo que o mestre abre pelo menu da sessão,
// a faixa do contador que as duas sessões veem e os gestos que só ele faz — marcar, desfazer a última, encerrar.

// extendedTestRoutes pendura os gestos. A marca e a linha vão no CAMINHO: nesta superfície o verbo é o caminho.
func (s Scene) extendedTestRoutes(r chi.Router) {
	base := sessionPattern + "/teste-estendido"
	r.Post(base+"/abrir", s.gmCommand(opensTheExtendedTest))
	r.Post(base+"/marcar/{marca}", s.gmCommand(marksTheExtendedTest))
	r.Post(base+"/linha/{entryId}/marcar/{marca}", s.gmCommand(marksTheExtendedTest))
	r.Post(base+"/desfazer", s.gmCommand(func(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
		return st.deps.Sessions().UndoExtendedMark(c.R.Context(), c.SessionID)
	}))
	r.Post(base+"/encerrar", s.gmCommand(func(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
		return st.deps.Sessions().CloseExtendedTest(c.R.Context(), c.SessionID)
	}))
}

// extendedSignals é o que o diálogo de abrir manda.
type extendedSignals struct {
	Name     string `json:"ext_name"`
	Required int    `json:"ext_required"`
	Each     bool   `json:"ext_each"`
}

// opensTheExtendedTest abre o contador com o que o diálogo diz, e limpa o nome DEPOIS de a mesa aceitar: o
// próximo teste não nasce com o nome do anterior. Lê os sinais antes do `NewSSE`, pela razão do `restQuality`.
func opensTheExtendedTest(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	c.R.Body = http.MaxBytesReader(nil, c.R.Body, 1<<20)
	var asked extendedSignals
	if err := datastar.ReadSignals(c.R, &asked); err != nil {
		return nil, fmt.Errorf("não entendi o teste estendido: %v", err)
	}
	state, err := st.deps.Sessions().OpenExtendedTest(c.R.Context(), c.SessionID, asked.Name, asked.Required, asked.Each)
	if err != nil {
		return nil, err
	}
	c.Signals["ext_name"] = ""
	return state, nil
}

// marksTheExtendedTest põe a marca do caminho na contagem da linha, ou na do grupo sem linha.
func marksTheExtendedTest(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().MarkExtendedTest(c.R.Context(), c.SessionID,
		chi.URLParam(c.R, "entryId"), chi.URLParam(c.R, "marca"))
}

// extendedComplexities são os sucessos exigidos que o diálogo oferece, com o nome da Tabela 5-2 (p223).
var extendedComplexities = []struct {
	Required int
	Name     string
}{{3, "baixa"}, {5, "média"}, {7, "alta"}}

// extendedLine é uma contagem do jeito que a faixa a escreve.
type extendedLine struct {
	EntryID, Label string
	// Score é o "2/5" grande, Tally o resto da conta, e Said o que falta ou o que o livro prevê.
	Score, Tally, Said string
	Done               bool
}

// extendedBadge é a etiqueta da faixa: a complexidade do livro, ou "cada um da fila".
func extendedBadge(t live.ExtendedTest) string {
	if len(t.Who) > 0 {
		return "Teste estendido · cada um da fila"
	}
	for _, c := range extendedComplexities {
		if c.Required == t.Required {
			return "Teste estendido · " + c.Name
		}
	}
	return fmt.Sprintf("Teste estendido · %d sucessos", t.Required)
}

// extendedLines são as contagens do teste: uma só no do grupo, uma por linha no de cada um.
func extendedLines(t live.ExtendedTest) []extendedLine {
	if len(t.Who) == 0 {
		return []extendedLine{extendedLineOf(t, "", "")}
	}
	lines := make([]extendedLine, 0, len(t.Who))
	for _, w := range t.Who {
		lines = append(lines, extendedLineOf(t, w.EntryID, w.Label))
	}
	return lines
}

func extendedLineOf(t live.ExtendedTest, entryID, label string) extendedLine {
	tally := t.TallyOf(entryID)
	line := extendedLine{
		EntryID: entryID, Label: label, Done: tally.Outcome != "",
		Score: fmt.Sprintf("%d/%d", tally.Successes, t.Required),
		Tally: fmt.Sprintf("sucessos · %d de %d falhas", tally.Failures, live.ExtendedFailureLimit),
	}
	switch tally.Outcome {
	case live.ExtendedReached:
		line.Said = fmt.Sprintf("Alcançado: %d sucessos antes de %d falhas (p222). O que isso causa é seu.", t.Required, live.ExtendedFailureLimit)
	case live.ExtendedTotalFailure:
		line.Said = fmt.Sprintf("Falha total: %d falhas antes de %d sucessos (p222). O que isso causa é seu.", live.ExtendedFailureLimit, t.Required)
	default:
		line.Said = missing(t.Required-tally.Successes, "sucesso", "Falta", "Faltam") + ". " +
			missing(live.ExtendedFailureLimit-tally.Failures, "falha", "Mais", "Mais") + " " +
			plural(live.ExtendedFailureLimit-tally.Failures, "é a falha total.", "são a falha total.")
	}
	return line
}

// eachLineMeta é a conta da linha no teste de cada um: "3 de 5 sucessos · 0 de 3 falhas · chegou".
func eachLineMeta(t live.ExtendedTest, l extendedLine) string {
	tally := t.TallyOf(l.EntryID)
	meta := fmt.Sprintf("%d de %d sucessos · %d de %d falhas", tally.Successes, t.Required, tally.Failures, live.ExtendedFailureLimit)
	switch tally.Outcome {
	case live.ExtendedReached:
		meta += " · chegou"
	case live.ExtendedTotalFailure:
		meta += " · falha total"
	}
	return meta
}

// missing escreve "Faltam 3 sucessos." e "Mais 2 falhas", no singular quando é um.
func missing(n int, noun, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%s 1 %s", one, noun)
	}
	return fmt.Sprintf("%s %d %ss", many, n, noun)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// extendedMark é o caminho do gesto de marcar, depois da sessão: o do grupo, ou o da linha.
func extendedMark(l extendedLine, kind string) string {
	if l.EntryID == "" {
		return "teste-estendido/marcar/" + kind
	}
	return "teste-estendido/linha/" + l.EntryID + "/marcar/" + kind
}
