package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"t20engine/app"
	"t20engine/domain/account"
	"t20engine/infra/config"
	"t20engine/infra/db"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/secret"
	"t20engine/infra/wire"
)

// AS RECUSAS DA PORTA, e elas são EXPORTADAS.
//
// Era o contrário: os sentinelas moravam no `serve/api` e a cena não podia
// lê-los — a porta dela dizia, por escrito, que isso seria "alcance que a
// divisão existe para cortar" —, então o adaptador os traduzia num vocabulário
// que a cena declarava só para atravessar a fronteira. Com as recusas aqui, no
// `app/`, quem está ACIMA pode lê-las e o tipo do meio deixa de existir
// (ALE-349, e é a mesma coisa que a ALE-348 fez com a recusa de sentar à mesa).
var (
	// ErrBadCredentials é a ÚNICA resposta para "não existe essa conta" e "senha
	// errada": distinguir as duas entrega a um chamador anônimo um jeito de
	// enumerar quem tem conta aqui.
	ErrBadCredentials = fmt.Errorf("e-mail ou senha não conferem: %w", app.ErrForbidden)
	// ErrEmailTaken é o endereço que já tem conta.
	ErrEmailTaken = fmt.Errorf("este e-mail já tem conta: %w", app.ErrRefused)
	// ErrBadInvite cobre desconhecido, vencido e GASTO do mesmo jeito — inclusive
	// o gasto que só a transação descobre, quando duas pessoas abrem o mesmo
	// link ao mesmo tempo. As três pedem a mesma coisa de quem está do outro
	// lado: um convite novo.
	ErrBadInvite = fmt.Errorf("este convite não serve: %w", app.ErrForbidden)
)

// accountInviteTTL é curto de propósito: o link passa de mão em mão na mesa e
// não por e-mail, então uma semana é generosa — e um link esquecido num
// histórico de conversa para de funcionar.
const accountInviteTTL = 7 * 24 * time.Hour

// bcryptCost é o custo do hash de senha.
//
// Doze é a decisão de segurança do servidor, e ela mora com a regra e não com o
// formulário — ver o `doc.go` do pacote.
const bcryptCost = 12

const sessionTTL = 7 * 24 * time.Hour

// Gate é a PORTA: entrar, cadastrar-se, e a sessão que sai disso.
type Gate struct {
	db      *sql.DB
	queries *sqlcgen.Queries
	cfg     config.Config
}

func NewGate(db *sql.DB, q *sqlcgen.Queries, cfg config.Config) Gate {
	return Gate{db: db, queries: q, cfg: cfg}
}

// SessionTTL é quanto dura a sessão, para quem monta o biscoito.
//
// O biscoito é do TRANSPORTE — quem sabe o que é `Secure` e `SameSite` é quem
// responde HTTP —, mas quanto ele dura é a mesma decisão que assina o token.
// Dois números separados divergiriam, e o sintoma seria uma sessão que o
// navegador esquece antes de o servidor recusar.
func (g Gate) SessionTTL() time.Duration { return sessionTTL }

// Authenticate é a REGRA por trás do login.
//
// Os DOIS caminhos respondem o mesmo erro. Rodar a comparação do bcrypt mesmo
// com e-mail desconhecido seria o próximo degrau: hoje ela não roda, e isso é
// um oráculo de tempo.
func (g Gate) Authenticate(ctx context.Context, email, password string) (sqlcgen.User, error) {
	user, err := g.queries.GetUserByEmail(ctx, wire.NormalizeEmail(email))
	if err != nil {
		return sqlcgen.User{}, ErrBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Passwordhash), []byte(password)) != nil {
		return sqlcgen.User{}, ErrBadCredentials
	}
	return user, nil
}

// Register é a REGRA por trás do cadastro: resolve o convite que este endereço
// tem de gastar, gera o hash e escreve a linha.
func (g Gate) Register(ctx context.Context, body account.RegisterBody) (sqlcgen.User, error) {
	// A NORMALIZAÇÃO é da regra, e não de quem a chama: uma garantia que mora no
	// transporte é uma garantia que o próximo transporte esquece. O `IsAdmin` do
	// `ADMIN_EMAILS` já compara normalizado, então um chamador que esquecesse a
	// linha escreveria `DONO@` como uma SEGUNDA linha em `users`, sem colidir
	// com `dono@` e com direito a dispensar convite: dois administradores onde
	// só cabe um. É idempotente para quem já normaliza.
	body.Email = wire.NormalizeEmail(body.Email)
	invite, err := g.registrationInvite(ctx, body.Email, body.InviteToken)
	if err != nil {
		return sqlcgen.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcryptCost)
	if err != nil {
		return sqlcgen.User{}, fmt.Errorf("gerar o hash da senha: %w", err)
	}
	now := dbvalue.NowISO()
	return g.createUser(ctx, sqlcgen.CreateUserParams{
		Email:        body.Email,
		Name:         dbvalue.NullString(body.Name),
		Passwordhash: string(hash),
		Createdat:    now,
		Updatedat:    now,
	}, invite)
}

// MintInvite cunha o link de uso único que abre UMA conta.
//
// Ele é pedido por DUAS telas — a administração e o hub —, e é por isso que o
// prazo mora aqui: com a conta de validade dentro de um manipulador HTTP, a
// segunda tela só teria duas saídas, chamar a própria rota por dentro ou copiar
// a conta.
func (g Gate) MintInvite(ctx context.Context, createdBy int64) (sqlcgen.AccountInvite, error) {
	token, err := secret.Token()
	if err != nil {
		return sqlcgen.AccountInvite{}, err
	}
	now := time.Now()
	return g.queries.CreateAccountInvite(ctx, sqlcgen.CreateAccountInviteParams{
		Token:     token,
		Createdby: createdBy,
		Createdat: dbvalue.IsoAt(now),
		Expiresat: dbvalue.IsoAt(now.Add(accountInviteTTL)),
	})
}

