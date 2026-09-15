package api

import (
	"context"
	"fmt"
	"net/http"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/httpio"

	"github.com/go-chi/chi/v5"
)

// handleListCharacters returns the caller's own characters (newest-updated first),
// each as the full aggregate —
func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	out, err := s.characterList(r.Context(), currentUser(r).ID)
	if err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not list characters")
		return
	}
	httpio.WriteJSON(w, http.StatusOK, out)
}

// characterList é o elenco de quem chama, agregado.
//
// Transport-agnostic, e esta é a SEXTA vez que a migração encontra a mesma
// forma — depois do `selfInitiativeEntry`, do `deleteAccount`, do trio da porta,
// do `mintAccountInvite` e do `campaignList`. Seis é padrão, não anedota: uma
// base com exatamente um transporte não tem por que separar regra de handler, e
// o segundo transporte é o que cobra a conta (ALE-239).
func (s *Server) characterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error) {
	rows, err := s.queries.ListCharactersByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	out := make([]sheet.CharacterDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := s.sheetRules().LoadCharacter(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

// characterFor is the preamble every character route repeats: read {id}, load
// the row, enforce the read/mutation guard, and emit the right error. Returns
// ok=false when it already wrote the response.
//
// Twenty-three handlers spelled these nine lines out, and the copy-paste left a
// real inconsistency behind it: three of them (items, expertises, spells)
// decoded the request body BEFORE authorizing, while the rest authorized first.
// Going through one helper forces a single order — authorize, then read the
// body — so an unauthorized caller can never reach a decoder.
//
// @example row, ok := s.characterFor(w, r); if !ok { return }
func (s *Server) characterFor(w http.ResponseWriter, r *http.Request) (sqlcgen.Character, bool) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return sqlcgen.Character{}, false
	}
	row, status, err := s.tableRules().authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		httpio.WriteError(w, status, err.Error())
		return sqlcgen.Character{}, false
	}
	return row, true
}

// intParam parses a chi :id-style path param, writing a 400 (like ParseIntPipe)
// and returning false on a non-numeric value.
func intParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	n, err := httpio.ParseInt(chi.URLParam(r, name))
	if err != nil {
		httpio.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Validation failed (numeric string is expected for %q)", name))
		return 0, false
	}
	return int64(n), true
}
