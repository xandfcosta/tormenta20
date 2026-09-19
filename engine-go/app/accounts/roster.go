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

func NewRoster(portao Gate) Roster { return Roster{gate: portao} }

// Delete apaga a conta e passa as campanhas dela a quem apagou, numa transação
// só — meia exclusão deixaria mesas com um dono que não existe mais.
//
// Devolve QUANTAS campanhas mudaram de mão, que é o número que a tela diz em voz
// alta antes de confirmar.
func (r Roster) Delete(ctx context.Context, quemPede, contaID int64) (int64, error) {
	if contaID == quemPede {
		return 0, ErrCannotDeleteSelf
	}
	tx, err := r.gate.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("abrir a transação da exclusão: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := r.gate.queries.WithTx(tx)
	movidas, err := q.TransferCampaigns(ctx, sqlcgen.TransferCampaignsParams{
		NewOwnerId: quemPede, UpdatedAt: dbvalue.NowISO(), OldOwnerId: contaID,
	})
	if err != nil {
		return 0, fmt.Errorf("passar as campanhas da conta %d para %d: %w", contaID, quemPede, err)
	}
	if err := q.DeleteUser(ctx, contaID); err != nil {
		return 0, fmt.Errorf("apagar a conta %d: %w", contaID, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("fechar a transação da exclusão: %w", err)
	}
	return movidas, nil
}
