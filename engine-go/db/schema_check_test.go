package db

import (
	"path/filepath"
	"strings"
	"testing"
)

// O boot recusa um banco a que falta tabela — mesmo com a migração constando
// aplicada (ALE-154). É o guarda da classe de defeito que deixou o tabuleiro um
// dia inteiro vivendo só em memória.

func TestOpenAcceptsAFreshDatabase(t *testing.T) {
	sqlDB, err := Open(filepath.Join(t.TempDir(), "novo.db"))
	if err != nil {
		t.Fatalf("banco recém-migrado recusado: %v", err)
	}
	_ = sqlDB.Close()
}

// O caso REAL: a tabela some, o `goose_db_version` continua dizendo que a
// migração foi aplicada, e o goose não tem o que fazer. Antes deste guarda, o
// servidor subia inteiro e só o disco sabia.
func TestOpenRefusesADatabaseMissingATable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "furado.db")
	sqlDB, err := Open(path)
	if err != nil {
		t.Fatalf("primeiro boot: %v", err)
	}
	if _, err := sqlDB.Exec("DROP TABLE session_boards"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	_ = sqlDB.Close()

	_, err = Open(path)

	if err == nil {
		t.Fatal("o servidor subiu com uma tabela faltando — é exatamente o defeito")
	}
	// A mensagem tem de dizer QUAL tabela: "schema inválido" mandaria alguém
	// procurar no escuro justamente no momento em que a mesa está esperando.
	if !strings.Contains(err.Error(), "session_boards") {
		t.Errorf("a recusa não nomeia a tabela que falta: %v", err)
	}
}

// A lista de tabelas sai das MIGRAÇÕES, não de uma constante: é o que faz o
// guarda nascer certo a cada migração nova sem ninguém lembrar de nada.
func TestExpectedTablesComesFromTheMigrations(t *testing.T) {
	tables, err := expectedTables(migrationsFS)
	if err != nil {
		t.Fatalf("ler as migrações: %v", err)
	}
	if len(tables) < 10 {
		t.Fatalf("só %d tabelas lidas das migrações (%v) — o regex parou de casar", len(tables), tables)
	}
	for _, esperada := range []string{"users", "characters", "sessions", "session_boards", "campaign_creatures"} {
		if !contains(tables, esperada) {
			t.Errorf("a tabela %q não foi lida das migrações: %v", esperada, tables)
		}
	}
	// O `DROP TABLE` da seção Down não pode entrar como "tem de existir", e o
	// nome do controle do goose não é declarado por migração nenhuma.
	if contains(tables, "goose_db_version") {
		t.Error("a tabela de controle do goose entrou na lista de esperadas")
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// Um banco vazio de verdade (sem nem o goose) é o caso do primeiro boot e tem
// de passar pela migração normalmente — o guarda não pode transformar "banco
// novo" em erro.
func TestOpenTwiceIsStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duas.db")
	for i := 0; i < 2; i++ {
		sqlDB, err := Open(path)
		if err != nil {
			t.Fatalf("boot %d: %v", i+1, err)
		}
		var n int
		if err := sqlDB.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table'`).Scan(&n); err != nil {
			t.Fatalf("contar tabelas: %v", err)
		}
		_ = sqlDB.Close()
		if n < 10 {
			t.Fatalf("boot %d deixou só %d tabelas", i+1, n)
		}
	}
}
