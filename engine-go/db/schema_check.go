package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// A migração pode CONSTAR aplicada sem a tabela existir (ALE-154).
//
// Foi o que aconteceu: a `session_boards` sumiu do banco de desenvolvimento com
// a 00005 marcada em `goose_db_version`, o goose disse "no migrations to run" e
// o servidor subiu inteiro. O tabuleiro passou um dia vivendo só em memória —
// cada gravação falhava numa linha de log que ninguém lê, e a tela estava
// perfeita até o processo reiniciar.
//
// A causa (down parcial, restauração de backup antigo, experimento) importa
// menos que o fato de nada ter acusado. Confiar no `goose_db_version` é confiar
// justamente em quem mentiu; então o boot pergunta ao SCHEMA.

// createTable casa o nome nas duas formas que as migrações usam. `[^\S\n]` no
// lugar de `\s` de propósito: `\s` atravessaria a quebra de linha e engoliria a
// linha seguinte quando um `CREATE TABLE` estivesse mal formatado.
var createTable = regexp.MustCompile(`(?i)CREATE\s+TABLE(?:[^\S\n]+IF[^\S\n]+NOT[^\S\n]+EXISTS)?[^\S\n]+["'` + "`" + `]?(\w+)`)

// dropTable é o irmão do `createTable`, e ele existe porque sem ele o schema
// nunca pode PERDER uma tabela (ALE-205).
//
// A derivação lia só os `CREATE`, então toda tabela criada por qualquer migração
// era esperada para sempre: a `00010` troca a `session_boards` de 1:1 para uma
// linha por tabuleiro aberto, e enquanto o guarda ignorasse o `DROP` da seção
// Up o servidor recusaria subir sobre um banco CORRETO — nomeando como faltante
// exatamente a tabela que a migração acabou de derrubar de propósito.
//
// O guarda continua derivado, que é a propriedade que importa: ele só aprendeu
// o segundo verbo que as migrações usam.
var dropTable = regexp.MustCompile(`(?i)DROP\s+TABLE(?:[^\S\n]+IF[^\S\n]+EXISTS)?[^\S\n]+["'` + "`" + `]?(\w+)`)

// expectedTables lê das PRÓPRIAS migrações embutidas quais tabelas têm de
// existir. Uma lista escrita à mão envelheceria em silêncio, e este repositório
// já foi mordido duas vezes por isso no mesmo dia (o `cloneState` que zerou o
// `TurnsTaken`, o `parseEntryPatch` que descartou o `creatureId`). Derivada,
// ela nasce certa a cada migração nova sem ninguém lembrar de nada.
//
// A ORDEM dos arquivos é o que faz criar e derrubar significarem alguma coisa:
// as migrações rodam em ordem de nome, e a resposta é o schema DEPOIS da última.
// `fs.Glob` já devolve ordenado; o `sort` aqui é para essa garantia estar dita
// onde ela é usada, e não a três pacotes de distância.
func expectedTables(migrations fs.FS) ([]string, error) {
	entries, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("listar migrações: %w", err)
	}
	sort.Strings(entries)
	found := map[string]bool{}
	for _, name := range entries {
		raw, err := fs.ReadFile(migrations, name)
		if err != nil {
			return nil, fmt.Errorf("ler %s: %w", name, err)
		}
		// Só a seção Up: a seção Down descreve o caminho de volta, e uma tabela
		// que ela recria não é uma tabela que o banco de hoje deva ter.
		up := upSection(string(raw))
		for _, match := range createTable.FindAllStringSubmatch(up, -1) {
			found[match[1]] = true
		}
		for _, match := range dropTable.FindAllStringSubmatch(up, -1) {
			delete(found, match[1])
		}
	}
	tables := make([]string, 0, len(found))
	for name := range found {
		tables = append(tables, name)
	}
	sort.Strings(tables)
	return tables, nil
}

// upSection devolve o trecho entre `-- +goose Up` e `-- +goose Down`.
func upSection(sql string) string {
	start := strings.Index(sql, "+goose Up")
	if start < 0 {
		return sql
	}
	rest := sql[start:]
	if end := strings.Index(rest, "+goose Down"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// assertSchema recusa subir quando o banco não tem uma tabela que as migrações
// declaram. Falhar aqui é BARULHENTO de propósito: o modo de falha que este
// guarda existe para matar é justamente o silencioso.
func assertSchema(sqlDB *sql.DB, migrations fs.FS) error {
	wanted, err := expectedTables(migrations)
	if err != nil {
		return err
	}
	rows, err := sqlDB.Query(`SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return fmt.Errorf("ler o schema do banco: %w", err)
	}
	defer rows.Close()

	existing := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("ler o schema do banco: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("ler o schema do banco: %w", err)
	}

	missing := []string{}
	for _, name := range wanted {
		if !existing[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"o banco não tem %d tabela(s) que as migrações declaram: %s."+
				" As migrações constam aplicadas, então o banco foi alterado por fora"+
				" (um `goose down` parcial, uma restauração de backup anterior, um"+
				" experimento). Restaure um backup ou recrie o banco — subir assim faz"+
				" o servidor gravar no vazio em silêncio (ALE-154)",
			len(missing), strings.Join(missing, ", "),
		)
	}
	return nil
}
