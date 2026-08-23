package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"

	"t20engine/db/sqlcgen"
)

// restMultiplier is the T20 night-rest recovery factor per accommodation quality:
// PV/PM gained = floor(level × factor). Mirrors REST_MULTIPLIER in
// characters-effects.service.ts (livro: descanso). An unknown condition falls back to
// 'normal' — the gateway already defaults, this keeps the domain rule self-contained.
var restMultiplier = map[string]float64{"ruim": 0.5, "normal": 1, "confortavel": 2, "luxuosa": 3}

// restedVitals is the PV/PM current pair a rest leaves the character on.
type restedVitals struct {
	hpCurrent int64
	mpCurrent int64
}

// endScene expires the character's scene-scoped effects (owner-or-GM authorized first).
// Transport-agnostic — the WS session-rest handler calls this per member character.
func (s *Server) endScene(ctx context.Context, user AuthUser, characterID int64) (int, error) {
	if _, status, err := s.authorizedCharacter(ctx, user, characterID); err != nil {
		return status, err
	}
	if err := s.queries.DeleteEffectsByScope(ctx, sqlcgen.DeleteEffectsByScopeParams{Characterid: characterID, Scope: "scene"}); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear effects")
	}
	// Os usos "1/cena" e as posturas vao junto (ALE-222). Aqui e nao no
	// `endScene` da SESSAO: este e o caminho que ja limpa a ficha, e e por onde
	// os dois transportes passam. A colisao C1 do glossario mede o que custa
	// escolher o outro — o `endScene` da sessao nao limpa efeito nenhum.
	if err := s.clearScenePlayState(ctx, characterID); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear the play state")
	}
	return http.StatusOK, nil
}

// endDay expires both scene- and day-scoped effects.
func (s *Server) endDay(ctx context.Context, user AuthUser, characterID int64) (int, error) {
	if _, status, err := s.authorizedCharacter(ctx, user, characterID); err != nil {
		return status, err
	}
	if err := s.queries.DeleteSceneAndDayEffects(ctx, characterID); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear effects")
	}
	if err := s.clearDayPlayState(ctx, characterID); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear the play state")
	}
	return http.StatusOK, nil
}

// assertGmAtLiveTable guarda as rotas de escopo da FICHA: encerrar cena e
// encerrar dia sao do MESTRE, DURANTE uma sessao -- decisao do dono (ALE-223).
//
// Isto INVERTE o guarda da ALE-216, que recusava enquanto houvesse mesa em
// curso e liberava fora dela. A leitura nova e que as duas acoes nao pertencem
// a quem esta editando uma ficha: descanso e decisao da mesa, e mesa e o que
// existe durante uma sessao. Fora dela ninguem as executa -- nem o dono.
//
// A pergunta e uma so e e PRECISA: mestre de uma campanha DESTE personagem que
// tenha sessao viva. Compor "e mestre?" com "ha sessao viva?" em duas consultas
// deixaria passar o mestre da campanha A com sessao rodando na campanha B.
//
// Autorizacao roda ANTES (e de novo dentro do helper de dominio, que o caminho
// do socket chama sem este guarda): sem essa ordem, um estranho receberia "nao
// e o mestre da mesa" e aprenderia que a mesa esta rodando hoje.
func (s *Server) assertGmAtLiveTable(w http.ResponseWriter, r *http.Request, id int64) bool {
	user := currentUser(r)
	if _, status, err := s.authorizedCharacter(r.Context(), user, id); err != nil {
		writeError(w, status, err.Error())
		return false
	}
	gm, err := s.queries.IsGmAtLiveTableForCharacter(r.Context(), sqlcgen.IsGmAtLiveTableForCharacterParams{
		CharacterId: id, OwnerId: user.ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not check the character's sessions")
		return false
	}
	if !gm {
		writeError(w, http.StatusForbidden, fmt.Sprintf(
			"Ending the scene or the day for character %d is the GM's, during a live session", id))
		return false
	}
	return true
}

// clearEffectScopes runs one of the scope-expiring domain helpers for the {id}
// character and answers with the scopes the client must drop from its cached
// character — a delta, so the sheet updates without a refetch.
func (s *Server) clearEffectScopes(
	w http.ResponseWriter,
	r *http.Request,
	expire func(context.Context, AuthUser, int64) (int, error),
	cleared []string,
) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if !s.assertGmAtLiveTable(w, r, id) {
		return
	}
	if status, err := expire(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"clearedScopes": cleared})
}

// handleEndScene is the sheet's own "Encerrar cena" (Efeitos tab): one player
// ending their scene, as opposed to the GM's session-wide rest that reaches
// endScene through the WS gateway.
func (s *Server) handleEndScene(w http.ResponseWriter, r *http.Request) {
	s.clearEffectScopes(w, r, s.endScene, []string{"scene"})
}

// handleEndDay ends the day, which also ends the running scene (book rest
// semantics) — hence both scopes in the delta.
func (s *Server) handleEndDay(w http.ResponseWriter, r *http.Request) {
	s.clearEffectScopes(w, r, s.endDay, []string{"scene", "day"})
}

