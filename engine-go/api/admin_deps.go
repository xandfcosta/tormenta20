package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"t20engine/plataforma"
	"time"

	"t20engine/db/sqlcgen"
)

// O QUE O HOSPEDEIRO DEVE À CENA DE ADMINISTRAÇÃO (ALE-278).
//
// A `admin.Deps` é declarada lá, no consumidor. O que mora aqui é o cumprimento
// dela, e ele é fino: cada método embrulha o que a casa já fazia, com o nome
// exportado que a interface pede e devolvendo a MENOR resposta que a tela usa.
//
// Quatro deles descartam o que a assinatura antiga devolvia — o caminho do
// backup, a contagem de campanhas transferidas, o status HTTP, a linha do banco
// das duas cunhagens. Cada um desses valores é um tipo a menos que a cena
// conhece, e o `backupDTO` em particular teria feito a tela depender da forma do
// JSON da API de backup.
//
// # O adaptador deixou de ser o `*Server` (ALE-278, fatia 6)
//
// Ele carrega o núcleo mais DUAS coisas: a configuração (é dela que saem o
// ambiente, o caminho do banco e a política de backup) e o `*sql.DB` (apagar
// conta é uma transação — as campanhas mudam de dono e a linha some juntas, ou
// uma mesa fica órfã de um usuário que não existe).
//
// As quatro regras que só esta cena usa — `deleteAccount`,
// `deleteUserKeepingCampaigns`, `backupDatabase` e `mintPasswordReset` — desceram
// para cá com ele. Elas nunca foram do servidor: estavam nele porque o handler
// HTTP que as chamava estava, e o `deleteAccount` em particular é a SEGUNDA
// regra deste repositório que apareceu soldada ao transporte que a alcançou
// primeiro.
type adminHost struct {
	sceneCore
	cfg plataforma.Config
	db  *sql.DB
}

func (s *Server) adminHost() adminHost {
	return adminHost{sceneCore: s.sceneCore(), cfg: s.cfg, db: s.db}
}

func (h adminHost) IsAdmin(email string) bool { return h.cfg.IsAdmin(email) }

func (h adminHost) Environment() string  { return string(h.cfg.AppEnv) }
func (h adminHost) DatabasePath() string { return h.cfg.DatabasePath }
func (h adminHost) DatabaseSize() int64  { return fileSize(h.cfg.DatabasePath) }

// LastBackup é o backup mais recente. `ok` em falso é "nenhum ainda", que é
// estado normal — a lista vazia não é erro.
func (h adminHost) LastBackup() (string, int64, bool) {
	lista := h.listBackups()
	if len(lista) == 0 {
		return "", 0, false
	}
	return lista[0].Name, lista[0].Size, true
}

func (h adminHost) BackupNow(ctx context.Context, at time.Time) error {
	_, err := h.backupDatabase(ctx, at)
	return err
}

func (h adminHost) DeleteAccount(r *http.Request, id, callerID int64) error {
	_, _, err := h.deleteAccount(r, id, callerID)
	return err
}

// MintAccountInvite é pedido por DUAS cenas — esta e o hub —, e as duas chamam
// a mesma função de pacote. Ele não subiu para o núcleo porque duas cenas em
// onze não é "toda cena": o núcleo é o que quase todo mundo pede, e uma linha
// repetida em dois adaptadores custa menos que uma assinatura no núcleo que
// nove cenas carregam sem usar.
func (h adminHost) MintAccountInvite(ctx context.Context, by int64) (sqlcgen.AccountInvite, error) {
	return mintAccountInvite(ctx, h.queries, by)
}

// MintPasswordReset é o par do `MintAccountInvite`. As duas devolvem a LINHA, e
// não o token, porque aquela já estava escrita assim — ver a razão na
// `admin.Deps`.
func (h adminHost) MintPasswordReset(ctx context.Context, userID, by int64) (sqlcgen.PasswordReset, error) {
	return h.mintPasswordReset(ctx, userID, by)
}

// IsUnknownUser é o sentinela desta casa visto de fora.
//
// A cena precisa separar "essa conta não existe mais" de "deu erro", porque as
// duas frases são diferentes na tela — e o valor que os distingue é daqui. Um
// predicado bastou: a porta precisou de um `RefusalMotive` porque tinha TRÊS
// casos, e aqui há um.
func (h adminHost) IsUnknownUser(err error) bool { return errors.Is(err, errUserNotFound) }
