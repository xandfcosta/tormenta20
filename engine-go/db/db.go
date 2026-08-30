// Package db owns the API server's SQLite data layer: schema migrations (goose)
// and the sqlc-generated typed queries. Kept OUT of the engine package so the
// WASM build (engine/ + cmd/wasm) stays dependency-free.
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo) — registers "sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens the SQLite database at path with foreign keys enforced + WAL, then
// applies all pending goose migrations. Uses the pure-Go modernc driver so the
// server cross-compiles cleanly like the rest of engine-go.
func Open(path string) (*sql.DB, error) {
	// Production keeps its database in its own directory (ALE-119), which does
	// not exist on a first boot — and SQLite reports the missing directory as a
	// plain "unable to open database file", which reads like a corrupt file.
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db dir %q: %w", dir, err)
		}
	}
	// `_txlock=immediate`: toda transação pega a trava de ESCRITA já no BEGIN,
	// em vez do `DEFERRED` padrão, que só a pega na primeira escrita (ALE-156).
	//
	// Sem isso, duas requisições simultâneas leem antes de qualquer uma
	// escrever, e as duas passam por uma trava de unicidade que é decidida no
	// código — foi o caso de "um personagem por jogador em cada mesa". Com o
	// lock no BEGIN, a segunda espera a primeira terminar e sua checagem já
	// enxerga o que a primeira gravou.
	//
	// O preço é serializar os ESCRITORES entre si; leitura fora de transação
	// continua livre (WAL), e as oito transações do app são todas de escrita.
	// Numa mesa doméstica isso é de graça, e o `busy_timeout` acima é quem
	// cobre a espera.
	// `synchronous(1)` é NORMAL, e ele vale 139ms POR TOQUE na ficha (ALE-273).
	//
	// O padrão do SQLite é FULL, que faz `fsync` a cada commit. Medido neste
	// repositório, num prato girante: o POST que muda o PV e redevolve a cena
	// levava 121ms de servidor e 139ms até o número mudar na tela. Com NORMAL,
	// no MESMO disco, os dois viraram 1,7ms e 12ms — o número que um SSD daria.
	// O controle que isolou a causa foi trocar só o LUGAR do arquivo (disco →
	// tmpfs) com o binário idêntico.
	//
	// O QUE SE PERDE, dito com precisão: numa queda de energia, os commits que
	// ainda não foram sincronizados. O banco NÃO corrompe — essa é a garantia
	// do WAL, e é o que separa NORMAL de OFF: o arquivo principal nunca fica
	// meio-escrito, porque o conteúdo novo mora no `-wal` e a recuperação relê
	// os quadros até o último commit válido, cada um com checksum.
	//
	// É `innodb_flush_log_at_trx_commit=2` para quem vem do MariaDB, e é o que a
	// documentação do SQLite recomenda para WAL na maioria das aplicações. Para
	// uma ficha de mesa, perder os últimos segundos numa queda de luz custa
	// menos que 139ms em cada clique de PV — decisão do dono, ALE-273.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"+
			"&_pragma=synchronous(1)&_pragma=busy_timeout(5000)&_txlock=immediate",
		path,
	)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}
	if err := migrate(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	// Depois de migrar, CONFERIR: a migração constar aplicada não prova que a
	// tabela existe (ALE-154).
	if err := assertSchema(sqlDB, migrationsFS); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return sqlDB, nil
}

func migrate(sqlDB *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