// restVitals applies the T20 night-rest recovery: PV/PM each gain floor(level × factor),
// clamped to their max, then persists. Returns the new current values so the gateway can
// mirror them onto the live tracker.
func (s *Server) restVitals(ctx context.Context, user AuthUser, characterID int64, condition string) (restedVitals, int, error) {
	row, status, err := s.authorizedCharacter(ctx, user, characterID)
	if err != nil {
		return restedVitals{}, status, err
	}
	mult, ok := restMultiplier[condition]
	if !ok {
		mult = restMultiplier["normal"]
	}
	gain := int64(math.Floor(float64(row.Level) * mult))
	next := restedVitals{
		hpCurrent: min(row.Hpmax, row.Hpcurrent+gain),
		mpCurrent: min(row.Mpmax, row.Mpcurrent+gain),
	}
	if err := s.queries.SetVitalsCurrent(ctx, sqlcgen.SetVitalsCurrentParams{
		HpCurrent: next.hpCurrent, MpCurrent: next.mpCurrent, UpdatedAt: nowISO(), ID: characterID,
	}); err != nil {
		return restedVitals{}, http.StatusInternalServerError, errors.New("Could not update vitals")
	}
	return next, http.StatusOK, nil
}

// handleAdjustEffect ports adjustActiveEffect: bump a temp-HP pool's amount by
// tempHpDelta; delete when it hits 0 ({removed, id}), else return the effect.
func (s *Server) handleAdjustEffect(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	effectID, ok := intParam(w, r, "effectId")
	if !ok {
		return
	}
	var body struct {
		TempHpDelta *int64 `json:"tempHpDelta"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.TempHpDelta == nil {
		writeValidationError(w, FieldErrorMap{"tempHpDelta": {"tempHpDelta must be an integer number"}})
		return
	}
	eff, err := s.queries.GetActiveEffect(r.Context(), effectID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && eff.Characterid != row.ID) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Active effect %d not found for character %d", effectID, row.ID))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load effect")
		return
	}

	var mods []map[string]any
	if json.Unmarshal([]byte(eff.Modifiers), &mods) != nil {
		writeError(w, http.StatusBadRequest, "Effect modifiers are malformed")
		return
	}
	idx := -1
	for i, m := range mods {
		if isTempHpModifier(m) {
			idx = i
			break
		}
	}
	if idx < 0 {
		writeError(w, http.StatusBadRequest, "Active effect has no temp HP to adjust")
		return
	}
	amount := max(0, toInt(mods[idx]["amount"])+int(*body.TempHpDelta))
	if amount == 0 {
		if err := s.queries.DeleteEffectByID(r.Context(), effectID); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not remove effect")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"removed": true, "id": effectID})
		return
	}
	mods[idx]["amount"] = amount
	next, _ := json.Marshal(mods)
	if err := s.queries.UpdateEffectModifiers(r.Context(), sqlcgen.UpdateEffectModifiersParams{Modifiers: string(next), ID: effectID}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update effect")
		return
	}
	writeJSON(w, http.StatusOK, EffectDTO{
		ID: eff.ID, CatalogID: eff.Catalogid, Scope: eff.Scope, Modifiers: string(next), CreatedAt: eff.Createdat,
	})
}

// handleDeleteEffect ports removeActiveEffect: 404 if the effect isn't on this
// character; returns {id}.
func (s *Server) handleDeleteEffect(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	effectID, ok := intParam(w, r, "effectId")
	if !ok {
		return
	}
	meta, err := s.queries.GetActiveEffectMeta(r.Context(), effectID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && meta.Characterid != row.ID) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Active effect %d not found", effectID))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load effect")
		return
	}
	if err := s.queries.DeleteEffectByID(r.Context(), effectID); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not remove effect")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": effectID})
}

type characterCampaignDTO struct {
	ID          int64              `json:"id"`
	CampaignID  int64              `json:"campaignId"`
	CharacterID int64              `json:"characterId"`
	Role        string             `json:"role"`
	AddedAt     string             `json:"addedAt"`
	Campaign    campaignSummaryDTO `json:"campaign"`
}

type campaignSummaryDTO struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	UpdatedAt   string  `json:"updatedAt"`
}

// handleListCharacterCampaigns ports members.listForCharacter: the campaigns a
// character has joined (owner-only, NOT GM). 404 missing, 403 not-owner.
func (s *Server) handleListCharacterCampaigns(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	owner, err := s.queries.GetCharacterOwner(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Character %d not found", id))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	if owner != currentUser(r).ID {
		writeError(w, http.StatusForbidden, fmt.Sprintf("Character %d belongs to another user", id))
		return
	}
	rows, err := s.queries.ListCampaignsForCharacter(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list campaigns")
		return
	}
	out := make([]characterCampaignDTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, characterCampaignDTO{
			ID: m.ID, CampaignID: m.Campaignid, CharacterID: m.Characterid, Role: m.Role, AddedAt: m.Addedat,
			Campaign: campaignSummaryDTO{
				ID: m.Campaignid, Name: m.Campaignname,
				Description: nullToPtr(m.Campaigndescription), UpdatedAt: m.Campaignupdatedat,
			},
		})
	}
	writeJSON(w, http.StatusOK, out)
}
