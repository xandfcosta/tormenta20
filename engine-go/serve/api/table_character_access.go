package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"t20engine/app/rest"

	"t20engine/infra/db/sqlcgen"
)

// A PORTA DE ENTRADA DE UM PERSONAGEM, do lado da MESA.
//
// Os dois guardas moravam no `character.go`, que era o arquivo de três donos do
// pacote — `Server`, `tableRules` e `sheetRules` juntos, e o nome não dizia
// nenhum dos três (ALE-330). Quem decide quem lê e quem escreve a ficha de
// outro é a mesa, e é por isso que eles são do `tableRules`: o dono passa, o
// mestre da campanha passa, o administrador passa, e mais ninguém.

// authorizedCharacter carrega a ficha e cobra a trava de leitura/escrita.
//
// O CORPO mora no `app/rest` (ALE-344), e o que sobra aqui é o número do HTTP.
// A regra é a mesma que o descanso do grupo usa, e uma segunda cópia dela
// divergiria em silêncio — o sintoma seria a mesa deixando entrar quem a ficha
// barra.
func (tr tableRules) authorizedCharacter(ctx context.Context, user AuthUser, id int64) (sqlcgen.Character, int, error) {
	row, err := rest.NewAccess(tr.queries).Character(ctx, callerOf(user), id)
	if err != nil {
		return row, statusForAccess(err), err
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
