package api

import (
	"context"
	"net/http"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"
	"t20engine/db/sqlcgen"
	"t20engine/sheet"
)

// O ESTADO DE JOGO da ficha vive em `sheet` desde a ALE-278: `sheet.sheet.PowerUseDTO`
// e `sheet.sheet.StanceDTO` são forma de DADO e viajam dentro do `sheet.CharacterDTO`.
// O que ficou aqui são os quatro handlers que os gravam, que é encanamento.

// handleUpdateConditionals substitui o CONJUNTO inteiro, como o irmão das
// condições do livro faz.
//
// Conjunto inteiro e não "ligue este"/"desligue aquele": a tela sempre sabe o
// estado final, e um toggle por requisição faria dois cliques rápidos chegarem
// fora de ordem e deixarem a ficha no oposto do que se pediu.
//
// Nenhuma validação contra catálogo, e isto é deliberado: além de Fúria e Ataque
// Poderoso, o jogador liga situacionais HOMEBREW, que catálogo nenhum conhece.
// Um id desconhecido não casa com modificador nenhum no motor e vira nada — que
// é melhor que recusar a requisição no meio do combate.
func (s *Server) handleUpdateConditionals(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		Conditionals []string `json:"conditionals"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.Conditionals == nil {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"conditionals": {"conditionals must be an array"}})
		return
	}
	ctx := r.Context()
	if err := s.queries.ClearCharacterConditionals(ctx, row.ID); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update conditionals")
		return
	}
	for _, id := range body.Conditionals {
		if err := s.queries.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
			Characterid: row.ID, Conditionalid: id,
		}); err != nil {
			plataforma.WriteError(w, http.StatusInternalServerError, "Could not update conditionals")
			return
		}
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string][]string{"conditionals": body.Conditionals})
}

// handleBumpPowerUse gasta MAIS UM uso.
//
// O corpo não carrega o total, e essa é a diferença que importa: dois cliques
// rápidos mandando "agora são 3" gravam 3 duas vezes e perdem um uso, enquanto
// dois "gastei mais um" somam 2. O incremento mora no `ON CONFLICT` da query,
// então nem o read-modify-write existe.
func (s *Server) handleBumpPowerUse(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		PowerID string `json:"powerId"`
		Scope   string `json:"scope"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.PowerID == "" {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"powerId": {"powerId is required"}})
		return
	}
	if body.Scope != "scene" && body.Scope != "day" {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"scope": {`scope must be "scene" or "day"`}})
		return
	}
	if err := s.queries.BumpCharacterPowerUse(r.Context(), sqlcgen.BumpCharacterPowerUseParams{
		Characterid: row.ID, Powerid: body.PowerID, Scope: body.Scope,
	}); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not record the power use")
		return
	}
	s.writePlayState(w, r, row.ID)
}

// handleSetStance registra o que foi pago para entrar na postura.
func (s *Server) handleSetStance(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	flag := chi.URLParam(r, "flag")
	if flag == "" {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"flag": {"flag is required"}})
		return
	}
	var body struct {
		Steps  int64 `json:"steps"`
		PmPaid int64 `json:"pmPaid"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.Steps < 0 || body.PmPaid < 0 {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"pmPaid": {"steps and pmPaid must be zero or more"}})
		return
	}
	if err := s.queries.UpsertCharacterStance(r.Context(), sqlcgen.UpsertCharacterStanceParams{
		Characterid: row.ID, Flag: flag, Steps: body.Steps, Pmpaid: body.PmPaid,
	}); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not record the stance")
		return
	}
	s.writePlayState(w, r, row.ID)
}

// handleDeleteStance esquece o pagamento — sair da postura.
func (s *Server) handleDeleteStance(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	if err := s.queries.RemoveCharacterStance(r.Context(), sqlcgen.RemoveCharacterStanceParams{
		Characterid: row.ID, Flag: chi.URLParam(r, "flag"),
	}); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not clear the stance")
		return
	}
	s.writePlayState(w, r, row.ID)
}

// writePlayState devolve o estado de jogo INTEIRO depois de uma escrita.
//
// Devolver o estado e não um "ok" é o que deixa a tela conferir o próprio
// otimismo: ela pinta antes, e o que volta é a verdade do servidor.
func (s *Server) writePlayState(w http.ResponseWriter, r *http.Request, id int64) {
	var dto sheet.CharacterDTO
	if err := sheet.LoadPlayState(r.Context(), s.queries, id, &dto); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not read the play state")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{
		"conditionals": dto.Conditionals,
		"powerUses":    dto.PowerUses,
		"stances":      dto.Stances,
	})
}

// clearScenePlayState é o que o DESCANSO DE CENA leva embora: os usos "1/cena" e
// as posturas.
//
// Ele se pendura no descanso da ficha e não no `EndScene` da sessão de
// propósito: os usos entram pelo caminho que JÁ limpa a ficha, junto dos
// efeitos. Era o `EndScene` da sessão que estava errado — ele não limpava
// efeito nenhum, e a bênção de duração "cena" sobrevivia ao fim da cena. A
// ALE-220 fechou isso pelo lado de lá: encerrar a cena agora percorre o grupo e
// chama ESTE caminho para cada ficha.
func (s *Server) clearScenePlayState(ctx context.Context, id int64) error {
	if err := s.queries.ClearCharacterPowerUsesByScope(ctx, sqlcgen.ClearCharacterPowerUsesByScopeParams{
		Characterid: id, Scope: "scene",
	}); err != nil {
		return err
	}
	return s.queries.ClearCharacterStances(ctx, id)
}

// clearDayPlayState é o DESCANSO DE DIA: leva o da cena e mais os usos "1/dia".
func (s *Server) clearDayPlayState(ctx context.Context, id int64) error {
	if err := s.clearScenePlayState(ctx, id); err != nil {
		return err
	}
	return s.queries.ClearCharacterPowerUsesByScope(ctx, sqlcgen.ClearCharacterPowerUsesByScopeParams{
		Characterid: id, Scope: "day",
	})
}
