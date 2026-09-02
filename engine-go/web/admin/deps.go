package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/a-h/templ"

	"t20engine/db/sqlcgen"
	"t20engine/web/ui"
)

// A PORTA DA ADMINISTRAÇÃO (ALE-278).
//
// Ela é a maior das quatro escritas até aqui — treze métodos contra nove da
// porta, seis da forja e ZERO do buscador —, e o tamanho não é vício: esta cena
// é um painel de controle sobre serviços do servidor. Fazer backup, cunhar
// convite, apagar conta e ler o tamanho do banco são coisas que o hospedeiro
// faz; a cena só oferece o botão e desenha o resultado.
//
// O sinal de que a fronteira está no lugar certo é que NENHUM destes métodos
// desenha nada, e nenhum handler daqui toca banco fora do `Queries`.
//
// # Cada um pede a MENOR pergunta que resolve
//
// É a regra que a forja abriu e a porta confirmou, e aqui ela apertou quatro
// assinaturas de uma vez. Todas as quatro devolviam mais do que a tela usa:
//
//   - `backupDatabase` devolvia o caminho do arquivo, e a cena descarta;
//   - `deleteAccount` devolvia quantas campanhas passaram e um status HTTP, e a
//     cena descarta os dois;
//   - `listBackups` devolvia a lista inteira de `backupDTO` — um tipo do
//     hospedeiro — e a cena lê o nome e o tamanho do PRIMEIRO.
//
// Não é economia de bytes: cada tipo que atravessa é um tipo que a cena passa a
// conhecer, e o `backupDTO` teria feito esta cena depender da forma do JSON da
// API de backup — que não tem nada a ver com ela.
//
// As duas CUNHAGENS são a exceção, e a razão está na linha delas: ali um
// contrato que já existia ganhou da regra.
type Deps interface {
	// Queries é o banco. A tela faz três leituras: os jogadores com as
	// contagens, os convites abertos, e o total de linhas de cada tabela.
	Queries() *sqlcgen.Queries
	// WritePage é a montagem da casca (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
	// CurrentUserID é quem está pedindo. Como na forja: o ID, e não o usuário,
	// porque o tipo do usuário é do hospedeiro.
	//
	// A cena o usa para UMA coisa — não oferecer o botão de apagar na própria
	// linha. A trava de verdade é do servidor; esta só evita oferecer um botão
	// que responderia erro.
	CurrentUserID(r *http.Request) int64

	// IsAdmin diz se um e-mail está no `ADMIN_EMAILS`. A cena mostra isso na
	// frase de posses da linha; ela não decide nada com isso.
	IsAdmin(email string) bool
	// Environment, DatabasePath e DatabaseSize são o painel de servidor.
	//
	// Os três vêm da configuração, que é do hospedeiro. Pedir a `Config` inteira
	// faria a cena conhecer trinta campos para mostrar dois — e o tipo é dele.
	Environment() string
	DatabasePath() string
	DatabaseSize() int64
	// LastBackup é o nome e o tamanho do backup mais recente, se houver algum.
	//
	// Ver acima: a lista inteira de `backupDTO` não atravessa. `ok` em falso é
	// "nenhum backup ainda", que é um estado normal e não um erro.
	LastBackup() (name string, size int64, ok bool)

	// BackupNow grava um backup agora. O caminho do arquivo fica do lado de lá.
	BackupNow(ctx context.Context, at time.Time) error
	// DeleteAccount apaga a conta e transfere as campanhas dela para quem
	// apagou. A REGRA é a mesma do handler JSON, e é por isso que ela mora no
	// hospedeiro: duas versões de "não se apaga a própria conta" divergiriam.
	DeleteAccount(r *http.Request, id, callerID int64) error
	// MintAccountInvite e MintPasswordReset cunham os dois links. O prazo de
	// cada um é regra do hospedeiro; a tela lê só o `.Token`.
	//
	// ELAS DEVOLVEM A LINHA e não o token, e essa é a única das quatro
	// assinaturas em que a regra da "menor pergunta" cedeu. O motivo é um
	// contrato que já existia: o `hub.Deps` pede `MintAccountInvite` com esta
	// forma desde a extração dele, e o `*Server` a cumpre. Encolher a resposta
	// aqui obrigaria o hospedeiro a ter DOIS métodos com o mesmo nome e formas
	// diferentes — o compilador recusa —, ou um segundo nome para a mesma coisa,
	// que é pior que a cena ler um campo.
	MintAccountInvite(ctx context.Context, by int64) (sqlcgen.AccountInvite, error)
	MintPasswordReset(ctx context.Context, userID, by int64) (sqlcgen.PasswordReset, error)
	// IsUnknownUser separa "essa conta não existe mais" de "deu erro".
	//
	// O sentinela é um valor do hospedeiro e a cena não pode alcançá-lo. Aqui
	// bastou um predicado em vez do `RefusalMotive` da porta, e a diferença é
	// que lá havia TRÊS casos a distinguir e aqui há um — a menor coisa que
	// resolve, outra vez.
	IsUnknownUser(err error) bool
}

// Scene é a administração montada com as dependências dela.
type Scene struct{ deps Deps }

// New monta a cena. Quem chama é quem tem as dependências na mão: hoje o `api`.
func New(d Deps) Scene { return Scene{deps: d} }
