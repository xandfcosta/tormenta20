package api

// Painel de servidor da tela de administração: o que o dono da mesa precisa
// saber sem abrir o terminal, e o backup.
//
// O que NÃO está aqui é deliberado: resetar e semear o banco ficam no terminal.
// Um botão destrutivo num celular, no meio da sessão, com o dedo perto, é
// acidente esperando — e é operação de uma vez por ano, com o notebook aberto.

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"t20engine/infra/db/dbvalue"
	"time"
)

type backupDTO struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"createdAt"`
}

// backupDatabase grava um instantâneo coerente com o próprio VACUUM INTO do
// SQLite.
//
// NÃO é cópia de arquivo, nem o CLI do sqlite3: com o WAL ligado, o `.db`
// sozinho não tem o que ainda está no `-wal`, então copiá-lo com a mesa viva
// produz um banco VELHO e nenhum erro que o diga. O `VACUUM INTO` lê as duas
// partes de uma vez, e por vir no driver dispensa um binário do sqlite3 no
// hospedeiro.
func (h adminHost) backupDatabase(ctx context.Context, at time.Time) (string, error) {
	if err := os.MkdirAll(h.cfg.BackupDir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir %q: %w", h.cfg.BackupDir, err)
	}
	name := fmt.Sprintf("t20-%s-%s.db", h.cfg.AppEnv, at.Format("20060102-150405"))
	path := filepath.Join(h.cfg.BackupDir, name)
	// O `VACUUM INTO` se recusa a sobrescrever, e é o comportamento que se quer:
	// dois backups no mesmo segundo não podem virar um em silêncio.
	if _, err := h.db.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		return "", err
	}
	return name, nil
}

// listBackups lê o DIRETÓRIO e não uma tabela: os arquivos são a verdade, e um
// largado ali pelo script do terminal tem de aparecer aqui também.
func (h adminHost) listBackups() []backupDTO {
	entries, err := os.ReadDir(h.cfg.BackupDir)
	if err != nil {
		return []backupDTO{}
	}
	out := make([]backupDTO, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if e.IsDir() || filepath.Ext(e.Name()) != ".db" || err != nil {
			continue
		}
		out = append(out, backupDTO{Name: e.Name(), Size: info.Size(), CreatedAt: dbvalue.IsoAt(info.ModTime())})
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

// ScheduleBackups faz o backup periódico e apaga os mais antigos.
//
// Periódico e não só manual: backup que depende de memória humana é backup que
// não existe na noite em que importa.
//
// Roda em goroutine própria e morre com o contexto do processo, junto com o
// desligamento gracioso. NÃO faz backup no boot: subir o servidor três vezes
// seguidas para mexer numa configuração não deve encher a pasta.
func (s *Server) ScheduleBackups(ctx context.Context) {
	if s.cfg.BackupEvery <= 0 || s.cfg.BackupKeep <= 0 {
		log.Printf("backup automático desligado (a cada %s, guardando %d)", s.cfg.BackupEvery, s.cfg.BackupKeep)
		return
	}
	log.Printf("backup automático a cada %s, guardando os %d últimos em %s",
		s.cfg.BackupEvery, s.cfg.BackupKeep, s.cfg.BackupDir)

	ticker := time.NewTicker(s.cfg.BackupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case at := <-ticker.C:
			name, err := s.adminHost().backupDatabase(ctx, at)
			if err != nil {
				// Melhor esforço de propósito: a mesa não pode parar porque o
				// disco encheu. Mas o erro é DITO, não engolido.
				log.Printf("backup automático falhou: %v", err)
				continue
			}
			log.Printf("backup automático: %s", name)
			s.pruneBackups()
		}
	}
}

// pruneBackups apaga os backups além do teto, do mais antigo para o mais novo.
//
// Apaga só o que ELE mesmo reconhece como backup — a mesma listagem que a tela
// de administração usa, que já filtra por extensão `.db` no diretório
// configurado. Um arquivo qualquer largado ali não é candidato a ser apagado.
func (s *Server) pruneBackups() {
	backups := s.adminHost().listBackups() // mais novo primeiro
	if len(backups) <= s.cfg.BackupKeep {
		return
	}
	for _, old := range backups[s.cfg.BackupKeep:] {
		path := filepath.Join(s.cfg.BackupDir, old.Name)
		if err := os.Remove(path); err != nil {
			log.Printf("não consegui apagar o backup antigo %s: %v", old.Name, err)
			continue
		}
		log.Printf("backup antigo removido: %s", old.Name)
	}
}
