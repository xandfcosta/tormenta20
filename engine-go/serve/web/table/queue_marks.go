package table

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/domain/live"
)

// A MARCA NA FILA na Mesa do mestre (ver GLOSSARY, **marca na fila**): marcar a linha de quem falhou, tirar a marca,
// e rolar na linha o dado que se rola na vez dela. Os três são do mestre: quem diz quem falhou é ele.

// queueMarkRoutes pendura os gestos da marca na rota da linha (`/iniciativa/{entryId}`).
func (s Scene) queueMarkRoutes(r chi.Router) {
	r.Post("/marcar", s.gmCommand(marksTheEntry))
	r.Post("/marcas/{marca}/tirar", s.gmCommand(unmarksTheEntry))
	r.Post("/na-vez/{dado}/rolar", s.gmCommand(rollsTheDue))
}

func marksTheEntry(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().MarkEntry(c.R.Context(), c.SessionID, chi.URLParam(c.R, "entryId"))
}

// unmarksTheEntry tira a n-ésima marca da linha. O índice vem do caminho, e a marca se acha pelo nome da magia,
// que é o que a regra usa: o nome no caminho teria acento a escapar.
func unmarksTheEntry(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	entryID := chi.URLParam(c.R, "entryId")
	n, err := strconv.Atoi(chi.URLParam(c.R, "marca"))
	if err != nil {
		return nil, fmt.Errorf("a marca %q não é um número", chi.URLParam(c.R, "marca"))
	}
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	for _, e := range state.Initiative {
		if e.ID != entryID {
			continue
		}
		if n < 0 || n >= len(e.Marks) {
			return nil, fmt.Errorf("%s tem %d marca(s), e o pedido foi a %d", e.Label, len(e.Marks), n+1)
		}
		return st.deps.Sessions().UnmarkEntry(c.R.Context(), c.SessionID, entryID, e.Marks[n].Source)
	}
	return nil, fmt.Errorf("a linha %q não está na fila", entryID)
}

// rollsTheDue rola o n-ésimo dado da vez da linha, ou recebe o total da mão do campo dele (`due_hand_<n>`). O campo
// vazio é rolar no app.
func rollsTheDue(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	n, err := strconv.Atoi(chi.URLParam(c.R, "dado"))
	if err != nil {
		return nil, fmt.Errorf("o dado %q não é um número", chi.URLParam(c.R, "dado"))
	}
	told, err := dueHandTotal(c.R, n)
	if err != nil {
		return nil, err
	}
	var fromTheTable []*int
	if told != nil {
		fromTheTable = []*int{told}
	}
	state, err := st.deps.Sessions().RollDue(c.R.Context(), c.SessionID, chi.URLParam(c.R, "entryId"), n,
		fromTheTable, c.User, st.deps.RollDie)
	if err != nil {
		return nil, err
	}
	// O campo da mão se limpa DEPOIS de a mesa aceitar, pela razão do `commandCtx.Signals`.
	c.Signals[dueHandSignal(n)] = ""
	return state, nil
}

// dueHandTotal lê o total da mão do dado da vez, ou nil quando o campo veio vazio ou não veio.
func dueHandTotal(r *http.Request, n int) (*int, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	signals := map[string]any{}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return nil, fmt.Errorf("não entendi o total da mesa: %v", err)
	}
	switch v := signals[dueHandSignal(n)].(type) {
	case float64:
		total := int(v)
		return &total, nil
	case string:
		if v == "" {
			return nil, nil
		}
		total, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("o total da mesa %q não é um número", v)
		}
		return &total, nil
	}
	return nil, nil
}

// dueHandSignal é o sinal do campo "ou rolei" do n-ésimo dado da vez.
func dueHandSignal(n int) string { return "due_hand_" + strconv.Itoa(n) }