// registrationInvite resolve o convite que este cadastro tem de gastar.
//
// Os endereços do `ADMIN_EMAILS` são a exceção, e a única: o dono precisa
// conseguir criar a própria conta numa máquina nova, e "quem se cadastra
// primeiro ganha a coroa" entregaria isso a quem abrisse a página antes.
func (g Gate) registrationInvite(
	ctx context.Context, email, token string,
) (*sqlcgen.AccountInvite, error) {
	if g.cfg.IsAdmin(email) {
		return nil, nil
	}
	invite, ok := g.usableInvite(ctx, token)
	if !ok {
		return nil, ErrBadInvite
	}
	return &invite, nil
}

// usableInvite carrega um convite que ainda pode ser gasto: ele existe, ninguém
// o usou, e não venceu.
func (g Gate) usableInvite(ctx context.Context, token string) (sqlcgen.AccountInvite, bool) {
	if token == "" {
		return sqlcgen.AccountInvite{}, false
	}
	invite, err := g.queries.GetAccountInvite(ctx, token)
	if err != nil || invite.Usedat.Valid {
		return sqlcgen.AccountInvite{}, false
	}
	wins, err := time.Parse(dbvalue.IsoLayout, invite.Expiresat)
	if err != nil || time.Now().UTC().After(wins) {
		return sqlcgen.AccountInvite{}, false
	}
	return invite, true
}

// createUser insere a conta e, quando o cadastro veio de um convite, o GASTA na
// MESMA transação.
//
// É o que faz o uso único ser invariante em vez de esperança: dois jogadores
// abrindo o mesmo link ao mesmo tempo passam os dois pela conferência de
// leitura, mas só um `UPDATE` acha `usedAt IS NULL`, e a conta do perdedor volta
// atrás junto.
//
// `invite` é nulo quando um endereço de `ADMIN_EMAILS` cria a própria conta.
func (g Gate) createUser(
	ctx context.Context, params sqlcgen.CreateUserParams, invite *sqlcgen.AccountInvite,
) (sqlcgen.User, error) {
	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return sqlcgen.User{}, fmt.Errorf("abrir a transação do cadastro: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := g.queries.WithTx(tx)
	user, err := q.CreateUser(ctx, params)
	if db.IsUniqueViolation(err) {
		return sqlcgen.User{}, ErrEmailTaken
	}
	if err != nil {
		return sqlcgen.User{}, fmt.Errorf("gravar a conta de %q: %w", params.Email, err)
	}
	if invite != nil {
		if err := spend(ctx, q, invite.ID, user.ID); err != nil {
			return sqlcgen.User{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return sqlcgen.User{}, fmt.Errorf("fechar a transação do cadastro: %w", err)
	}
	return user, nil
}

// spend marca o convite como usado, e ZERO linhas afetadas quer dizer que
// alguém ganhou a corrida entre a conferência e o `UPDATE`.
func spend(ctx context.Context, q *sqlcgen.Queries, inviteID, userID int64) error {
	rows, err := q.SpendAccountInvite(ctx, sqlcgen.SpendAccountInviteParams{
		Usedat: sql.NullString{String: dbvalue.NowISO(), Valid: true},
		Usedby: sql.NullInt64{Int64: userID, Valid: true},
		ID:     inviteID,
	})
	if err != nil {
		return fmt.Errorf("gastar o convite %d: %w", inviteID, err)
	}
	if rows == 0 {
		return ErrBadInvite
	}
	return nil
}

// SignSession assina o JWT de uma sessão.
//
// Ela devolve o TOKEN e não escreve biscoito nenhum: pôr o cookie é do
// transporte, e a versão que morava no `serve/api` recebia um
// `http.ResponseWriter` e chegava a escrever um 500 lá dentro.
//
// O `sub` é um NÚMERO e não uma string, que é a forma que o `UserOfSession`
// espera.
func (g Gate) SignSession(user sqlcgen.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(parseExpiry(g.cfg.JWTExpiresIn)).Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(g.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("assinar a sessão de %d: %w", user.ID, err)
	}
	return signed, nil
}

// UserOfSession confere a assinatura HS256 e a expiração, e devolve o id do
// usuário (`sub`).
func (g Gate) UserOfSession(token string) (int64, error) {
	tok, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de assinatura inesperado")
		}
		return []byte(g.cfg.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return 0, fmt.Errorf("sessão inválida: %w", app.ErrForbidden)
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("sessão sem reivindicações: %w", app.ErrForbidden)
	}
	// O JSON decodifica número como `float64`.
	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, fmt.Errorf("sessão sem dono: %w", app.ErrForbidden)
	}
	return int64(sub), nil
}

// parseExpiry lê as formas de `JWT_EXPIRES_IN` que esta configuração aceita
// ("7d", "12h", "30m"). Cai em sete dias no que não reconhecer.
func parseExpiry(s string) time.Duration {
	if s == "" {
		return sessionTTL
	}
	unit := s[len(s)-1]
	n, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		return sessionTTL
	}
	switch unit {
	case 'd':
		return time.Duration(n) * 24 * time.Hour
	case 'h':
		return time.Duration(n) * time.Hour
	case 'm':
		return time.Duration(n) * time.Minute
	default:
		return sessionTTL
	}
}
