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
	"time"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/secret"
)

// accountInviteTTL é curto de propósito: o link passa de mão em mão na mesa e
// não por e-mail, então uma semana é generosa — e um link esquecido num
// histórico de conversa para de funcionar.
const accountInviteTTL = 7 * 24 * time.Hour

// mintAccountInvite cunha o link de uso único.
//
// Independente de TRANSPORTE, para o handler HTTP e a cena em templ lerem a
// mesma regra.
func mintAccountInvite(ctx context.Context, q *sqlcgen.Queries, criadoPor int64) (sqlcgen.AccountInvite, error) {
	token, err := secret.Token()
	if err != nil {
		return sqlcgen.AccountInvite{}, err
	}
	now := time.Now()
	return q.CreateAccountInvite(ctx, sqlcgen.CreateAccountInviteParams{
		Token:     token,
		Createdby: criadoPor,
		Createdat: dbvalue.IsoAt(now),
		Expiresat: dbvalue.IsoAt(now.Add(accountInviteTTL)),
	})
}
