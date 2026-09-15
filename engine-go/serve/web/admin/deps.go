package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/a-h/templ"

	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA DA ADMINISTRAÇÃO, a mais larga de todas — a cena é um painel de
// controle sobre serviços do servidor, e fazer backup, cunhar convite, apagar
// conta e medir o banco são coisas do hospedeiro.
//
// Cada método pede a MENOR pergunta que resolve, e isso não é economia de
// bytes: cada tipo que atravessa é um tipo que a cena passa a conhecer. O
// `backupDTO`, por exemplo, teria feito esta tela depender da forma do JSON da
// API de backup, que não tem nada a ver com ela.
type Deps interface {
	// Queries é o banco: os jogadores com as contagens, os convites abertos, e
	// o total de linhas de cada tabela.
	Queries() *sqlcgen.Queries
	// WritePage é a montagem da casca (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
	// CurrentUserID serve para não oferecer o botão de apagar na própria linha.
	// A trava de verdade é do servidor.
	CurrentUserID(r *http.Request) int64

	// IsAdmin diz se um e-mail está no `ADMIN_EMAILS`. A cena mostra, não decide.
	IsAdmin(email string) bool
	// Os três do painel de servidor vêm da configuração, que é do hospedeiro.
	// Pedir a `Config` inteira faria a cena conhecer trinta campos para mostrar
	// dois.
	Environment() string
	DatabasePath() string
	DatabaseSize() int64
	// LastBackup: `ok` falso é "nenhum backup ainda", que é estado normal.
	LastBackup() (name string, size int64, ok bool)

	BackupNow(ctx context.Context, at time.Time) error
	// DeleteAccount apaga a conta e transfere as campanhas dela. A regra mora no
	// hospedeiro porque é a mesma do handler JSON, e duas versões de "não se
	// apaga a própria conta" divergiriam.
	DeleteAccount(r *http.Request, id, callerID int64) error
	// As duas cunhagens devolvem a LINHA e não o token, que é a única vez em que
	// a regra da menor pergunta cede: o `hub.Deps` já pede `MintAccountInvite`
	// com esta forma, e encolher aqui obrigaria o hospedeiro a ter dois métodos
	// de mesmo nome — o compilador recusa.
	MintAccountInvite(ctx context.Context, by int64) (sqlcgen.AccountInvite, error)
	MintPasswordReset(ctx context.Context, userID, by int64) (sqlcgen.PasswordReset, error)
	// IsUnknownUser separa "essa conta não existe mais" de "deu erro": o
	// sentinela é valor do hospedeiro e a cena não o alcança.
	IsUnknownUser(err error) bool
}

// Scene é a administração montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
