package api

// Tela de administração (ALE-120): quem está na mesa, o que cada um tem, e as
// duas ações que sobram sobre uma conta — mandar um link de redefinição de
// senha e apagar. Tudo atrás de requireAdmin; a UI só decide o que MOSTRAR.

import (
	"context"
	"errors"
	"net/http"
	"t20engine/plataforma"
	"time"

	"t20engine/db/sqlcgen"
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

// handleAdminListUsers: GET /admin/users — every account with what it owns, so
// the screen can say what deleting one would cost before it happens.
func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListUsersWithCounts(r.Context())
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not list users")
		return
	}
	out := make([]adminUserDTO, 0, len(rows))
	for _, u := range rows {
		out = append(out, adminUserDTO{
			ID: u.ID, Email: u.Email, Name: plataforma.NullToPtr(u.Name), IsAdmin: s.cfg.IsAdmin(u.Email),
			Campaigns: u.Campaigns, Characters: u.Characters, CreatedAt: u.Createdat,
		})
	}
	plataforma.WriteJSON(w, http.StatusOK, out)
}

// handleAdminDeleteUser: DELETE /admin/users/{id}. The mesas the account owns
// move to the CALLER before it goes, so the chronicle survives the player
// leaving the table — the decision the owner made when this was designed.
// Their characters do go with them (the rows point at the account).
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	moved, status, err := s.deleteAccount(r, id, currentUser(r).ID)
	if err != nil {
		plataforma.WriteError(w, status, err.Error())
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "transferredCampaigns": moved})
}

// deleteAccount is the RULE behind the delete: you cannot remove your own
// account, and the campaigns move to whoever removes it.
//
// Transport-agnostic on purpose. It was extracted when the pilot's admin screen
// (ALE-219) needed the same rule from a second transport and found it welded to
// the HTTP handler — the SECOND time the pilot hit that shape, after
// `selfInitiativeEntry`, which was welded to the socket gateway. Two surfaces,
// two rules pinned to whichever transport reached them first.
func (s *Server) deleteAccount(r *http.Request, id, callerID int64) (int64, int, error) {
	if id == callerID {
		// Not paranoia: the admin list shows your own row, and the menu is the
		// same one. Deleting yourself would take your mesas nowhere.
		return 0, http.StatusBadRequest, errors.New("You cannot delete your own account")
	}
	moved, err := s.deleteUserKeepingCampaigns(r, id, callerID)
	if err != nil {
		return 0, http.StatusInternalServerError, errors.New("Could not delete user")
	}
	return moved, http.StatusOK, nil
}

// deleteUserKeepingCampaigns moves the campaigns and deletes the account in ONE
// transaction: a half-done delete would leave mesas owned by a row that no
// longer exists.
func (s *Server) deleteUserKeepingCampaigns(r *http.Request, userID, newOwnerID int64) (int64, error) {
	ctx := r.Context()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)
	moved, err := q.TransferCampaigns(ctx, sqlcgen.TransferCampaignsParams{
		NewOwnerId: newOwnerID, UpdatedAt: plataforma.NowISO(), OldOwnerId: userID,
	})
	if err != nil {
		return 0, err
	}
	if err := q.DeleteUser(ctx, userID); err != nil {
		return 0, err
	}
	return moved, tx.Commit()
}

// handleAdminCreatePasswordReset: POST /admin/users/{id}/password-reset. The
// admin never sees or types a password — they hand over a single-use link and
// the player picks their own.
func (s *Server) handleAdminCreatePasswordReset(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	reset, err := s.mintPasswordReset(r.Context(), id, currentUser(r).ID)
	if errors.Is(err, errUserNotFound) {
		plataforma.WriteError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create reset link")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, accountInviteDTO{Token: reset.Token, ExpiresAt: reset.Expiresat})
}

// errUserNotFound separa "não existe" de "deu errado" para quem CHAMA
// decidir o que dizer: a rota JSON responde 404, a cena do piloto desenha um
// aviso. A regra não sabe qual é o transporte, e é esse o ponto.
var errUserNotFound = errors.New("usuário não existe")

// mintPasswordReset cunha o link de uso único que o admin entrega (ALE-120).
//
// A REGRA está aqui e não no manipulador HTTP, e esta é a SÉTIMA vez que a
// migração encontra o mesmo padrão — sete é padrão, não anedota. Duas telas
// precisam cunhar o mesmo link, e enquanto a conta de validade morava dentro
// de um `http.HandlerFunc` a segunda tela só tinha duas saídas: chamar a
// própria rota por dentro, ou copiar a conta.
//
// O prazo é 24h contra os 7 dias do convite, e a diferença é de risco: o
// convite abre uma conta que ainda NÃO existe, este abre uma que já existe e
// tem fichas dentro. Um link esquecido numa conversa vale mais para um
// estranho.
func (s *Server) mintPasswordReset(ctx context.Context, usuarioID, criadoPor int64) (sqlcgen.PasswordReset, error) {
	if _, err := s.queries.GetUserByID(ctx, usuarioID); err != nil {
		return sqlcgen.PasswordReset{}, errUserNotFound
	}
	now := time.Now()
	return s.queries.CreatePasswordReset(ctx, sqlcgen.CreatePasswordResetParams{
		Token:     generateInviteToken(),
		Userid:    usuarioID,
		Createdby: criadoPor,
		Createdat: plataforma.IsoAt(now),
		Expiresat: plataforma.IsoAt(now.Add(passwordResetTTL)),
	})
}

// handleAdminListInvites: GET /admin/invites — the links already handed out and
// still good, so the admin can copy one again instead of minting a second.
func (s *Server) handleAdminListInvites(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListOpenAccountInvites(r.Context(), plataforma.NowISO())
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not list invites")
		return
	}
	out := make([]accountInviteDTO, 0, len(rows))
	for _, i := range rows {
		out = append(out, accountInviteDTO{Token: i.Token, ExpiresAt: i.Expiresat})
	}
	plataforma.WriteJSON(w, http.StatusOK, out)
}
