package hub

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/db/sqlcgen"
	"t20engine/web/ui"
)

// A PORTA do hub (ALE-278, segunda cena).
//
// O hub é o menu principal: quem eu sou, que campanha está viva, e as portas
// para as outras cenas. A porta dele é pequena — um campo do banco e três
// métodos —, e o formato é o que a forja estabeleceu: a interface é declarada
// AQUI, no consumidor, e o `*api.Server` a cumpre.

// Viewer é quem está olhando, na língua DESTA cena.
//
// O `api` tem um `AuthUser` com os mesmos quatro campos, e a cena não o usa de
// propósito: pedir o tipo do hospedeiro faria este pacote importar o `api`, que
// o importa de volta para montar rota. Um struct de quatro campos é o preço de a
// fronteira existir — e ele documenta, por ser pequeno, exatamente o que a cena
// precisa saber sobre quem entrou.
type Viewer struct {
	ID    int64
	Email string
	Name  *string
	// IsAdmin decide se a porta da administração aparece. Ele é derivado do
	// `ADMIN_EMAILS` a cada requisição e nunca guardado — o papel não tem linha
	// para envelhecer contra (ALE-120).
	IsAdmin bool
}

// Deps é o que o hub pede de quem o hospeda.
type Deps interface {
	// Queries é o banco: o hub lê as campanhas e a sessão viva.
	Queries() *sqlcgen.Queries
	// CurrentViewer é quem está pedindo, já traduzido para a língua desta cena.
	CurrentViewer(r *http.Request) Viewer
	// MintAccountInvite cunha o convite que o menu oferece — criar conta para
	// outra pessoa é um ato da CASA, não desta tela.
	MintAccountInvite(ctx context.Context, byUserID int64) (sqlcgen.AccountInvite, error)
	// ExpiredSessionCookie é o cookie que apaga a sessão no logout. O formato
	// dele depende da configuração (domínio, `Secure`, `SameSite`), e essa
	// decisão é do hospedeiro.
	ExpiredSessionCookie() *http.Cookie
	// TableRoute é o endereço de uma sessão ao vivo.
	//
	// Ela entra pela porta e não é copiada, e a razão está escrita na função
	// original: ela existe para os quatro lugares que apontam para a Mesa
	// CONCORDAREM, e uma cópia aqui seria o quinto que diverge. Quem sabe onde
	// cada cena está montada é o hospedeiro.
	TableRoute(campaignID, sessionID int64) string
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// Scene é o hub montado com as dependências dele.
type Scene struct{ deps Deps }

// New monta a cena.
func New(d Deps) Scene { return Scene{deps: d} }
