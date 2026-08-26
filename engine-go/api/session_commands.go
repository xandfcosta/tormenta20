package api

import "t20engine/aovivo"

import (
	"context"
	"errors"
	"net/http"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"
)

// CONTEXTO: SESSÃO AO VIVO — os comandos da iniciativa, do turno e da cena.
//
// Estes eram mensagens de socket e viraram rotas (ALE-253). A tradução é
// mecânica de propósito: a AUTORIZAÇÃO e a MUTAÇÃO não se movem — o
// `sessionForCaller`, o `assertGm` e o `sessionStore` continuam onde estavam e
// continuam sendo os mesmos. O que muda é quem os chama.
//
// O agrupamento por assunto é o primeiro passo do trabalho de bounded context
// (ALE-254): aqui ainda não há imposição do compilador, só vizinhança. Os nomes
// seguem os prefixos que o pacote já usa (`session_`, `board_`, `character_`) —
// abrir `sessao_*.go` ao lado de `session_*.go` seria duas palavras para um
// conceito, que é o que o GLOSSARIO proíbe.

// mountLiveRoutes pendura os comandos da mesa ao vivo (ALE-253).
//
// Fica AQUI e não no `server.go` por dois motivos: o roteador da raiz já passa
// das 200 linhas com o teto da casa em 500, e — o que importa mais — cada
// contexto passa a ser lido inteiro num arquivo só, comando e rota juntos
// (ALE-254). O dia de virar pacote começa por tabuleiro.Mover um arquivo, não por caçar
// linhas espalhadas.
func (s *Server) mountLiveRoutes(r chi.Router) {
	r.Route("/{id}/initiative", func(r chi.Router) {
		r.Post("/", s.handleInitiativeAdd)
		r.Post("/self", s.handleInitiativeSelf)
		r.Post("/populate", s.handleInitiativePopulate)
		r.Post("/next-turn", s.handleNextTurn)
		r.Post("/previous-turn", s.handlePreviousTurn)
		r.Delete("/", s.handleInitiativeReset)
		r.Patch("/{entryId}", s.handleInitiativeUpdate)
		r.Delete("/{entryId}", s.handleInitiativeRemove)
		r.Patch("/{entryId}/vitals", s.handleVitalsPatch)
		r.Post("/{entryId}/vitals/delta", s.handleVitalsDelta)
	})
	r.Route("/{id}/scene", func(r chi.Router) {
		r.Post("/start", s.handleSceneStart)
		r.Post("/end", s.handleSceneEnd)
	})
	r.Post("/{id}/rest", s.handleSessionRest)
	r.Post("/{id}/initiative/{entryId}/effects", s.handleTableSpellEffect)

	r.Route("/{id}/board", func(r chi.Router) {
		r.Get("/", s.handleBoardGet)
		r.Get("/as-player", s.handleBoardAsPlayer)
		r.Post("/", s.handleBoardOpen)
		r.Delete("/", s.handleBoardClose)
		r.Post("/curtain", s.handleBoardCurtain)
		r.Post("/populate", s.handleBoardPopulate)
		r.Post("/terrain", s.handleBoardTerrainPaint)

		r.Post("/tokens", s.handleBoardTokenAdd)
		r.Patch("/tokens/{tokenId}", s.handleBoardTokenUpdate)
		r.Delete("/tokens/{tokenId}", s.handleBoardTokenRemove)
		r.Post("/tokens/{tokenId}/duplicate", s.handleBoardTokenDuplicate)
		r.Post("/tokens/{tokenId}/move", s.handleBoardMovePropose)

		r.Post("/move/commit", s.handleBoardMoveCommit)
		r.Post("/move/cancel", s.handleBoardMoveCancel)

		r.Post("/markers", s.handleBoardMarkerAdd)
		r.Patch("/markers/{markerId}", s.handleBoardMarkerUpdate)
		r.Delete("/markers/{markerId}", s.handleBoardMarkerRemove)

		r.Get("/places", s.handleBoardPlaces)
		r.Post("/places/{placeId}/reopen", s.handleBoardReopen)
		r.Get("/places/{placeId}/scene", s.handleBoardPlaceScene)
		r.Put("/places/{placeId}/scene", s.handleBoardPlaceSave)
		r.Delete("/places/{placeId}", s.handleBoardPlaceRemove)
	})
}

