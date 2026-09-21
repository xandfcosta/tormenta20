package accounts

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"t20engine/app"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/secret"
)

// ErrBadResetLink cobre as TRÊS formas de um link de redefinição não servir:
// desconhecido, vencido e já gasto — inclusive o gasto que só a transação
// descobre.
//
// As três dizem a mesma coisa a quem clicou, e distinguir contaria a um
// estranho se um token existe.
var ErrBadResetLink = fmt.Errorf("este link não serve mais: %w", app.ErrForbidden)

// ErrUnknownAccount é a conta que não existe mais.
//
// Ela é separada do erro comum para quem CHAMA decidir o que dizer: a tela da
// administração desenha um aviso, e uma rota JSON responderia 404. O caso de uso
// não sabe qual é o transporte, e é esse o ponto.
var ErrUnknownAccount = fmt.Errorf("esta conta não existe: %w", app.ErrNotFound)

// passwordResetTTL é curto contra os sete dias do convite, e a diferença é de
// RISCO: o convite abre uma conta que ainda não existe, este abre uma que já
// existe e tem fichas dentro. Um link esquecido numa conversa vale mais para um
// estranho.
const passwordResetTTL = 24 * time.Hour

// Resets é a redefinição de senha por LINK.
//
// As duas pontas dela são ANÔNIMAS por necessidade: quem esqueceu a senha não
// consegue autenticar para trocá-la. O que as protege é o token — de uso único,
// vence em 24h, e não diz nada sobre a conta quando é inválido.
//
// O ADMINISTRADOR gera o link e nunca vê nem digita a senha de ninguém.
type Resets struct {
	gate Gate
}

func NewResets(gate Gate) Resets { return Resets{gate: gate} }

// Mint cunha o link de uso único que o administrador entrega em mãos.
//
// Ele confere que a conta existe ANTES de escrever a linha: um link apontando
// para um id apagado só falharia no clique de quem o recebeu, e aí já teria sido
// entregue.
func (r Resets) Mint(ctx context.Context, accountID, createdBy int64) (sqlcgen.PasswordReset, error) {
	if _, err := r.gate.queries.GetUserByID(ctx, accountID); err != nil {
		return sqlcgen.PasswordReset{}, ErrUnknownAccount
	}
	token, err := secret.Token()
	if err != nil {
		return sqlcgen.PasswordReset{}, err
	}
	now := time.Now()
	return r.gate.queries.CreatePasswordReset(ctx, sqlcgen.CreatePasswordResetParams{
		Token:     token,
		Userid:    accountID,
		Createdby: createdBy,
		Createdat: dbvalue.IsoAt(now),
		Expiresat: dbvalue.IsoAt(now.Add(passwordResetTTL)),
	})
}

// OwnerOfLink é de quem é a conta que este link redefine.
//
// Ela junta as duas perguntas que a tela faz — o link vale? de quem é? —
// porque repartidas a cena carregaria a linha do banco no meio só para ter o id
// do dono. O que interessa à tela é o E-MAIL, para quem clicou saber que está
// mudando a conta certa.
func (r Resets) OwnerOfLink(ctx context.Context, token string) (string, error) {
	link, ok := r.usable(ctx, token)
	if !ok {
		return "", ErrBadResetLink
	}
	user, err := r.gate.queries.GetUserByID(ctx, link.Userid)
	if err != nil {
		return "", ErrBadResetLink
	}
	return user.Email, nil
}

// Apply troca a senha e GASTA o link na mesma transação.
//
// O gasto é CONDICIONAL: duas pessoas correndo o mesmo link não podem as duas
// escolher a senha da conta. Quem perde recebe a mesma recusa de um link
// vencido.
func (r Resets) Apply(ctx context.Context, token, password string) error {
	link, ok := r.usable(ctx, token)
	if !ok {
		return ErrBadResetLink
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("gerar o hash da senha: %w", err)
	}

	tx, err := r.gate.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir a transação da redefinição: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := r.gate.queries.WithTx(tx)
	spent, err := q.SpendPasswordReset(ctx, sqlcgen.SpendPasswordResetParams{
		Usedat: dbvalue.NullString(&[]string{dbvalue.NowISO()}[0]), ID: link.ID,
	})
	if err != nil {
		return fmt.Errorf("gastar o link %d: %w", link.ID, err)
	}
	if spent == 0 {
		return ErrBadResetLink
	}
	if err := q.UpdateUserPassword(ctx, sqlcgen.UpdateUserPasswordParams{
		Passwordhash: string(hash), Updatedat: dbvalue.NowISO(), ID: link.Userid,
	}); err != nil {
		return fmt.Errorf("gravar a senha da conta %d: %w", link.Userid, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fechar a transação da redefinição: %w", err)
	}
	return nil
}

// usable carrega um link que ainda pode ser gasto: ele existe, ninguém o usou, e
// não venceu.
func (r Resets) usable(ctx context.Context, token string) (sqlcgen.PasswordReset, bool) {
	if token == "" {
		return sqlcgen.PasswordReset{}, false
	}
	link, err := r.gate.queries.GetPasswordReset(ctx, token)
	if err != nil || link.Usedat.Valid {
		return sqlcgen.PasswordReset{}, false
	}
	wins, err := time.Parse(dbvalue.IsoLayout, link.Expiresat)
	if err != nil || time.Now().UTC().After(wins) {
		return sqlcgen.PasswordReset{}, false
	}
	return link, true
}
