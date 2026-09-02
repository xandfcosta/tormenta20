package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"t20engine/db/sqlcgen"
)

// O QUE O HOSPEDEIRO DEVE À CENA DE ADMINISTRAÇÃO (ALE-278).
//
// A `admin.Deps` é declarada lá, no consumidor. O que mora aqui é o cumprimento
// dela, e ele é fino: cada método embrulha o que o `Server` já fazia, com o nome
// exportado que a interface pede e devolvendo a MENOR resposta que a tela usa.
//
// Quatro deles descartam o que a assinatura antiga devolvia — o caminho do
// backup, a contagem de campanhas transferidas, o status HTTP, a linha do banco
// das duas cunhagens. Cada um desses valores é um tipo a menos que a cena
// conhece, e o `backupDTO` em particular teria feito a tela depender da forma do
// JSON da API de backup.

func (s *Server) IsAdmin(email string) bool { return s.cfg.IsAdmin(email) }

func (s *Server) Environment() string  { return string(s.cfg.AppEnv) }
func (s *Server) DatabasePath() string { return s.cfg.DatabasePath }
func (s *Server) DatabaseSize() int64  { return fileSize(s.cfg.DatabasePath) }

// LastBackup é o backup mais recente. `ok` em falso é "nenhum ainda", que é
// estado normal — a lista vazia não é erro.
func (s *Server) LastBackup() (string, int64, bool) {
	lista := s.listBackups()
	if len(lista) == 0 {
		return "", 0, false
	}
	return lista[0].Name, lista[0].Size, true
}

func (s *Server) BackupNow(ctx context.Context, at time.Time) error {
	_, err := s.backupDatabase(ctx, at)
	return err
}

func (s *Server) DeleteAccount(r *http.Request, id, callerID int64) error {
	_, _, err := s.deleteAccount(r, id, callerID)
	return err
}

// MintPasswordReset é o par do `MintAccountInvite` que o `server.go` já expunha
// para o hub. As duas devolvem a LINHA, e não o token, porque aquela já estava
// escrita assim — ver a razão na `admin.Deps`.
func (s *Server) MintPasswordReset(ctx context.Context, userID, by int64) (sqlcgen.PasswordReset, error) {
	return s.mintPasswordReset(ctx, userID, by)
}

// IsUnknownUser é o sentinela desta casa visto de fora.
//
// A cena precisa separar "essa conta não existe mais" de "deu erro", porque as
// duas frases são diferentes na tela — e o valor que os distingue é daqui. Um
// predicado bastou: a porta precisou de um `RefusalMotive` porque tinha TRÊS
// casos, e aqui há um.
func (s *Server) IsUnknownUser(err error) bool { return errors.Is(err, errUserNotFound) }
