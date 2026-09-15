package api

// Convite de CONTA. A mesa é servida na LAN, então cadastro aberto seria uma
// porta de verdade: qualquer um que alcançasse o endereço criaria conta. O admin
// cunha um link de uso único e o entrega ao jogador, que ainda escolhe a própria
// senha — o admin nunca a vê.
//
// Não confundir com o convite de CAMPANHA (`campaigns.inviteToken`), que traz um
// usuário EXISTENTE para uma mesa. Este é o que faz a conta existir, e é por
// isso que ele mora numa rota própria e se gasta ao ser usado.

import (
	"context"
	"database/sql"
	"errors"
	"t20engine/infra/platform"
	"time"

	"t20engine/infra/db/sqlcgen"
)

// accountInviteTTL é curto de propósito: o link passa de mão em mão na mesa e
// não por e-mail, então uma semana é generosa — e um link esquecido num
// histórico de conversa para de funcionar.
const accountInviteTTL = 7 * 24 * time.Hour

// inviteRejected é a ÚNICA resposta para desconhecido, gasto e vencido — quem
// chama é anônimo, e dizer qual dos três foi só ajuda quem está sondando
// tokens.
const inviteRejected = "Invite is invalid or expired"

type accountInviteDTO struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

// mintAccountInvite cunha o link de uso único.
//
// Independente de TRANSPORTE, para o handler HTTP e a cena em templ lerem a
// mesma regra.
func mintAccountInvite(ctx context.Context, q *sqlcgen.Queries, criadoPor int64) (sqlcgen.AccountInvite, error) {
	now := time.Now()
	return q.CreateAccountInvite(ctx, sqlcgen.CreateAccountInviteParams{
		Token:     generateInviteToken(),
		Createdby: criadoPor,
		Createdat: platform.IsoAt(now),
		Expiresat: platform.IsoAt(now.Add(accountInviteTTL)),
	})
}

// usableInvite carrega um convite que ainda pode ser gasto: ele existe, ninguém
// o usou, e não venceu.
func (a accountRules) usableInvite(ctx context.Context, token string) (sqlcgen.AccountInvite, bool) {
	if token == "" {
		return sqlcgen.AccountInvite{}, false
	}
	invite, err := a.queries.GetAccountInvite(ctx, token)
	if err != nil || invite.Usedat.Valid {
		return sqlcgen.AccountInvite{}, false
	}
	expiresAt, err := time.Parse(platform.IsoLayout, invite.Expiresat)
	if err != nil || time.Now().UTC().After(expiresAt) {
		return sqlcgen.AccountInvite{}, false
	}
	return invite, true
}

// errInviteSpent quer dizer que outra pessoa usou o link entre a conferência e o
// insert. Ele chega ao jogador como a MESMA recusa de um link vencido.
var errInviteSpent = errors.New("invite already spent")

// createUser insere a conta e, quando o cadastro veio de um convite, o GASTA na
// MESMA transação. É o que faz o uso único ser invariante em vez de esperança:
// dois jogadores abrindo o mesmo link ao mesmo tempo passam os dois pela
// conferência de leitura, mas só um `UPDATE` acha `usedAt IS NULL`, e a conta do
// perdedor volta atrás junto.
//
// `invite` é nulo quando um endereço de `ADMIN_EMAILS` cria a própria conta.
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
		Usedat: sql.NullString{String: platform.NowISO(), Valid: true},
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
