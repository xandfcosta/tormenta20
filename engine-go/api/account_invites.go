package api

// Convite de CONTA (ALE-120). Serving the table on the LAN made open
// registration a real door: anyone reaching http://<ip>:3001 could create an
// account. Now the admin issues a single-use link and hands it to the player,
// who still picks their own password — the admin never sees it.
//
// Not to be confused with the CAMPAIGN invite (campaigns.inviteToken), which
// brings an EXISTING user into a mesa. This one is what makes the account exist,
// which is why it lives on its own route and is spent on use.

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"t20engine/db/sqlcgen"
)

// accountInviteTTL is deliberately short-ish: the link is passed hand to hand at
// the table, not mailed, so a week is generous and a stale link left in a chat
// history stops working.
const accountInviteTTL = 7 * 24 * time.Hour

// inviteRejected is the ONE answer for unknown, spent and expired alike — the
// caller is anonymous, and telling them which of the three it was only helps
// someone probing tokens.
const inviteRejected = "Invite is invalid or expired"

type accountInviteDTO struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

// handleCreateAccountInvite issues a fresh invite: POST /admin/invites (admin only).
func (s *Server) handleCreateAccountInvite(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	invite, err := s.queries.CreateAccountInvite(r.Context(), sqlcgen.CreateAccountInviteParams{
		Token:     generateInviteToken(),
		Createdby: currentUser(r).ID,
		Createdat: isoAt(now),
		Expiresat: isoAt(now.Add(accountInviteTTL)),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create invite")
		return
	}
	writeJSON(w, http.StatusCreated, accountInviteDTO{Token: invite.Token, ExpiresAt: invite.Expiresat})
}

// handleResolveAccountInvite answers whether a link still works: GET
// /account-invites/{token}. Anonymous by necessity — it is read BEFORE the
// account exists, so the register screen can tell "peça um convite" apart from
// "esse link já foi usado" without the player submitting a form first.
func (s *Server) handleResolveAccountInvite(w http.ResponseWriter, r *http.Request) {
	invite, ok := s.usableInvite(r.Context(), chi.URLParam(r, "token"))
	if !ok {
		writeError(w, http.StatusNotFound, inviteRejected)
		return
	}
	writeJSON(w, http.StatusOK, accountInviteDTO{Token: invite.Token, ExpiresAt: invite.Expiresat})
}

// usableInvite loads an invite that can still be spent: it exists, nobody used
// it, and it has not expired.
func (s *Server) usableInvite(ctx context.Context, token string) (sqlcgen.AccountInvite, bool) {
	if token == "" {
		return sqlcgen.AccountInvite{}, false
	}
	invite, err := s.queries.GetAccountInvite(ctx, token)
	if err != nil || invite.Usedat.Valid {
		return sqlcgen.AccountInvite{}, false
	}
	expiresAt, err := time.Parse(isoLayout, invite.Expiresat)
	if err != nil || time.Now().UTC().After(expiresAt) {
		return sqlcgen.AccountInvite{}, false
	}
	return invite, true
}

// errInviteSpent means someone else used the link between the check and the
// insert. It reaches the player as the same rejection as a stale link.
var errInviteSpent = errors.New("invite already spent")

// createUser inserts the account and, when the registration came from an invite,
// spends it in the SAME transaction. That is what makes single use an invariant
// instead of a hope: two players opening the same link at once both pass the
// read check, but only one UPDATE finds `usedAt IS NULL`, and the loser's
// account is rolled back with it (ALE-120).
//
// invite is nil when an ADMIN_EMAILS address bootstraps its own account.
func (s *Server) createUser(
	ctx context.Context, params sqlcgen.CreateUserParams, invite *sqlcgen.AccountInvite,
) (sqlcgen.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return sqlcgen.User{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := s.queries.WithTx(tx)
	user, err := q.CreateUser(ctx, params)
	if err != nil {
		return sqlcgen.User{}, err
	}
	if invite != nil {
		if err := spend(ctx, q, invite.ID, user.ID); err != nil {
			return sqlcgen.User{}, err
		}
	}
	return user, tx.Commit()
}

func spend(ctx context.Context, q *sqlcgen.Queries, inviteID, userID int64) error {
	rows, err := q.SpendAccountInvite(ctx, sqlcgen.SpendAccountInviteParams{
		Usedat: sql.NullString{String: nowISO(), Valid: true},
		Usedby: sql.NullInt64{Int64: userID, Valid: true},
		ID:     inviteID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return errInviteSpent
	}
	return nil
}
