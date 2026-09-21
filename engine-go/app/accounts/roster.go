package accounts

import (
	"context"
	"fmt"

	"t20engine/app"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// ErrCannotDeleteSelf é a conta de quem pede.
//
// Não é paranoia: a lista da administração mostra a SUA linha, com o mesmo menu
// das outras, e apagar-se levaria as suas mesas para lugar nenhum — o `Delete`
// transfere as campanhas para quem apaga, e não há para quem transferir quando
// os dois são a mesma pessoa.
var ErrCannotDeleteSelf = fmt.Errorf("ninguém apaga a própria conta: %w", app.ErrRefused)

// Roster é o ELENCO de contas do servidor visto por quem administra.
type Roster struct {
	gate Gate
}

func NewRoster(gate Gate) Roster { return Roster{gate: gate} }

// Delete apaga a conta e passa as campanhas dela a quem apagou, numa transação
// só — meia exclusão deixaria mesas com um dono que não existe mais.
//
// Devolve QUANTAS campanhas mudaram de mão, que é o número que a tela diz em voz
// alta antes de confirmar.
func (r Roster) Delete(ctx context.Context, requester, accountID int64) (int64, error) {
	if accountID == requester {
		return 0, ErrCannotDeleteSelf
	}
	tx, err := r.gate.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("abrir a transação da exclusão: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := r.gate.queries.WithTx(tx)
	moved, err := q.TransferCampaigns(ctx, sqlcgen.TransferCampaignsParams{
		NewOwnerId: requester, UpdatedAt: dbvalue.NowISO(), OldOwnerId: accountID,
	})
	if err != nil {
		return 0, fmt.Errorf("passar as campanhas da conta %d para %d: %w", accountID, requester, err)
	}
	if err := q.DeleteUser(ctx, accountID); err != nil {
		return 0, fmt.Errorf("apagar a conta %d: %w", accountID, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("fechar a transação da exclusão: %w", err)
	}
	return moved, nil
}
