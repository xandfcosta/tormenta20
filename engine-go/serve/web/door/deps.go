package door

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/domain/account"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA da porta. `door` e `port` são a mesma palavra em português e vão
// confundir alguém: `door` é a CENA de entrar, `port` é a interface que uma cena
// declara. Ver GLOSSARY.md.
//
// Duas coisas NÃO atravessam esta fronteira:
//
// O USUÁRIO INTEIRO. `AuthUser` é tipo do `api`, e pedi-lo faria a cena importar
// o hospedeiro, que a importa de volta para montar rota: ciclo. A cena só
// precisa saber SE há sessão, então a porta pede `HasSession`. A regra vale para
// toda porta — uma que devolve tipo do hospedeiro não é porta, é o hospedeiro
// com outro nome.
//
// O `bcrypt`. A cena escrevia o hash da senha nova com o custo criptográfico do
// `api`, pedido por uma porta, para fazer trabalho que não é dela. Hoje o
// `ResetPassword` faz o caminho inteiro do outro lado.
type Deps interface {
	// Queries é o banco. A porta lê UMA coisa: o e-mail que o link de
	// redefinição aponta.
	Queries() *sqlcgen.Queries
	// WritePage é a montagem da casca (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
	HasSession(r *http.Request) bool
	// Authenticate confere e-mail e senha. O erro NÃO se distingue na tela —
	// conta inexistente e senha errada dão a mesma frase.
	Authenticate(ctx context.Context, email, password string) (sqlcgen.User, error)
	// CreateAccount fica no hospedeiro porque o registro é caminho compartilhado
	// com a API JSON: a porta é uma entrada dele, não a dona.
	CreateAccount(ctx context.Context, body account.RegisterBody) (sqlcgen.User, error)
	// IssueSession escreve o cookie. `false` é não ter conseguido assinar, e aí
	// a cena mostra a recusa em vez de mandar para dentro.
	IssueSession(w http.ResponseWriter, user sqlcgen.User) bool
	// ResetLinkOwner é o e-mail que o link aponta, e se ele ainda vale. Mostrar
	// o e-mail é a única coisa que esta rota anônima revela, e ela existe para
	// quem clicou saber que está mudando a conta certa.
	ResetLinkOwner(ctx context.Context, token string) (email string, ok bool)
	ResetPassword(ctx context.Context, token, password string) bool
	// SignUpRefusal devolve uma CHAVE e não a frase pronta: o hospedeiro
	// classifica o erro (os sentinelas são dele), a cena escolhe o texto.
	SignUpRefusal(err error) (motive RefusalMotive, status int)
}

// RefusalMotive é o vocabulário de recusa do registro, e ele é DA CENA.
type RefusalMotive string

const (
	RefusalEmailTaken RefusalMotive = "email-em-uso"
	RefusalBadInvite  RefusalMotive = "convite-invalido"
	RefusalInternal   RefusalMotive = "interno"
)

// Scene é a porta montada com as dependências dela.
//
// Uma struct e não a interface direta porque o Go não aceita interface como
// RECEPTOR, e os handlers precisam ser métodos para as rotas ficarem legíveis.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