// liveCtx é o que o socket chamava de `msgCtx`: quem pediu, em que mesa, com
// que papel. Resolvido uma vez por requisição.
type liveCtx struct {
	UserID     int64
	campaignID int64
	sessionID  int64
	Role       string
}

// liveAccess resolve a mesa e o papel de quem chamou, ou responde e devolve
// `false`. Espelha o `access` do gateway, com os ids vindo do CAMINHO em vez do
// corpo — que é a única diferença real entre os dois transportes.
func (s *Server) liveAccess(w http.ResponseWriter, r *http.Request) (liveCtx, bool) {
	campaignID, ok := intParam(w, r, "campaignId")
	if !ok {
		return liveCtx{}, false
	}
	sessionID, ok := intParam(w, r, "id")
	if !ok {
		return liveCtx{}, false
	}
	user := currentUser(r)
	_, Role, status, err := s.sessionForCaller(r.Context(), user, campaignID, sessionID)
	if err != nil {
		plataforma.WriteError(w, status, err.Error())
		return liveCtx{}, false
	}
	return liveCtx{UserID: user.ID, campaignID: campaignID, sessionID: sessionID, Role: Role}, true
}

// requireGmRole barra o que é do mestre. Espelha o `requireGm` do gateway.
//
// 403 e não 401: quem chega aqui está autenticado e é da mesa — só não manda
// nesta ação. Dizer "não autorizado" faria o cliente tentar entrar de novo.
func requireGmRole(w http.ResponseWriter, ctx liveCtx) bool {
	if ctx.Role != "gm" {
		plataforma.WriteError(w, http.StatusForbidden, "Only the GM can do this")
		return false
	}
	return true
}

// mutateAndPublish roda a mutação, transmite o estado novo à mesa e responde a
// quem pediu. Espelha o `mutateAndBroadcast` do gateway.
//
// A RESPOSTA é o ack que o socket dava por callback — e de graça, porque HTTP
// já responde. Ela leva o estado do ponto de vista de QUEM PEDIU: o mestre
// recebe inteiro, o jogador recebe redigido, pelo mesmo `aovivo.RedactForPlayers` que
// alimenta o broadcast. Responder o estado cheio a um jogador aqui abriria pela
// porta da frente exatamente o que a sala por papel fecha.
func (s *Server) mutateAndPublish(
	w http.ResponseWriter, ctx liveCtx, mutate func() (*aovivo.SessionRuntimeState, error),
) {
	state, err := mutate()
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.publishSessionState(ctx.sessionID, state)
	plataforma.WriteJSON(w, http.StatusOK, aovivo.StateForRole(ctx.Role, state))
}

// publishSessionState transmite o estado às duas salas por papel e persiste.
// Espelha o `emitSessionState` do gateway.
func (s *Server) publishSessionState(sessionID int64, state *aovivo.SessionRuntimeState) {
	s.sse.EmitOrdered(sessionID, "gm", "session-state", state.Seq, state)
	s.sse.EmitOrdered(sessionID, "player", "session-state", state.Seq, aovivo.RedactForPlayers(state))
	s.emSegundoPlano.Add(1)
	go func() {
		defer s.emSegundoPlano.Done()
		s.persistSessionAndWarn(sessionID)
	}()
}

// --- Iniciativa ------------------------------------------------------------

func (s *Server) handleInitiativeAdd(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	var body struct {
		Entry map[string]any `json:"entry"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.Entry == nil {
		plataforma.WriteError(w, http.StatusBadRequest, "entry is required")
		return
	}
	entry, err := s.materializeEntry(r.Context(), ctx.UserID, ctx.campaignID, body.Entry)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.AddInitiativeEntry(ctx.sessionID, entry)
	})
}

func (s *Server) handleInitiativeRemove(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	entryID := chi.URLParam(r, "entryId")
	if entryID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "entryId is required")
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.RemoveInitiativeEntry(ctx.sessionID, entryID)
	})
}

func (s *Server) handleNextTurn(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.NextTurn(ctx.sessionID)
	})
}

func (s *Server) handlePreviousTurn(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.PreviousTurn(ctx.sessionID)
	})
}

func (s *Server) handleInitiativeSelf(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok {
		return
	}
	// NÃO é do mestre: quem registra a própria iniciativa é o jogador, e o
	// `selfInitiativeEntry` confere que ele é dono do personagem.
	var body struct {
		CharacterID int64 `json:"characterId"`
		D20         int64 `json:"d20"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.CharacterID == 0 {
		plataforma.WriteError(w, http.StatusBadRequest, "characterId is required")
		return
	}
	entry, err := s.selfInitiativeEntry(ctx.UserID, ctx.campaignID, body.CharacterID, body.D20)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.UpsertInitiativeEntry(ctx.sessionID, entry)
	})
}

