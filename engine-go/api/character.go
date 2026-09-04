package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/sheet"

	"github.com/go-chi/chi/v5"
)

// handleListCharacters returns the caller's own characters (newest-updated first),
// each as the full aggregate —
func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	out, err := s.characterList(r.Context(), currentUser(r).ID)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not list characters")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, out)
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
		dto, err := s.LoadCharacter(ctx, row)
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
	row, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		plataforma.WriteError(w, status, err.Error())
		return sqlcgen.Character{}, false
	}
	return row, true
}

// authorizedCharacter loads a character and enforces the read/mutation guard
// (owner or campaign GM). Returns the row, or an HTTP status + error to emit.
func (s *Server) authorizedCharacter(ctx context.Context, user AuthUser, id int64) (sqlcgen.Character, int, error) {
	row, err := s.queries.GetCharacter(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return row, http.StatusNotFound, fmt.Errorf("Character %d not found", id)
	}
	if err != nil {
		return row, http.StatusInternalServerError, errors.New("Could not load character")
	}
	// The admin passes the same door as the owner and the campaign's GM: a table
	// they administer includes the sheets in it (ALE-120).
	if row.Ownerid == user.ID || user.IsAdmin {
		return row, http.StatusOK, nil
	}
	isGm, err := s.queries.IsCampaignGmForCharacter(ctx, sqlcgen.IsCampaignGmForCharacterParams{
		Characterid: id,
		Ownerid:     user.ID,
	})
	if err != nil {
		return row, http.StatusInternalServerError, errors.New("Could not check access")
	}
	if !isGm {
		return row, http.StatusForbidden, fmt.Errorf("Character %d belongs to another user", id)
	}
	return row, http.StatusOK, nil
}

// assertCharacterOwner is the strict owner-only check
// the WS vitals gate uses: a player may edit only a character they own. Transport-agnostic.
func (s *Server) assertCharacterOwner(ctx context.Context, userID, characterID int64) (int, error) {
	owner, err := s.queries.GetCharacterOwner(ctx, characterID)
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, fmt.Errorf("Character %d not found", characterID)
	}
	if err != nil {
		return http.StatusInternalServerError, errors.New("Could not load character")
	}
	if owner != userID {
		return http.StatusForbidden, fmt.Errorf(
			"Caller %d can only edit their own character's vitals (character %d)", userID, characterID)
	}
	return http.StatusOK, nil
}

// intParam parses a chi :id-style path param, writing a 400 (like ParseIntPipe)
// and returning false on a non-numeric value.
func intParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	n, err := plataforma.ParseInt(chi.URLParam(r, name))
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Validation failed (numeric string is expected for %q)", name))
		return 0, false
	}
	return int64(n), true
}

// Os três invólucros abaixo existem para os chamadores de dentro do `api` não
// mudarem junto com a extração (ALE-278): a lógica mora no `sheet`, e aqui só
// se passa o que o `Server` tem na mão. Eles somem quando cada cena receber as
// dependências dela por construtor.

func (s *Server) LoadCharacter(ctx context.Context, c sqlcgen.Character) (sheet.CharacterDTO, error) {
	return sheet.Load(ctx, s.queries, c)
}

func (s *Server) ComputeSheet(ctx context.Context, row sqlcgen.Character) (engine.ComputedSheetV2, error) {
	return sheet.LoadAndCompute(ctx, s.queries, s.catalogs, row)
}
