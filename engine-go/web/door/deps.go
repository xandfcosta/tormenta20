package door

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/account"
	"t20engine/db/sqlcgen"
	"t20engine/web/ui"
)

// A PORTA da porta — e a piada não é de graça, é o aviso da colisão (ALE-278).
//
// `door` e `port` são a mesma palavra em português, e o GLOSSARIO registra a
// distinção porque ela vai confundir alguém: `door` é a CENA de entrar, `port` é
// a interface que uma cena declara para o hospedeiro. Este arquivo é a `port` da
// `door`.
//
// # O que ela pede
//
// Medido antes de escrever: a cena tocava OITO coisas do `Server`. Cinco são
// mecânica de conta que não é da tela — autenticar, criar, abrir sessão, e as
// duas pontas do link de redefinição —, uma é a casca, uma é o banco, e a última
// era a que precisava mudar de forma.
//
// # A que mudou de forma, e por quê
//
// A cena chamava `s.sessionUser(r)`, que devolve `AuthUser` — tipo do `api`.
// Pedir isso aqui faria a cena IMPORTAR o hospedeiro, que a importa de volta
// para montar rota: ciclo. E a cena não precisa do usuário: o único uso era
// perguntar SE havia sessão, para mandar quem já entrou para longe da tela de
// login. Então a porta pede `HasSession`, que é a pergunta de verdade.
//
// É a mesma decisão que a forja tomou ao pedir `CurrentUserID(r) int64` em vez
// do usuário inteiro, e a regra que as duas seguem é uma: **uma porta que
// devolve tipo do hospedeiro não é porta, é o hospedeiro com outro nome.**
//
// # E a que NÃO está aqui
//
// O `bcrypt` não atravessa. A cena escrevia o hash da senha nova ela mesma,
// usando o `bcryptCost` do `api` — o que a obrigaria a pedir uma constante de
// custo criptográfico por uma porta, para fazer trabalho que não é dela.
// `ResetPassword` faz o caminho inteiro do outro lado: confere o link, gera o
// hash e grava. A cena pergunta se deu certo.
type Deps interface {
	// Queries é o banco. A porta lê UMA coisa: o e-mail da conta que o link de
	// redefinição aponta, para a tela dizer qual conta está sendo mudada.
	Queries() *sqlcgen.Queries
	// WritePage é a montagem da casca: ela injeta os estáticos e as
	// sobreposições que a cena não pode conhecer (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
	// HasSession responde se o pedido já traz uma sessão válida — e SÓ isso.
	// Ver a explicação acima: o usuário inteiro é tipo do hospedeiro.
	HasSession(r *http.Request) bool
	// Authenticate confere e-mail e senha. O erro não se distingue na tela: a
	// porta responde a mesma frase para conta inexistente e senha errada, e
	// quem decide isso é a cena, não esta interface.
	Authenticate(ctx context.Context, email, password string) (sqlcgen.User, error)
	// CreateAccount faz nascer a conta, com o convite de uso único. Ela fica no
	// hospedeiro porque o registro é caminho compartilhado com a API JSON — a
	// porta é uma entrada dele, não a dona.
	CreateAccount(ctx context.Context, body account.RegisterBody) (sqlcgen.User, error)
	// IssueSession escreve o cookie. Devolve `false` quando não conseguiu
	// assinar, e aí a cena mostra a recusa em vez de mandar para dentro.
	IssueSession(w http.ResponseWriter, user sqlcgen.User) bool
	// ResetLinkOwner é o e-mail da conta que o link aponta, e se ele ainda vale.
	// A cena mostra o e-mail para quem clicou saber que está mudando a conta
	// certa — é a única coisa que esta rota anônima revela.
	ResetLinkOwner(ctx context.Context, token string) (email string, ok bool)
	// ResetPassword confere o link, gera o hash e grava, num caminho só. Ver
	// acima por que o `bcrypt` não atravessa esta fronteira.
	ResetPassword(ctx context.Context, token, password string) bool
	// SignUpRefusal traduz o erro de `CreateAccount` no que a tela mostra: a
	// frase e o status. Os sentinelas que ela distingue — e-mail em uso,
	// convite recusado, convite gasto — são valores do hospedeiro, e a cena não
	// pode alcançá-los sem importá-lo.
	//
	// Ela devolve uma CHAVE e não a frase pronta, porque o texto que o jogador
	// lê é da tela: é aqui que a voz da porta mora, e ela não vai para o `api`
	// junto com a mecânica.
	SignUpRefusal(err error) (motive RefusalMotive, status int)
}

// RefusalMotive é o vocabulário de recusa do registro, e ele é DA CENA.
//
// O hospedeiro classifica o erro; a cena escolhe a frase. Sem isso, ou a cena
// importa os sentinelas do `api` (que é o ciclo), ou o `api` passa a escrever
// texto de tela (que é a voz da porta morando fora dela).
type RefusalMotive string

const (
	RefusalEmailTaken RefusalMotive = "email-em-uso"
	RefusalBadInvite  RefusalMotive = "convite-invalido"
	RefusalInternal   RefusalMotive = "interno"
)

// Scene é a porta montada com as dependências dela.
//
// Uma struct e não a interface direta porque o Go não aceita interface como
// RECEPTOR — e um invólucro de um campo é o preço de os handlers continuarem
// métodos, que é o que mantém as rotas legíveis.
type Scene struct{ deps Deps }

// New monta a cena. Quem chama é quem tem as dependências na mão: hoje o `api`.
func New(d Deps) Scene { return Scene{deps: d} }
