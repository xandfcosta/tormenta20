package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"t20engine/infra/db/sqlcgen"
)

// A PORTA DE ENTRADA DE UM PERSONAGEM, do lado da MESA.
//
// Os dois guardas moravam no `character.go`, que era o arquivo de três donos do
// pacote — `Server`, `tableRules` e `sheetRules` juntos, e o nome não dizia
// nenhum dos três (ALE-330). Quem decide quem lê e quem escreve a ficha de
// outro é a mesa, e é por isso que eles são do `tableRules`: o dono passa, o
// mestre da campanha passa, o administrador passa, e mais ninguém.

// authorizedCharacter loads a character and enforces the read/mutation guard
// (owner or campaign GM). Returns the row, or an HTTP status + error to emit.
func (tr tableRules) authorizedCharacter(ctx context.Context, user AuthUser, id int64) (sqlcgen.Character, int, error) {
	row, err := tr.queries.GetCharacter(ctx, id)
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
	isGm, err := tr.queries.IsCampaignGmForCharacter(ctx, sqlcgen.IsCampaignGmForCharacterParams{
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
func (tr tableRules) assertCharacterOwner(ctx context.Context, userID, characterID int64) (int, error) {
	owner, err := tr.queries.GetCharacterOwner(ctx, characterID)
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
