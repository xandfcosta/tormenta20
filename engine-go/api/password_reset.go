package api

// Redefinição de senha por link (ALE-120). O admin gera, o jogador escolhe a
// própria senha — o admin nunca vê nem digita a senha de ninguém.
//
// As duas rotas daqui são ANÔNIMAS por necessidade: quem esqueceu a senha não
// consegue autenticar para trocá-la. O que as protege é o token, e ele é de uso
// único, expira em 24h e não diz nada sobre a conta quando é inválido.

import (
	"context"
	"errors"
	"net/http"
	"t20engine/plataforma"
	"time"

	"t20engine/db/sqlcgen"
)

const resetRejected = "Reset link is invalid or expired"

type resetPasswordBody struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// applyReset writes the new hash and spends the link in ONE transaction, and
// the spend is conditional — two people racing the same link cannot both set a
// password on the account.
func (a accountRules) applyReset(ctx context.Context, Reset sqlcgen.PasswordReset, hash string) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := a.queries.WithTx(tx)
	spent, err := q.SpendPasswordReset(ctx, sqlcgen.SpendPasswordResetParams{
		Usedat: plataforma.NullString(ptrTo(plataforma.NowISO())), ID: Reset.ID,
	})
	if err != nil {
		return err
	}
	if spent == 0 {
		return errResetSpent
	}
	if err := q.UpdateUserPassword(ctx, sqlcgen.UpdateUserPasswordParams{
		Passwordhash: hash, Updatedat: plataforma.NowISO(), ID: Reset.Userid,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

var errResetSpent = errors.New("Reset link already spent")

func writeResetError(w http.ResponseWriter, err error) {
	if errors.Is(err, errResetSpent) {
		plataforma.WriteError(w, http.StatusForbidden, resetRejected)
		return
	}
	plataforma.WriteError(w, http.StatusInternalServerError, "Could not Reset password")
}

// usableReset loads a link that can still be spent: it exists, nobody used it,
// and it has not expired.
func (a accountRules) usableReset(ctx context.Context, token string) (sqlcgen.PasswordReset, bool) {
	if token == "" {
		return sqlcgen.PasswordReset{}, false
	}
	Reset, err := a.queries.GetPasswordReset(ctx, token)
	if err != nil || Reset.Usedat.Valid {
		return sqlcgen.PasswordReset{}, false
	}
	expiresAt, err := time.Parse(plataforma.IsoLayout, Reset.Expiresat)
	if err != nil || time.Now().UTC().After(expiresAt) {
		return sqlcgen.PasswordReset{}, false
	}
	return Reset, true
}

func ptrTo[T any](v T) *T { return &v }
