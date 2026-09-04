package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// A tela de administração (ALE-120), pelo router real. O que está aqui são as
// consequências: apagar uma conta MOVE as mesas dela, o backup é um snapshot
// que abre, e nada disso responde a quem não é admin.

// Aqui morava o TestAdminScreenRoutesRejectEveryoneElse. A rota saiu na ALE-277 e a garantia
// está em `TestANonAdminDoesNotReachTheInviteRoute`, na cena.

// Aqui morava o TestAdminUserListCountsWhatEachAccountOwns. A rota saiu na ALE-277 e a garantia
// está em `TestHoldingsAndHowTheyRead`, no `web/admin`.

func TestDeletingAnAccountMovesItsCampaignsToTheAdmin(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	campaign := seedCampaign(t, s, player)

	// A REGRA direto, e não a rota: `DELETE /admin/users/{id}` saiu na ALE-277,
	// e o que este caso prende é para onde vão as MESAS de quem some — que é
	// decisão de produto e não de transporte.
	movidas, _, err := s.deleteAccount(httptest.NewRequest(http.MethodDelete, "/", nil), player, admin)
	if err != nil {
		t.Fatalf("apagar a conta falhou: %v", err)
	}
	if movidas != 1 {
		t.Errorf("mesas transferidas = %d, esperado 1", movidas)
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

// Aqui morava o TestTheAdminCannotDeleteThemselves. A rota saiu na ALE-277 e a garantia
// está em `TestThePanelDoesNotOfferDeletingYourOwnAccount`, no `web/admin`.

func TestTheBackupIsADatabaseThatOpens(t *testing.T) {
	s := newTestServer(t, adminEmail)
	s.cfg.BackupDir = filepath.Join(t.TempDir(), "backups")
	seedUser(t, s, adminEmail)

	// A REGRA direto: o que este caso prende é que a cópia ABRE como banco, e
	// isso nunca foi do transporte.
	name, err := s.backupDatabase(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("o backup falhou: %v", err)
	}
	name = filepath.Base(name)
	copyPath := filepath.Join(s.cfg.BackupDir, name)
	if _, err := os.Stat(copyPath); err != nil {
		t.Fatalf("o backup não está no disco: %v", err)
	}
	if got := usersInDatabase(t, copyPath); got != 1 {
		t.Errorf("a cópia tem %d contas, esperado a conta semeada", got)
	}

	if backups := s.listBackups(); len(backups) != 1 {
		t.Errorf("a listagem não devolveu o backup recém-criado: %+v", backups)
	}
}

// Aqui morava o TestAdminStatusReportsTheRunningServer. A rota saiu na ALE-277 e a garantia
// está em o painel da administração, que desenha o mesmo estado.

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

// openSQLite is the raw driver, not db.Open: the backup must be readable as it
// came out, with no migration run over it to paper a problem.
func openSQLite(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}
