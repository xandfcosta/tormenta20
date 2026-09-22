package table

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O TESTE DE QUEM SANGRA, na faixa da vez (T20 p236). O jogador rola e digita
// (decisão do dono, ALE-366): a faixa pergunta o d20, e na falha o d6. Quem
// responde é o MESTRE ou o DONO da ficha — os outros leem a pergunta e o
// resultado, que são da mesa inteira.

// bleedingView é o que a faixa desenha do teste.
type bleedingView struct {
	// Line é a frase do passo: a pergunta, a falha pedindo o d6, ou o resultado.
	Line string
	// Die é o dado que o passo pede ("d20" ou "d6"), e vazio quando o teste já
	// terminou — aí só a frase fica.
	Die string
	// Faces é o maior valor do dado, para o campo e para o "Rolar".
	Faces int
	// CanAnswer diz se QUEM OLHA responde: o mestre, ou o dono da ficha.
	CanAnswer bool
}

// bleedingPromptOf monta a vista do teste da vez, ou nil fora dele.
func bleedingPromptOf(st *live.SessionRuntimeState, role string, mine map[int64]bool) *bleedingView {
	if st == nil || st.Scene == nil || st.Scene.Bleeding == nil {
		return nil
	}
	check := st.Scene.Bleeding
	v := &bleedingView{
		Line:      live.BleedingLine(check, engine.BleedingCheckDC),
		CanAnswer: role == app.RoleGM || mine[check.CharacterID],
	}
	switch {
	case check.Resolved():
	case check.AwaitingD6:
		v.Die, v.Faces = "d6", 6
	default:
		v.Die, v.Faces = "d20", 20
	}
	return v
}

// bleedingSignals é o que o Datastar manda: só o número do dado é lido.
type bleedingSignals struct {
	BleedingRoll int `json:"bleeding_roll"`
}

// handleBleedingRoll recebe o d20 ou o d6 do teste da vez.
//
// Ele não passa pelo `gmCommand` porque o gesto é do mestre E do dono da ficha,
// como o registro da iniciativa — e segue a mesma ordem daquele handler: lê os
// sinais ANTES do `NewSSE`, que fecha o corpo, e responde SEMPRE em SSE.
func (s Scene) handleBleedingRoll(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var signals bleedingSignals
	readErr := datastar.ReadSignals(r, &signals)
	sse := datastar.NewSSE(w, r)
	failure := ""
	if readErr != nil {
		failure = fmt.Sprintf("não entendi o dado enviado: %v", readErr)
	} else if err := s.answerBleeding(r, campaignID, sessionID, chi.URLParam(r, "die"), signals.BleedingRoll); err != nil {
		failure = err.Error()
	}
	_ = sse.MarshalAndPatchSignals(map[string]any{"bleeding_error": failure, "bleeding_roll": ""})
}

// answerBleeding autoriza e entrega o dado ao caso de uso.
func (s Scene) answerBleeding(r *http.Request, campaignID, sessionID int64, die string, value int) error {
	userID := s.deps.CurrentUserID(r)
	_, role, err := s.access.Session(r.Context(), app.Caller{ID: userID}, campaignID, sessionID)
	if err != nil {
		return err
	}
	state, err := s.deps.Sessions().State(r.Context(), sessionID)
	if err != nil {
		return err
	}
	check := state.Scene.PendingBleeding(die == "d6")
	if check == nil {
		return errors.New("a vez não está esperando este dado")
	}
	if role != app.RoleGM {
		_, mine, _ := s.tableRoster(r.Context(), userID, campaignID)
		if !mine[check.CharacterID] {
			return fmt.Errorf("o teste é de %s, e só o dono da ficha ou o mestre rolam por ele", check.Label)
		}
	}
	switch die {
	case "d20":
		state, err = s.deps.Sessions().RollBleedingD20(r.Context(), sessionID, value)
	case "d6":
		state, err = s.deps.Sessions().RollBleedingD6(r.Context(), sessionID, value)
	default:
		return fmt.Errorf("o teste pede d20 ou d6, e veio %q", die)
	}
	if err != nil {
		return err
	}
	s.deps.PublishSessionState(sessionID, state)
	return nil
}
