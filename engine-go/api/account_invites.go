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
	"t20engine/plataforma"
	"time"

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

// mintAccountInvite cunha o link de uso único.
//
// Transport-agnostic, e esta é a QUARTA vez que a migração encontra a mesma
// forma — depois do `selfInitiativeEntry` (socket), do `deleteAccount` (handler
// HTTP) e do trio da porta (ALE-229). Já não é anedota: é o que uma base com
// exatamente um transporte parece por dentro, e o segundo transporte é o que
// torna isso visível.
func mintAccountInvite(ctx context.Context, q *sqlcgen.Queries, criadoPor int64) (sqlcgen.AccountInvite, error) {
	now := time.Now()
	return q.CreateAccountInvite(ctx, sqlcgen.CreateAccountInviteParams{
		Token:     generateInviteToken(),
		Createdby: criadoPor,
		Createdat: plataforma.IsoAt(now),
		Expiresat: plataforma.IsoAt(now.Add(accountInviteTTL)),
	})
}

// usableInvite loads an invite that can still be spent: it exists, nobody used
// it, and it has not expired.
func (a accountRules) usableInvite(ctx context.Context, token string) (sqlcgen.AccountInvite, bool) {
	if token == "" {
		return sqlcgen.AccountInvite{}, false
	}
	invite, err := a.queries.GetAccountInvite(ctx, token)
	if err != nil || invite.Usedat.Valid {
		return sqlcgen.AccountInvite{}, false
	}
	expiresAt, err := time.Parse(plataforma.IsoLayout, invite.Expiresat)
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
func (a accountRules) createUser(
	ctx context.Context, params sqlcgen.CreateUserParams, invite *sqlcgen.AccountInvite,
) (sqlcgen.User, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return sqlcgen.User{}, err
	}
	defer func() { _ = tx.Rollback() }()

	q := a.queries.WithTx(tx)
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
		Usedat: sql.NullString{String: plataforma.NowISO(), Valid: true},
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
