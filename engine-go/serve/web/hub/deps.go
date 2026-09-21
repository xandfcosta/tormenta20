package hub

import (
	"net/http"

	"github.com/a-h/templ"

	"t20engine/app/accounts"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA do hub, que é o menu principal: quem eu sou, que campanha está viva,
// e os caminhos para as outras cenas.

// Viewer é quem está olhando, na língua DESTA cena. O `api` tem um `AuthUser`
// com os mesmos quatro campos, e a tradução é o preço de a fronteira existir.
type Viewer struct {
	ID    int64
	Email string
	Name  *string
	// IsAdmin decide se a porta da administração aparece. É derivado do
	// `ADMIN_EMAILS` a cada requisição e nunca guardado: o papel não tem linha
	// para envelhecer contra.
	IsAdmin bool
}

// Deps é o que o hub pede de quem o hospeda.
type Deps interface {
	// Queries é o banco: o hub lê as campanhas e a sessão viva.
	Queries() *sqlcgen.Queries
	// CurrentViewer é quem está pedindo, já traduzido para a língua desta cena.
	CurrentViewer(r *http.Request) Viewer
	// ExpiredSessionCookie apaga a sessão no logout. O formato depende da
	// configuração (domínio, `Secure`, `SameSite`), que é do hospedeiro.
	ExpiredSessionCookie() *http.Cookie
	// TableRoute entra pela porta e não é copiada: ela existe para os quatro
	// lugares que apontam para a Mesa CONCORDAREM, e uma cópia aqui seria o
	// quinto que diverge.
	TableRoute(campaignID, sessionID int64) string
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component)
}

// Scene é o hub montado com as dependências dele.
// Scene é o hub montado com as dependências dele.
//
// O `gate` chega por PARÂMETRO e não pela porta: cunhar o convite de conta é
// caso de uso, e o `app/` está abaixo desta cena (ALE-349). Criar conta para
// outra pessoa continua sendo ato da CASA e não desta tela — o que mudou é onde
// a casa guarda a regra.
type Scene struct {
	deps Deps
	gate accounts.Gate
}

func New(d Deps, gate accounts.Gate) Scene { return Scene{deps: d, gate: gate} }
