package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/a-h/templ"

	"t20engine/app/accounts"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA DA ADMINISTRAÇÃO. Ela foi a mais larga de todas, e o que sobrou é o
// que de fato é do HOSPEDEIRO: o banco, a casca, quem pede, e os serviços do
// servidor — ambiente, caminho e tamanho do arquivo, e o backup.
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

	// BackupNow fica na PORTA, e é deliberado: copiar o arquivo do banco é
	// serviço do hospedeiro — sistema de arquivos e política de retenção —, e
	// não caso de uso de conta nenhuma. A ALE-349 mudou de lado o que era regra
	// de CONTA, e só isso.
	BackupNow(ctx context.Context, at time.Time) error
}

// Scene é a administração montada com as dependências dela.
//
// As quatro entradas que saíram da porta — apagar conta, cunhar convite, cunhar
// link de senha, e o predicado que distinguia "conta inexistente" — chegam por
// PARÂMETRO, do `app/accounts` (ALE-349). O predicado não veio junto porque
// deixou de fazer sentido: `accounts.ErrUnknownAccount` é valor exportado, e a
// cena o lê com `errors.Is`.
type Scene struct {
	deps   Deps
	gate   accounts.Gate
	resets accounts.Resets
	roster accounts.Roster
}

func New(d Deps, portao accounts.Gate, redefinicoes accounts.Resets, elenco accounts.Roster) Scene {
	return Scene{deps: d, gate: portao, resets: redefinicoes, roster: elenco}
}
