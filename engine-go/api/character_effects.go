package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"t20engine/plataforma"

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

// EndScene expires the character's scene-scoped effects (owner-or-GM authorized first).
// Transport-agnostic — the WS session-rest handler calls this per member character.
func (tr tableRules) EndScene(ctx context.Context, user AuthUser, characterID int64) (int, error) {
	if _, status, err := tr.authorizedCharacter(ctx, user, characterID); err != nil {
		return status, err
	}
	if err := tr.queries.DeleteEffectsByScope(ctx, sqlcgen.DeleteEffectsByScopeParams{Characterid: characterID, Scope: "scene"}); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear effects")
	}
	// Os usos "1/cena" e as posturas vao junto (ALE-222). Aqui e nao no
	// `EndScene` da SESSAO: este e o caminho que ja limpa a ficha, e e por onde
	// os dois transportes passam. Desde a ALE-220 o `EndScene` da sessao
	// tambem chega ate aqui, uma ficha por vez, pelo `expirePartyScene`.
	if err := tr.clearScenePlayState(ctx, characterID); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear the play state")
	}
	return http.StatusOK, nil
}

// endDay expires both scene- and day-scoped effects.
func (tr tableRules) endDay(ctx context.Context, user AuthUser, characterID int64) (int, error) {
	if _, status, err := tr.authorizedCharacter(ctx, user, characterID); err != nil {
		return status, err
	}
	if err := tr.queries.DeleteSceneAndDayEffects(ctx, characterID); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear effects")
	}
	if err := tr.clearDayPlayState(ctx, characterID); err != nil {
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
	if _, status, err := s.tableRules().authorizedCharacter(r.Context(), user, id); err != nil {
		plataforma.WriteError(w, status, err.Error())
		return false
	}
	gm, err := s.queries.IsGmAtLiveTableForCharacter(r.Context(), sqlcgen.IsGmAtLiveTableForCharacterParams{
		CharacterId: id, OwnerId: user.ID,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not check the character's sessions")
		return false
	}
	if !gm {
		plataforma.WriteError(w, http.StatusForbidden, fmt.Sprintf(
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
		plataforma.WriteError(w, status, err.Error())
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string][]string{"clearedScopes": cleared})
}

// handleEndScene is the sheet's own "Encerrar cena" (Efeitos tab): one player
// ending their scene, as opposed to the GM's session-wide rest that reaches
// EndScene through the WS gateway.
func (s *Server) handleEndScene(w http.ResponseWriter, r *http.Request) {
	s.clearEffectScopes(w, r, s.tableRules().EndScene, []string{"scene"})
}

// handleEndDay ends the day, which also ends the running scene (book rest
// semantics) — hence both scopes in the delta.
func (s *Server) handleEndDay(w http.ResponseWriter, r *http.Request) {
	s.clearEffectScopes(w, r, s.tableRules().endDay, []string{"scene", "day"})
}

// restVitals applies the T20 night-rest recovery: PV/PM each gain floor(level × factor),
// clamped to their max, then persists. Returns the new current values so the gateway can
// mirror them onto the live tracker.
func (tr tableRules) restVitals(ctx context.Context, user AuthUser, characterID int64, condition string) (restedVitals, int, error) {
	row, status, err := tr.authorizedCharacter(ctx, user, characterID)
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
	if err := tr.queries.SetVitalsCurrent(ctx, sqlcgen.SetVitalsCurrentParams{
		HpCurrent: next.hpCurrent, MpCurrent: next.mpCurrent, UpdatedAt: plataforma.NowISO(), ID: characterID,
	}); err != nil {
		return restedVitals{}, http.StatusInternalServerError, errors.New("Could not update vitals")
	}
	return next, http.StatusOK, nil
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

// Aqui morava o `character_play_state_test.go`, com sete casos sobre o estado de
// JOGO da ficha (ALE-222). Ele saiu na ALE-277 junto com as rotas JSON que
// dirigia, e cada garantia dele continua presa em outra camada:
//
//   - os USOS de poder (somam, e a cena não se mistura com o dia): a aba de
//     Poderes da ficha, em `TestUsingChargesTheMpAndCountsTheUse`;
//   - o DESCANSO de cena e de dia: a Mesa, em
//     `TestTheSceneRestExpiresTheSheetsWithoutTurningTheSceneOff` e
//     `TestTheDayRestUsesTheQualityTheGmChose`;
//   - os SITUACIONAIS: a aba de Efeitos, em `TestActiveConditionalsEnterTheAttack`.
//
// Um deles não migrou porque a BEHAVIOR sumiu junto: "a lista de conditionals
// substitui o conjunto inteiro" era do handler JSON, e a cena alterna UM de cada
// vez (`toggleSituational`). Guardar uma substituição que ninguém faz seria
// teste sobre código morto.
