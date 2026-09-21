package api

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// AS CONSEQUÊNCIAS das duas ações pesadas da administração: apagar uma conta
// MOVE as mesas dela, e o backup é um snapshot que abre de verdade.
//
// Nenhuma das duas passa por rota aqui, e é deliberado — o que elas prendem é
// decisão de produto, não transporte. A AUTORIZAÇÃO tem casa própria, na cena.

// A recusa a quem não é admin NÃO se prende aqui: ela é do
// `TestANonAdminDoesNotReachTheInviteRoute`, na cena.

// A contagem do que cada conta possui é do `TestHoldingsAndHowTheyRead`, no
// `web/admin`.

func TestDeletingAnAccountMovesItsCampaignsToTheAdmin(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	campaign := seedCampaign(t, s, player)

	// A REGRA direto, e não a rota: o que este caso prende é para onde vão as
	// MESAS de quem some, que é decisão de produto e não de transporte.
	moved, err := s.accountRoster().Delete(context.Background(), admin, player)
	if err != nil {
		t.Fatalf("apagar a conta falhou: %v", err)
	}
	if moved != 1 {
		t.Errorf("mesas transferidas = %d, esperado 1", moved)
	}
	row, err := s.queries.GetCampaign(context.Background(), campaign)
	if err != nil {
		t.Fatalf("a mesa não podia ter sumido junto: %v", err)
	}
	if row.Ownerid != admin {
		t.Errorf("dono da mesa = %d, esperado o admin (%d)", row.Ownerid, admin)
	}
	if countUsers(t, s) != 1 {
		t.Errorf("a conta tinha de ter sido apagada, restaram %d", countUsers(t, s))
	}
}

// O admin não apagar a si mesmo é do
// `TestThePanelDoesNotOfferDeletingYourOwnAccount`, no `web/admin`.

func TestTheBackupIsADatabaseThatOpens(t *testing.T) {
	s := newTestServer(t, adminEmail)
	s.cfg.BackupDir = filepath.Join(t.TempDir(), "backups")
	seedUser(t, s, adminEmail)

	// A REGRA direto: o que este caso prende é que a cópia ABRE como banco, e
	// isso nunca foi do transporte.
	name, err := s.adminHost().backupDatabase(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("o backup falhou: %v", err)
	}
	name = filepath.Base(name)
	// O NOME CARREGA A DATA, e isso é do operador e não da máquina: quem procura
	// "o backup de antes do descanso" acha pelo carimbo, e a poda depende dos
	// nomes serem distintos por segundo.
	//
	// O que ele protege é a família que o `go vet` ACEITA — trocar o LAYOUT da
	// data. (Verbo errado no `fmt.Sprintf` o próprio `vet` pega, e ele roda
	// dentro do `go test`.) Sabotado com `time.RFC3339`, o nome sai
	// `t20--2026-09-04T10:36:07-03:00.db`: dois-pontos em nome de arquivo, e a
	// poda, que ordena por nome, deixa de ordenar por tempo.
	//
	// O AMBIENTE é opcional no padrão de propósito: a bancada não configura
	// `APP_ENV`, e o nome sai com dois traços seguidos. O que se prende aqui é o
	// CARIMBO, que é o que a poda usa e o que a pessoa lê — inventar uma
	// exigência sobre o ambiente seria prender o que ninguém prometeu.
	if !regexp.MustCompile(`^t20-[a-z]*-\d{8}-\d{6}\.db$`).MatchString(name) {
		t.Errorf("o backup se chama %q, e não `t20-<ambiente>-<data>-<hora>.db`", name)
	}
	copyPath := filepath.Join(s.cfg.BackupDir, name)
	if _, err := os.Stat(copyPath); err != nil {
		t.Fatalf("o backup não está no disco: %v", err)
	}
	if got := usersInDatabase(t, copyPath); got != 1 {
		t.Errorf("a cópia tem %d contas, esperado a conta semeada", got)
	}

	if backups := s.adminHost().listBackups(); len(backups) != 1 {
		t.Errorf("a listagem não devolveu o backup recém-criado: %+v", backups)
	}
}

// O estado do servidor em execução é desenhado pelo painel da administração, e é
// lá que ele se prende.

func usersInDatabase(t *testing.T, path string) int {
	t.Helper()
	database, err := openSQLite(path)
	if err != nil {
		t.Fatalf("abrir %q: %v", path, err)
	}
	defer func() { _ = database.Close() }()
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("ler a cópia: %v", err)
	}
	return count
}

// openSQLite é o driver cru, e não o `db.Open`: o backup tem de ser legível como
// saiu, sem migração rodando por cima para tapar um problema.
func openSQLite(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}
