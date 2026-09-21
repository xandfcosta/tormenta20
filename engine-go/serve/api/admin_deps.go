package api

import (
	"context"
	"database/sql"
	"t20engine/infra/config"
	"time"
)

// O QUE O HOSPEDEIRO DEVE À CENA DE ADMINISTRAÇÃO.
//
// A `admin.Deps` é declarada lá, no consumidor. O que mora aqui é o cumprimento
// dela, e ele é fino: cada método embrulha o que a casa já fazia, com o nome
// exportado que a interface pede e devolvendo a MENOR resposta que a tela usa.
//
// O `BackupNow` descarta o caminho que a assinatura antiga devolvia, e é o
// último exemplo vivo da regra: cada valor a menos que atravessa é um tipo a
// menos que a cena conhece — o `backupDTO` teria feito a tela depender da forma
// do JSON da API de backup.
//
// # O adaptador não é o `*Server`
//
// Ele carrega o núcleo mais DUAS coisas: a configuração (é dela que saem o
// ambiente, o caminho do banco e a política de backup) e o `*sql.DB`. O banco
// continua aqui, mas por outra razão: era a TRANSAÇÃO de apagar conta que o
// justificava, e ela virou `accounts.Roster` na ALE-349 — o que resta é o
// `VACUUM INTO` do backup, que precisa da conexão e não de transação nenhuma.
//
// E o `backupDatabase` é a única regra que sobrou deste lado, de propósito:
// copiar arquivo é serviço do hospedeiro, não caso de uso de conta.
type adminHost struct {
	sceneCore
	cfg config.Config
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
	list := h.listBackups()
	if len(list) == 0 {
		return "", 0, false
	}
	return list[0].Name, list[0].Size, true
}

func (h adminHost) BackupNow(ctx context.Context, at time.Time) error {
	_, err := h.backupDatabase(ctx, at)
	return err
}