func (s *Server) handleInitiativeUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	entryID := chi.URLParam(r, "entryId")
	if entryID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "entryId is required")
		return
	}
	var body struct {
		Patch map[string]any `json:"patch"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	patch := parseEntryPatch(body.Patch)
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.UpdateInitiativeEntry(ctx.sessionID, entryID, patch)
	})
}

func (s *Server) handleInitiativeReset(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.Reset(ctx.sessionID)
	})
}

// handleInitiativePopulate traz os PCs da campanha. Idempotente: pula quem já
// está. Transmite o que POUSOU mesmo em caso de erro parcial — um cliente com a
// fila defasada se ressincroniza até num no-op.
func (s *Server) handleInitiativePopulate(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	combatants, err := s.listPlayerCombatants(r.Context(), ctx.campaignID)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadGateway, "Could not Load party")
		return
	}
	state, addErr := s.populateParty(ctx.sessionID, combatants)
	if state == nil {
		state = s.sessions.GetState(ctx.sessionID)
	}
	s.publishSessionState(ctx.sessionID, state)
	if addErr != nil {
		plataforma.WriteError(w, http.StatusBadRequest, addErr.Error())
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, aovivo.StateForRole(ctx.Role, state))
}

// --- Cena ------------------------------------------------------------------

func (s *Server) handleSceneStart(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.StartScene(ctx.sessionID)
	})
}

// handleSceneEnd não usa o `mutateAndPublish` porque a mesa precisa de DOIS
// avisos: o estado, e que as FICHAS mudaram (ALE-220).
//
// As fichas não estão no estado do rastreador, então sem o segundo o efeito
// morto e o "usado 1/cena" ficariam na tela até alguém recarregar. O aviso é o
// `session-rest`, que no fio significa "o servidor expirou o escopo X do
// grupo" — que é exatamente o que aconteceu.
func (s *Server) handleSceneEnd(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	state, err := s.endSceneForTable(currentUser(r), ctx.campaignID, ctx.sessionID)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.publishSessionState(ctx.sessionID, state)
	s.sse.Emit(ctx.sessionID, "", "session-rest", map[string]any{
		"sessionId": ctx.sessionID, "scope": "scene",
	})
	plataforma.WriteJSON(w, http.StatusOK, aovivo.StateForRole(ctx.Role, state))
}

// --- Vitais e descanso -----------------------------------------------------

func (s *Server) handleVitalsPatch(w http.ResponseWriter, r *http.Request) {
	ctx, entryID, ok := s.vitalsAccess(w, r)
	if !ok {
		return
	}
	var body struct {
		Patch struct {
			HpCurrent *int64 `json:"hpCurrent"`
			MpCurrent *int64 `json:"mpCurrent"`
		} `json:"patch"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.PatchVitals(ctx.sessionID, entryID, body.Patch.HpCurrent, body.Patch.MpCurrent)
	})
}

