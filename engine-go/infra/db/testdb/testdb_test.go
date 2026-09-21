package testdb

import (
	"os"
	"testing"

	"t20engine/infra/db"
)

func TestMain(m *testing.M) { os.Exit(Run(m)) }

// O banco copiado ABRE PELO CAMINHO DE PRODUÇÃO, e é isso que faz a bancada
// valer alguma coisa.
//
// `db.Open` roda o `assertSchema` (ALE-154), que RECUSA subir nomeando as
// tabelas que faltam. Então este caso não afirma "o arquivo existe" — ele afirma
// que o molde chegou até aqui com o schema inteiro.
//
// É o que prende a sutileza documentada no `Run`: o molde é FECHADO antes de
// qualquer cópia porque o `Close` faz o checkpoint do WAL. Copiado com escrita
// pendente no `-wal`, ele nasceria sem tabelas — e a falha diria "falta tabela",
// que manda procurar defeito na migração e não na cópia.
func TestAFreshDatabaseOpensThroughProduction(t *testing.T) {
	base, err := db.Open(Fresh(t))
	if err != nil {
		t.Fatalf("abrir o banco copiado: %v", err)
	}
	defer func() { _ = base.Close() }()

	var tables int
	row := base.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table'`)
	if err := row.Scan(&tables); err != nil {
		t.Fatalf("contar tabelas: %v", err)
	}
	// O DENOMINADOR: um banco vazio também "abre", e sem esta conta o caso
	// acima passaria sobre um arquivo de zero byte que o goose acabou de migrar
	// do nada. Dez é folgado — são mais de vinte —, e o número exato mudaria a
	// cada migração nova sem dizer nada.
	if tables < 10 {
		t.Errorf("%d tabelas no banco copiado: o molde não trouxe o schema", tables)
	}
}

// Duas cópias são INDEPENDENTES: escrever numa não aparece na outra.
//
// Sem isto, um `Fresh` que devolvesse o caminho do PRÓPRIO molde passaria em
// tudo — até o primeiro teste que grava, e daí em diante os casos veriam o dado
// uns dos outros na ordem em que rodassem. É o pior defeito possível numa
// bancada: ele não falha, ele acopla.
func TestTwoFreshDatabasesDoNotShareState(t *testing.T) {
	first, segundo := Fresh(t), Fresh(t)
	if first == segundo {
		t.Fatal("as duas cópias são o mesmo arquivo")
	}

	a, err := db.Open(first)
	if err != nil {
		t.Fatalf("abrir a primeira: %v", err)
	}
	defer func() { _ = a.Close() }()
	if _, err := a.Exec(
		`INSERT INTO users (email, passwordHash, name, createdAt, updatedAt) VALUES (?, ?, ?, ?, ?)`,
		"so-na-primeira@t20.local", "x", "Só na primeira", "2026-09-01T00:00:00Z", "2026-09-01T00:00:00Z"); err != nil {
		t.Fatalf("gravar na primeira: %v", err)
	}

	b, err := db.Open(segundo)
	if err != nil {
		t.Fatalf("abrir a segunda: %v", err)
	}
	defer func() { _ = b.Close() }()
	var howMany int
	if err := b.QueryRow(`SELECT count(*) FROM users`).Scan(&howMany); err != nil {
		t.Fatalf("contar na segunda: %v", err)
	}
	if howMany != 0 {
		t.Errorf("a segunda cópia já tem %d usuário(s): as bancadas estão compartilhando estado", howMany)
	}
}
