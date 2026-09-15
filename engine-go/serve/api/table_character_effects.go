package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/httpio"

	"t20engine/infra/db/sqlcgen"
)

// restMultiplier é o fator de recuperação do descanso noturno do T20 por
// qualidade de acomodação: PV/PM ganhos = floor(nível × fator). Acomodação
// desconhecida cai para 'normal', o que mantém a regra inteira aqui dentro.
var restMultiplier = map[string]float64{"ruim": 0.5, "normal": 1, "confortavel": 2, "luxuosa": 3}

// restedVitals is the PV/PM current pair a rest leaves the character on.
type restedVitals struct {
	hpCurrent int64
	mpCurrent int64
}

// EndScene expira os efeitos de escopo de CENA do personagem (a autorização —
// dono ou mestre — roda antes). Sem transporte: quem chama o faz por ficha.
func (tr tableRules) EndScene(ctx context.Context, user AuthUser, characterID int64) (int, error) {
	if _, status, err := tr.authorizedCharacter(ctx, user, characterID); err != nil {
		return status, err
	}
	if err := tr.queries.DeleteEffectsByScope(ctx, sqlcgen.DeleteEffectsByScopeParams{Characterid: characterID, Scope: "scene"}); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear effects")
	}
	// Os usos "1/cena" e as posturas vão junto. Aqui e não no `EndScene` da
	// SESSÃO: este é o caminho que já limpa a ficha, e é por onde os dois
	// transportes passam — o da sessão chega até aqui uma ficha por vez, pelo
	// `expirePartyScene`.
	if err := tr.clearScenePlayState(ctx, characterID); err != nil {
		return http.StatusInternalServerError, errors.New("Could not clear the play state")
	}
	return http.StatusOK, nil
}

// endDay expira os efeitos de escopo de cena E de dia.
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
// encerrar dia são do MESTRE, DURANTE uma sessão.
//
// As duas ações não pertencem a quem está editando uma ficha: descanso é decisão
// da mesa, e mesa é o que existe durante uma sessão. Fora dela ninguém as
// executa — nem o dono.
//
// A pergunta é UMA SÓ e é PRECISA: mestre de uma campanha DESTE personagem que
// tenha sessão viva. Compor "é mestre?" com "há sessão viva?" em duas consultas
// deixaria passar o mestre da campanha A com sessão rodando na campanha B.
//
// A autorização roda ANTES (e de novo dentro do helper de domínio, que outro
// caminho chama sem este guarda): sem essa ordem, um estranho receberia "não é o
// mestre da mesa" e aprenderia que a mesa está rodando hoje.
func (s *Server) assertGmAtLiveTable(w http.ResponseWriter, r *http.Request, id int64) bool {
	user := currentUser(r)
	if _, status, err := s.tableRules().authorizedCharacter(r.Context(), user, id); err != nil {
		httpio.WriteError(w, status, err.Error())
		return false
	}
	gm, err := s.queries.IsGmAtLiveTableForCharacter(r.Context(), sqlcgen.IsGmAtLiveTableForCharacterParams{
		CharacterId: id, OwnerId: user.ID,
	})
	if err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not check the character's sessions")
		return false
	}
	if !gm {
		httpio.WriteError(w, http.StatusForbidden, fmt.Sprintf(
			"Ending the scene or the day for character %d is the GM's, during a live session", id))
		return false
	}
	return true
}

// clearEffectScopes roda um dos helpers de domínio que expiram escopo para o
// personagem `{id}` e responde com os escopos que o cliente deve descartar da
// ficha em cache — um delta, para a tela atualizar sem rebuscar.
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
		httpio.WriteError(w, status, err.Error())
		return
	}
	httpio.WriteJSON(w, http.StatusOK, map[string][]string{"clearedScopes": cleared})
}

// handleEndScene é o "Encerrar cena" da própria ficha (aba Efeitos): um jogador
// encerrando a cena dele, ao contrário do descanso da mesa inteira que o mestre
// dispara.
func (s *Server) handleEndScene(w http.ResponseWriter, r *http.Request) {
	s.clearEffectScopes(w, r, s.tableRules().EndScene, []string{"scene"})
}

// handleEndDay encerra o dia, o que encerra a cena em curso junto (é o descanso
// do livro) — daí os dois escopos no delta.
func (s *Server) handleEndDay(w http.ResponseWriter, r *http.Request) {
	s.clearEffectScopes(w, r, s.tableRules().endDay, []string{"scene", "day"})
}

// restVitals aplica a recuperação do descanso noturno do T20: PV e PM ganham
// cada um floor(nível × fator), aparado no máximo, e grava. Devolve os novos
// valores atuais para quem chama espelhá-los no rastreador vivo.
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
		HpCurrent: next.hpCurrent, MpCurrent: next.mpCurrent, UpdatedAt: dbvalue.NowISO(), ID: characterID,
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