func (s *Server) handleVitalsDelta(w http.ResponseWriter, r *http.Request) {
	ctx, entryID, ok := s.vitalsAccess(w, r)
	if !ok {
		return
	}
	var body struct {
		HpDelta *int64 `json:"hpDelta"`
		MpDelta *int64 `json:"mpDelta"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	s.mutateAndPublish(w, ctx, func() (*aovivo.SessionRuntimeState, error) {
		return s.sessions.DeltaVitals(ctx.sessionID, entryID, body.HpDelta, body.MpDelta)
	})
}

// vitalsAccess é o preâmbulo comum dos dois: resolve a mesa, lê a entrada e
// autoriza. Espelha o `vitalsAccess` do gateway — o mestre edita qualquer um, o
// jogador só o próprio personagem, NPC é do mestre.
func (s *Server) vitalsAccess(w http.ResponseWriter, r *http.Request) (liveCtx, string, bool) {
	ctx, ok := s.liveAccess(w, r)
	if !ok {
		return liveCtx{}, "", false
	}
	entryID := chi.URLParam(r, "entryId")
	if entryID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "entryId is required")
		return liveCtx{}, "", false
	}
	if err := s.assertVitalsEditableFor(r.Context(), ctx, entryID); err != nil {
		plataforma.WriteError(w, http.StatusForbidden, err.Error())
		return liveCtx{}, "", false
	}
	return ctx, entryID, true
}

func (s *Server) handleSessionRest(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	var body struct {
		Scope     string `json:"scope"`
		Condition string `json:"condition"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.Condition == "" {
		body.Condition = "normal"
	}
	done, total, err := s.restParty(currentUser(r), ctx.campaignID, ctx.sessionID, body.Scope, body.Condition)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadGateway, "Could not Load campaign members")
		return
	}
	if done > 0 {
		s.publishSessionState(ctx.sessionID, s.sessions.GetState(ctx.sessionID))
	}
	s.sse.Emit(ctx.sessionID, "", "session-rest", map[string]any{
		"sessionId": ctx.sessionID, "scope": body.Scope, "condition": body.Condition,
	})
	// `healed` mantém o nome antigo porque o cliente o lê, mas conta os DOIS
	// escopos: encerrar cena que falha deixa de somar (ALE-155).
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{
		"rested": body.Scope, "characters": total, "healed": done,
	})
}

// handleTableSpellEffect aplica uma magia de buff a um COMBATENTE da mesa.
//
// O nome não é `handleApplyEffect`, e isso é deliberado: já existe um
// `handleApplyEffect` que é da FICHA e lê `{id}` como personagem. Montá-lo aqui
// compilaria e leria o id da SESSÃO como id de ficha — errado em silêncio, o
// pior desfecho possível de uma troca de transporte. Dois handlers com o mesmo
// nome em contextos diferentes é exatamente o que o trabalho de bounded context
// (ALE-254) existe para tornar impossível.
func (s *Server) handleTableSpellEffect(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	entryID := chi.URLParam(r, "entryId")
	if entryID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "entryId is required")
		return
	}
	var body struct {
		SpellID string  `json:"spellId"`
		Scope   *string `json:"scope"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.SpellID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "spellId is required")
		return
	}
	characterID, err := s.characterOfEntry(ctx.sessionID, entryID)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, _, err := s.applySpellBuffEffect(r.Context(), characterID, body.SpellID, body.Scope); err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.sse.Emit(ctx.sessionID, "", "effect-applied", map[string]any{
		"sessionId": ctx.sessionID, "characterId": characterID, "spellId": body.SpellID,
	})
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"applied": body.SpellID, "characterId": characterID})
}

// characterOfEntry resolve a ficha por trás de uma linha da fila. NPC não tem,
// e é por isso que ele não recebe magia de buff: não há números derivados para
// mexer (os do bloco são escritos à mão).
func (s *Server) characterOfEntry(sessionID int64, entryID string) (int64, error) {
	state := s.sessions.GetState(sessionID)
	idx := aovivo.FindEntryIndex(state, entryID)
	if idx < 0 {
		return 0, errors.New("Entry " + entryID + " not found")
	}
	entry := state.Initiative[idx]
	if entry.CharacterID == nil {
		return 0, errors.New("Only character entries can receive spell effects")
	}
	return *entry.CharacterID, nil
}

// persistSettleAndWarn persiste e avisa a mesa SÓ quando o sinal de sujeira
// vira — primeira falha, ou uma tentativa que se recuperou. Espelha o
// `persistAndWarn` do gateway; quem é dono do sinal é o store.
func (s *Server) persistSessionAndWarn(sessionID int64) {
	Dirty, changed := s.sessions.Persist(context.Background(), sessionID)
	if !changed {
		return
	}
	s.sse.Emit(sessionID, "", "persistence-warning", map[string]any{
		"sessionId": sessionID, "Dirty": Dirty,
	})
}
