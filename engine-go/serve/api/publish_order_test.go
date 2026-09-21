package api

import "t20engine/domain/live"

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// O QUADRO QUE DESCE TEM DE SEGUIR A ORDEM DA MUTAÇÃO.
//
// O sintoma é "uma condição some no meio de três", e o mecanismo é
// determinístico:
//
//	apply()  → trava, muta, CLONA, destrava
//	publish  → emite o clone  ← FORA da trava
//
// Duas mutações concorrentes serializam a MUTAÇÃO e não a EMISSÃO. A que mutou
// primeiro pode emitir por último, o cliente aplica o estado inteiro sem número
// de versão, o quadro velho vence, e a condição que estava no quadro novo
// desaparece da tela.
//
// ELE É DE SERVIDOR e não de e2e: perseguir isso pela suíte é loteria — a base
// medida foi 1 vermelho em 8 corridas, e com essa base oito corridas não
// distinguem conserto de sorte (Fisher exato: p = 0,57).
//
// A garantia também é do SERVIDOR e não do cliente: SSE preserva a ordem POR
// CONEXÃO, então basta emitir na ordem certa. Pedir ao cliente que descarte
// quadro velho exigiria versão no fio e uma segunda cópia da regra em cada tela
// que escuta.
func TestTheFrameFollowsTheOrderOfTheMutation(t *testing.T) {
	const sessionID = int64(7)
	// Repetição porque a corrida é de agendamento: uma passada só não a
	// visita. Sem o conserto isto fica vermelho em poucas dezenas.
	const attempts = 200

	for attempt := range attempts {
		s := newTestServer(t)
		conn := s.sse.Add(sessionID, "c1", "gm")

		var wg sync.WaitGroup
		for _, name := range []string{"Abalado", "Agarrado", "Cego", "Surdo", "Lento", "Fraco"} {
			wg.Add(1)
			go func(label string) {
				defer wg.Done()
				state, err := s.sessions.AddInitiativeEntry(sessionID, live.InitiativeEntry{
					Label: label, Type: "npc", Initiative: 10,
				})
				if err != nil {
					return
				}
				s.tableRules().publishSessionState(sessionID, state)
			}(name)
		}
		wg.Wait()

		last := ultimoQuadro(conn)
		if last == "" {
			t.Fatalf("tentativa %d: nenhum quadro desceu — o canal não existe, e ausência aqui não é resultado", attempt)
		}
		// O ÚLTIMO quadro é o que fica na tela. Ele tem de conter as duas
		// entradas: as duas mutações já aconteceram quando o `wg.Wait` voltou.
		missing := ""
		for _, name := range []string{"Abalado", "Agarrado", "Cego", "Surdo", "Lento", "Fraco"} {
			if !strings.Contains(last, name) {
				missing = name
				break
			}
		}
		if missing != "" {
			t.Fatalf("tentativa %d: o último quadro perdeu %q — a tela fica com ela sumida.\n%s",
				attempt, missing, resumoDoQuadro(t, last))
		}
		s.sse.Remove(sessionID, "c1")
	}
}

// ultimoQuadro drena a fila e devolve o último quadro, que é o que sobrevive na
// tela — os anteriores são sobrescritos por ele.
func ultimoQuadro(conn *live.SSEConn) string {
	var last string
	for {
		select {
		case frame := <-conn.Frames:
			if strings.Contains(string(frame), "event: session-state") {
				last = string(frame)
			}
		default:
			return last
		}
	}
}

// resumoDoQuadro lista os rótulos que o quadro carrega, para a falha dizer o
// que sobrou em vez de despejar o JSON inteiro.
func resumoDoQuadro(t *testing.T, frame string) string {
	t.Helper()
	_, body, _ := strings.Cut(frame, "data: ")
	var state struct {
		Initiative []struct {
			Label string `json:"label"`
		} `json:"initiative"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(body)), &state); err != nil {
		return frame
	}
	labels := make([]string, 0, len(state.Initiative))
	for _, e := range state.Initiative {
		labels = append(labels, e.Label)
	}
	return "sobrou: [" + strings.Join(labels, " ") + "]"
}
