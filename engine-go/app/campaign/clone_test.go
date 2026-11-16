package campaign

import (
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"t20engine/infra/db"
)

// O CLONE E O SCHEMA ANDAM JUNTOS. A lista do que a cópia leva era escrita à mão, e
// cada tabela nova que pendia do item precisava ser lembrada aqui também; três delas
// não foram (a escolha do item, a da melhoria e a marca de feita por mim), e a cópia
// da mesa chegava sem elas. Nada falhava: uma tabela esquecida só é uma cópia mais
// pobre.
//
// O guarda lê o SCHEMA migrado — e não uma lista de tabelas escrita no teste, que
// envelheceria do mesmo jeito — e exige de cada tabela que pende da ficha ou do item
// uma de duas coisas: um passo do `cloneSteps`, ou uma linha abaixo dizendo por que a
// cópia nasce sem ela.
var notCloned = map[string]string{
	// O ESTADO DE JOGO fica no rascunho: a cópia nasce inteira (ALE-564), e o rascunho
	// não tem estado de jogo nenhum desde a migração 23.
	"character_damage":           "estado de jogo: a cópia nasce com o PV e o PM cheios",
	"character_nonlethal_damage": "estado de jogo: a cópia nasce sem dano não letal",
	"active_effects":             "estado de jogo: a cópia nasce sem efeito ativo",
	"character_conditionals":     "estado de jogo: as situações ligadas são da cena",
	"character_power_uses":       "estado de jogo: a cópia nasce com os usos inteiros",
	"character_stances":          "estado de jogo: a cópia nasce sem postura",
	"character_wild_shape":       "estado de jogo: a cópia nasce na forma dela",
	// A CAMPANHA aponta para a ficha da mesa, e não o contrário: são da campanha, e o
	// rascunho não tem campanha.
	"campaign_members":      "da campanha: quem cria o membro é a entrada na mesa",
	"campaign_grants":       "da campanha: o mestre concede à cópia, já na mesa",
	"campaign_improvements": "da campanha: o mestre concede à cópia, já na mesa",
	"campaign_silences":     "da campanha: o mestre silencia na mesa",
	"campaign_rules":        "da campanha: a regra da mesa é do mestre, e nasce na mesa (ALE-612)",
}

var insertedTable = regexp.MustCompile(`INSERT INTO (\w+)`)

func TestEveryCharacterTableIsClonedOrSaysWhyNot(t *testing.T) {
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "schema.db"))
	if err != nil {
		t.Fatalf("abrir o banco migrado: %v", err)
	}
	defer sqlDB.Close()
	// A DESCENDÊNCIA da ficha pelas chaves estrangeiras, e não pelo texto do schema: a
	// escolha do encanto pende do ENCANTO, que pende do item, e um `REFERENCES
	// characters(id)` casado no texto a deixava de fora.
	rows, err := sqlDB.Query(`SELECT m.name, f."table" FROM sqlite_master m, pragma_foreign_key_list(m.name) f
		WHERE m.type = 'table'`)
	if err != nil {
		t.Fatalf("ler as chaves estrangeiras: %v", err)
	}
	parents := map[string][]string{}
	for rows.Next() {
		var table, parent string
		if err := rows.Scan(&table, &parent); err != nil {
			t.Fatal(err)
		}
		parents[table] = append(parents[table], parent)
	}
	rows.Close()
	var children []string
	for table := range parents {
		if table != "characters" && descendsFromCharacters(table, parents, map[string]bool{}) {
			children = append(children, table)
		}
	}
	slices.Sort(children)
	// O CONTROLE: o caso só mede alguma coisa se achou as filhas que se sabe existirem,
	// a neta incluída.
	if !slices.Contains(children, "character_items") || !slices.Contains(children, "character_item_enchant_choices") {
		t.Fatalf("a varredura do schema não achou as tabelas do item (%v): o caso mediria o vazio", children)
	}

	cloned := map[string]bool{}
	for _, step := range cloneSteps(0, 0, "") {
		for _, m := range insertedTable.FindAllStringSubmatch(step.sql, -1) {
			cloned[m[1]] = true
		}
	}
	for _, table := range children {
		_, explained := notCloned[table]
		switch {
		case cloned[table] && explained:
			t.Errorf("%s é copiada pelo `cloneSteps` E está no `notCloned`: tire de um dos dois", table)
		case !cloned[table] && !explained:
			t.Errorf("%s pende da ficha ou do item e a cópia da mesa nasce sem ela: "+
				"acrescente um passo ao `cloneSteps` (app/campaign/clone.go), ou diga no `notCloned` por que não", table)
		}
	}
	for table := range notCloned {
		if !slices.Contains(children, table) {
			t.Errorf("o `notCloned` explica %s, que não pende mais da ficha: tire a linha", table)
		}
	}
}

// descendsFromCharacters diz se a tabela pende da ficha, direto ou por outra tabela.
func descendsFromCharacters(table string, parents map[string][]string, seen map[string]bool) bool {
	if seen[table] {
		return false
	}
	seen[table] = true
	for _, parent := range parents[table] {
		if parent == "characters" || descendsFromCharacters(parent, parents, seen) {
			return true
		}
	}
	return false
}
