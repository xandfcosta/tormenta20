package api

// Tela de administração: quem está na mesa, o que cada um tem, e as
// duas ações que sobram sobre uma conta — mandar um link de redefinição de
// senha e apagar. Tudo atrás de requireAdmin; a UI só decide o que MOSTRAR.

import (
	"context"
	"errors"
	"net/http"
	"t20engine/infra/platform"
	"time"

	"t20engine/infra/db/sqlcgen"
)

// passwordResetTTL is short on purpose: unlike an invite, this link opens an
// EXISTING account, so a stale one left in a chat is worth more to a stranger.
const passwordResetTTL = 24 * time.Hour

type adminUserDTO struct {
	ID         int64   `json:"id"`
	Email      string  `json:"email"`
	Name       *string `json:"name"`
	IsAdmin    bool    `json:"isAdmin"`
	Campaigns  int64   `json:"campaigns"`
	Characters int64   `json:"characters"`
	CreatedAt  string  `json:"createdAt"`
}

// deleteAccount is the RULE behind the delete: you cannot remove your own
// account, and the campaigns move to whoever removes it.
//
// Transport-agnostic on purpose: two surfaces need this rule, and welding it to
// the HTTP handler would leave the second one calling its own route from the
// inside or copying the rule.
func (h adminHost) deleteAccount(r *http.Request, id, callerID int64) (int64, int, error) {
	if id == callerID {
		// Not paranoia: the admin list shows your own row, and the menu is the
		// same one. Deleting yourself would take your mesas nowhere.
		return 0, http.StatusBadRequest, errors.New("You cannot delete your own account")
	}
	moved, err := h.deleteUserKeepingCampaigns(r, id, callerID)
	if err != nil {
		return 0, http.StatusInternalServerError, errors.New("Could not delete user")
	}
	return moved, http.StatusOK, nil
}

// deleteUserKeepingCampaigns moves the campaigns and deletes the account in ONE
// transaction: a half-done delete would leave mesas owned by a row that no
// longer exists.
func (h adminHost) deleteUserKeepingCampaigns(r *http.Request, userID, newOwnerID int64) (int64, error) {
	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	q := h.queries.WithTx(tx)
	moved, err := q.TransferCampaigns(ctx, sqlcgen.TransferCampaignsParams{
		NewOwnerId: newOwnerID, UpdatedAt: platform.NowISO(), OldOwnerId: userID,
	})
	if err != nil {
		return 0, err
	}
	if err := q.DeleteUser(ctx, userID); err != nil {
		return 0, err
	}
	return moved, tx.Commit()
}

// errUserNotFound separa "não existe" de "deu errado" para quem CHAMA
// decidir o que dizer: a rota JSON responde 404, a cena do app desenha um
// aviso. A regra não sabe qual é o transporte, e é esse o ponto.
var errUserNotFound = errors.New("usuário não existe")

// mintPasswordReset cunha o link de uso único que o admin entrega.
//
// A REGRA está aqui e não no manipulador HTTP: duas telas cunham o mesmo link, e
// com a conta de validade dentro de um `http.HandlerFunc` a segunda só teria
// duas saídas — chamar a própria rota por dentro, ou copiar a conta.
//
// O prazo é 24h contra os 7 dias do convite, e a diferença é de risco: o
// convite abre uma conta que ainda NÃO existe, este abre uma que já existe e
// tem fichas dentro. Um link esquecido numa conversa vale mais para um
// estranho.
func (h adminHost) mintPasswordReset(ctx context.Context, usuarioID, criadoPor int64) (sqlcgen.PasswordReset, error) {
	if _, err := h.queries.GetUserByID(ctx, usuarioID); err != nil {
		return sqlcgen.PasswordReset{}, errUserNotFound
	}
	now := time.Now()
	return h.queries.CreatePasswordReset(ctx, sqlcgen.CreatePasswordResetParams{
		Token:     generateInviteToken(),
		Userid:    usuarioID,
		Createdby: criadoPor,
		Createdat: platform.IsoAt(now),
		Expiresat: platform.IsoAt(now.Add(passwordResetTTL)),
	})
}
