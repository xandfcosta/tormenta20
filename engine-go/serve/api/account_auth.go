package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"t20engine/domain/account"
	"t20engine/infra/platform"
	"time"

	"t20engine/infra/db"
	"t20engine/infra/db/sqlcgen"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// AuthUser é o contrato de identidade devolvido ao cliente. `Name` é nulo quando
// não foi preenchido.
type AuthUser struct {
	ID    int64   `json:"id"`
	Email string  `json:"email"`
	Name  *string `json:"name"`
	// IsAdmin é DERIVADO do ADMIN_EMAILS a cada pedido, nunca guardado: o papel
	// não tem linha para envelhecer, e é isto que a tela lê para mostrar a porta
	// da administração.
	IsAdmin bool `json:"isAdmin"`
}

func (a accountRules) authUser(u sqlcgen.User) AuthUser {
	out := AuthUser{ID: u.ID, Email: u.Email, IsAdmin: a.cfg.IsAdmin(u.Email)}
	if u.Name.Valid {
		out.Name = &u.Name.String
	}
	return out
}

// A FORMA dos dois pedidos mora no `account`, junto com as validações que a
// leem. Aqui fica o handler.

// createAccount é a REGRA por trás do cadastro: resolve o convite que este
// endereço tem de gastar, gera o hash e escreve a linha.
//
// Ela não depende do transporte, e isso não é zelo: uma regra pregada ao
// primeiro transporte que a alcançou só aparece como problema quando chega o
// segundo — e aí ela é reescrita, em vez de reusada.
func (a accountRules) createAccount(ctx context.Context, body account.RegisterBody) (sqlcgen.User, error) {
	// A normalização é da REGRA, e não de quem a chama: uma garantia que mora no
	// transporte é uma garantia que o próximo transporte esquece. O `IsAdmin` do
	// `ADMIN_EMAILS` já compara normalizado, então um chamador que esquecesse a
	// linha escreveria `DONO@` como uma SEGUNDA linha em `users`, sem colidir com
	// `dono@` e com direito a dispensar convite: dois administradores onde só
	// cabe um. É idempotente para quem já normaliza.
	body.Email = platform.NormalizeEmail(body.Email)
	invite, err := a.registrationInvite(ctx, body.Email, body.InviteToken)
	if err != nil {
		return sqlcgen.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcryptCost)
	if err != nil {
		return sqlcgen.User{}, err
	}
	now := platform.NowISO()
	return a.createUser(ctx, sqlcgen.CreateUserParams{
		Email:        body.Email,
		Name:         platform.NullString(body.Name),
		Passwordhash: string(hash),
		Createdat:    now,
		Updatedat:    now,
	}, invite)
}

// errInviteRejected cobre desconhecido, gasto e expirado do mesmo jeito — ver o
// `inviteRejected`.
var errInviteRejected = errors.New(inviteRejected)

// registrationInvite resolve o convite que este cadastro tem de gastar. Os
// endereços do ADMIN_EMAILS são a exceção, e a única: o dono precisa conseguir
// criar a própria conta numa máquina nova, e "quem se cadastra primeiro ganha a
// coroa" entregaria isso a quem abrisse a página antes.
func (a accountRules) registrationInvite(
	ctx context.Context, email, token string,
) (*sqlcgen.AccountInvite, error) {
	if a.cfg.IsAdmin(email) {
		return nil, nil
	}
	invite, ok := a.usableInvite(ctx, token)
	if !ok {
		return nil, errInviteRejected
	}
	return &invite, nil
}

func writeRegisterError(w http.ResponseWriter, err error, email string) {
	switch {
	case db.IsUniqueViolation(err):
		platform.WriteError(w, http.StatusConflict, "Email already registered: "+email)
	case errors.Is(err, errInviteRejected), errors.Is(err, errInviteSpent):
		platform.WriteError(w, http.StatusForbidden, inviteRejected)
	default:
		platform.WriteError(w, http.StatusInternalServerError, "Could not create user")
	}
}

// errBadCredentials é a ÚNICA resposta para "não existe essa conta" e "senha
// errada": distinguir as duas entrega a um chamador anônimo um jeito de enumerar
// quem tem conta aqui.
var errBadCredentials = errors.New("invalid credentials")

// authenticate é a REGRA por trás do login, fora do transporte pela mesma razão
// do `createAccount`.
//
// O que importa aqui é que os DOIS caminhos respondem o mesmo erro. Rodar a
// comparação do bcrypt mesmo com e-mail desconhecido seria o próximo degrau: hoje
// ela não roda, e isso é um oráculo de tempo.
func (a accountRules) authenticate(ctx context.Context, email, password string) (sqlcgen.User, error) {
	user, err := a.queries.GetUserByEmail(ctx, platform.NormalizeEmail(email))
	if err != nil {
		return sqlcgen.User{}, errBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Passwordhash), []byte(password)) != nil {
		return sqlcgen.User{}, errBadCredentials
	}
	return user, nil
}

// issueSession assina um JWT para o usuário e põe o biscoito de sessão. Devolve
// falso (depois de escrever um 500) quando a assinatura falha.
func (a accountRules) issueSession(w http.ResponseWriter, user sqlcgen.User) bool {
	token, err := a.signToken(user)
	if err != nil {
		platform.WriteError(w, http.StatusInternalServerError, "Could not sign session")
		return false
	}
	http.SetCookie(w, sessionCookie(a.cfg, token, int(sessionTTL.Seconds())))
	return true
}

const sessionTTL = 7 * 24 * time.Hour

// signToken assina HS256 sobre `{sub, email}` com a expiração configurada. O
// `sub` é um NÚMERO e não uma string, que é a forma que o `verifyToken` espera.
func (a accountRules) signToken(user sqlcgen.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(parseExpiry(a.cfg.JWTExpiresIn)).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(a.cfg.JWTSecret))
}

// verifyToken confere a assinatura HS256 e a expiração, e devolve o id do
// usuário (`sub`).
func (a accountRules) verifyToken(tokenStr string) (int64, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.cfg.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return 0, errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid claims")
	}
	sub, ok := claims["sub"].(float64) // JSON numbers decode as float64
	if !ok {
		return 0, errors.New("missing sub")
	}
	return int64(sub), nil
}

// Ela recebe a CONFIGURAÇÃO em vez de pendurar no `*Server`: uma função que só
// precisa de dois campos não tem razão para exigir um servidor inteiro.
func sessionCookie(cfg platform.Config, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     cfg.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

// parseExpiry lê as formas de JWT_EXPIRES_IN que esta configuração aceita ("7d",
// "12h", "30m"). Cai em 7 dias no que não reconhecer.
func parseExpiry(s string) time.Duration {
	if s == "" {
		return sessionTTL
	}
	unit := s[len(s)-1]
	n, err := platform.ParseInt(s[:len(s)-1])
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

func extractBearer(h string) string {
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
