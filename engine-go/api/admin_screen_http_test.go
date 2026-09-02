package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// A tela de administração (ALE-120), pelo router real. O que está aqui são as
// consequências: apagar uma conta MOVE as mesas dela, o backup é um snapshot
// que abre, e nada disso responde a quem não é admin.

func TestAdminScreenRoutesRejectEveryoneElse(t *testing.T) {
	s := newTestServer(t, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")

	routes := []struct{ method, path string }{
		{http.MethodGet, "/admin/users"},
		{http.MethodDelete, "/admin/users/1"},
		{http.MethodPost, "/admin/users/1/password-reset"},
		{http.MethodGet, "/admin/invites"},
		{http.MethodGet, "/admin/status"},
		{http.MethodGet, "/admin/backups"},
		{http.MethodPost, "/admin/backups"},
	}
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			if rec := anon(t, s, route.method, route.path); rec.Code != http.StatusUnauthorized {
				t.Errorf("anônimo: esperado 401, veio %d", rec.Code)
			}
			rec := authed(t, s, player, route.method, route.path, "")
			if rec.Code != http.StatusForbidden {
				t.Errorf("jogador: esperado 403, veio %d (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

// A lista existe para dizer o que apagar uma conta custaria, então os números
// são a razão de ela existir.
func TestAdminUserListCountsWhatEachAccountOwns(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	seedCampaign(t, s, player)
	seedCharacter(t, s, player, "Herói", 10, 10, 5, 5)

	rec := authed(t, s, admin, http.MethodGet, "/admin/users", "")

	var users []adminUserDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("resposta não é uma lista (%s): %v", rec.Body.String(), err)
	}
	byEmail := map[string]adminUserDTO{}
	for _, u := range users {
		byEmail[u.Email] = u
	}
	if got := byEmail["jogador@t20.local"]; got.Campaigns != 1 || got.Characters != 1 {
		t.Errorf("jogador = %+v, esperado 1 mesa e 1 ficha", got)
	}
	if !byEmail[adminEmail].IsAdmin || byEmail["jogador@t20.local"].IsAdmin {
		t.Errorf("o papel não veio certo: %s", rec.Body.String())
	}
}

// A decisão do dono: a crônica sobrevive à saída do jogador.
func TestDeletingAnAccountMovesItsCampaignsToTheAdmin(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	campaign := seedCampaign(t, s, player)

	rec := authed(t, s, admin, http.MethodDelete, "/admin/users/"+id64(player), "")

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if moved, _ := jsonField(t, rec, "transferredCampaigns").(float64); moved != 1 {
		t.Errorf("transferredCampaigns = %v, esperado 1 (%s)", moved, rec.Body.String())
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

// A lista mostra a SUA linha e o mesmo menu — apagar a si mesmo levaria suas
// mesas para lugar nenhum.
func TestTheAdminCannotDeleteThemselves(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)

	rec := authed(t, s, admin, http.MethodDelete, "/admin/users/"+id64(admin), "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if countUsers(t, s) != 1 {
		t.Error("a própria conta foi apagada")
	}
}

// O backup pela tela é o mesmo snapshot WAL-safe do script: tem de ABRIR, e é
// por isso que o teste lê o arquivo em vez de conferir que ele existe.
func TestTheBackupIsADatabaseThatOpens(t *testing.T) {
	s := newTestServer(t, adminEmail)
	s.cfg.BackupDir = filepath.Join(t.TempDir(), "backups")
	admin := seedUser(t, s, adminEmail)

	rec := authed(t, s, admin, http.MethodPost, "/admin/backups", "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, veio %d (%s)", rec.Code, rec.Body.String())
	}
	name, _ := jsonField(t, rec, "name").(string)
	copyPath := filepath.Join(s.cfg.BackupDir, name)
	if _, err := os.Stat(copyPath); err != nil {
		t.Fatalf("o backup não está no disco: %v", err)
	}
	if got := usersInDatabase(t, copyPath); got != 1 {
		t.Errorf("a cópia tem %d contas, esperado a conta semeada", got)
	}

	list := authed(t, s, admin, http.MethodGet, "/admin/backups", "")
	var backups []backupDTO
	if err := json.Unmarshal(list.Body.Bytes(), &backups); err != nil || len(backups) != 1 {
		t.Errorf("a listagem não devolveu o backup recém-criado: %s", list.Body.String())
	}
}

func TestAdminStatusReportsTheRunningServer(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	seedCampaign(t, s, admin)

	rec := authed(t, s, admin, http.MethodGet, "/admin/status", "")

	var status serverStatusDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("resposta inválida (%s): %v", rec.Body.String(), err)
	}
	if status.Users != 1 || status.Campaigns != 1 {
		t.Errorf("contagens = %+v, esperado 1 conta e 1 mesa", status)
	}
	if status.DatabasePath == "" {
		t.Error("o caminho do banco é o que o dono usa para achar o arquivo")
	}
}

// usersInDatabase opens a SQLite file directly — the point is that the backup
// is a working database, not a file of the right size.
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
