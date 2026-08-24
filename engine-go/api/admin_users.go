package api

// Tela de administração (ALE-120): quem está na mesa, o que cada um tem, e as
// duas ações que sobram sobre uma conta — mandar um link de redefinição de
// senha e apagar. Tudo atrás de requireAdmin; a UI só decide o que MOSTRAR.

import (
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
	caller := currentUser(r)
	if id == caller.ID {
		// Not paranoia: the admin list shows your own row, and the menu is the
		// same one. Deleting yourself would take your mesas nowhere.
		plataforma.WriteError(w, http.StatusBadRequest, "You cannot delete your own account")
		return
	}
	moved, err := s.deleteUserKeepingMesas(r, id, caller.ID)
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not delete user")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "transferredCampaigns": moved})
}

// deleteUserKeepingMesas moves the campaigns and deletes the account in ONE
// transaction: a half-done delete would Leave mesas owned by a row that no
// longer exists.
func (s *Server) deleteUserKeepingMesas(r *http.Request, UserID, newOwnerID int64) (int64, error) {
	ctx := r.Context()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)
	moved, err := q.TransferCampaigns(ctx, sqlcgen.TransferCampaignsParams{
		NewOwnerId: newOwnerID, UpdatedAt: plataforma.NowISO(), OldOwnerId: UserID,
	})
	if err != nil {
		return 0, err
	}
	if err := q.DeleteUser(ctx, UserID); err != nil {
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
	if _, err := s.queries.GetUserByID(r.Context(), id); err != nil {
		plataforma.WriteError(w, http.StatusNotFound, "User not found")
		return
	}
	now := time.Now()
	Reset, err := s.queries.CreatePasswordReset(r.Context(), sqlcgen.CreatePasswordResetParams{
		Token:     generateInviteToken(),
		Userid:    id,
		Createdby: currentUser(r).ID,
		Createdat: plataforma.IsoAt(now),
		Expiresat: plataforma.IsoAt(now.Add(passwordResetTTL)),
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create Reset link")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, accountInviteDTO{Token: Reset.Token, ExpiresAt: Reset.Expiresat})
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
