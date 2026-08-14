package api

// Painel de servidor da tela de administração (ALE-120): o que o dono da mesa
// precisa saber sem abrir o terminal, e o backup.
//
// O que NÃO está aqui é deliberado: resetar e semear o banco ficam no terminal.
// Um botão destrutivo num celular, no meio da sessão, com o dedo perto, é
// acidente esperando — e é operação de uma vez por ano, com o notebook aberto.

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type serverStatusDTO struct {
	Environment  string `json:"environment"`
	DatabasePath string `json:"databasePath"`
	DatabaseSize int64  `json:"databaseSize"`
	Users        int64  `json:"users"`
	Campaigns    int64  `json:"campaigns"`
	Characters   int64  `json:"characters"`
}

type backupDTO struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
}

// handleAdminStatus: GET /admin/status.
func (s *Server) handleAdminStatus(w http.ResponseWriter, r *http.Request) {
	counts, err := s.queries.TableCounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not read status")
		return
	}
	writeJSON(w, http.StatusOK, serverStatusDTO{
		Environment:  string(s.cfg.AppEnv),
		DatabasePath: s.cfg.DatabasePath,
		DatabaseSize: fileSize(s.cfg.DatabasePath),
		Users:        counts.Users, Campaigns: counts.Campaigns, Characters: counts.Characters,
	})
}

// handleAdminCreateBackup: POST /admin/backups.
func (s *Server) handleAdminCreateBackup(w http.ResponseWriter, r *http.Request) {
	name, err := s.backupDatabase(r.Context(), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not back up: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, backupDTO{
		Name: name, Size: fileSize(filepath.Join(s.cfg.BackupDir, name)), CreatedAt: nowISO(),
	})
}

// handleAdminListBackups: GET /admin/backups, newest first.
func (s *Server) handleAdminListBackups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.listBackups())
}

// backupDatabase writes a consistent snapshot with SQLite's own VACUUM INTO.
//
// Not a file copy, and not the sqlite3 CLI either: with WAL on, the `.db` alone
// is missing whatever still sits in the `-wal`, so copying it while the table is
// live yields an OLD database and no error to say so (ALE-119 measured exactly
// that). VACUUM INTO reads a coherent snapshot of both parts, and being built
// into the driver means the host needs no sqlite3 binary.
func (s *Server) backupDatabase(ctx context.Context, at time.Time) (string, error) {
	if err := os.MkdirAll(s.cfg.BackupDir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir %q: %w", s.cfg.BackupDir, err)
	}
	name := fmt.Sprintf("t20-%s-%s.db", s.cfg.AppEnv, at.Format("20060102-150405"))
	path := filepath.Join(s.cfg.BackupDir, name)
	// VACUUM INTO refuses to overwrite, which is the behaviour we want: two
	// backups in the same second must not silently become one.
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		return "", err
	}
	return name, nil
}

// listBackups reads the directory instead of a table: the files are the truth,
// and one dropped in by the CLI script has to show up here too.
func (s *Server) listBackups() []backupDTO {
	entries, err := os.ReadDir(s.cfg.BackupDir)
	if err != nil {
		return []backupDTO{}
	}
	out := make([]backupDTO, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if e.IsDir() || filepath.Ext(e.Name()) != ".db" || err != nil {
			continue
		}
		out = append(out, backupDTO{Name: e.Name(), Size: info.Size(), CreatedAt: isoAt(info.ModTime())})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
