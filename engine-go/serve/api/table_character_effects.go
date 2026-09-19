package api

import (
	"context"
	"fmt"
	"net/http"
	"t20engine/app/rest"
	"t20engine/infra/httpio"

	"t20engine/infra/db/sqlcgen"
)

// EndScene e endDay expiram os escopos da ficha. Os dois DELEGAM: o corpo mora
// no `app/rest` (ALE-344), e o que sobra é o número do HTTP.
//
// Eles ficam com esta forma — `(int, error)` — porque é a que o
// `clearEffectScopes` recebe por parâmetro, e ele serve os dois.
func (tr tableRules) EndScene(ctx context.Context, user AuthUser, characterID int64) (int, error) {
	if err := rest.NewScopes(tr.queries, tr.catalogs).EndScene(ctx, callerOf(user), characterID); err != nil {
		return statusForAccess(err), err
	}
	return http.StatusOK, nil
}

func (tr tableRules) endDay(ctx context.Context, user AuthUser, characterID int64) (int, error) {
	if err := rest.NewScopes(tr.queries, tr.catalogs).EndDay(ctx, callerOf(user), characterID); err != nil {
		return statusForAccess(err), err
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
