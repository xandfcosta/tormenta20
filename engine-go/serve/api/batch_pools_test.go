package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// O LOTE TEM DE DAR O MESMO QUE A FICHA INTEIRA.
//
// # Por que este guarda é obrigatório
//
// O `sheet.PoolsForCharacters` monta um agregado PARCIAL — a linha, as raças, as
// classes e os itens — e o `sheet.Load` monta o agregado inteiro. São dois
// caminhos até o mesmo número, e dois caminhos é a família de defeito que a
// ALE-355 existe para apagar: o `hpMax` gravado e o derivado discordavam havia
// meses num bardo, sem nada acusar.
//
// O modo de falha é o mesmo de sempre, SILENCIOSO: o parcial deixar de carregar
// alguma coisa que o `engine.VitalContextFor` lê. Hoje ele lê os escalares, as
// classes, a raça e os itens; no dia em que ler mais uma relação, a Mesa
// mostraria uma barra e a ficha do mesmo personagem mostraria outra.
//
// # Por que ele roda sobre a SEMENTE e não sobre a bancada
//
// Porque o defeito depende do DADO. Um guerreiro sem item mágico fecha por
// qualquer caminho; quem separa os dois caminhos é o personagem com item que
// mexe em atributo, o multiclasse e o que tem raça com bônus. Semear isso à mão
// seria escolher quais casos existem — e o elenco da semente já tem os três,
// porque ele é a vitrine do app. Percorrê-lo inteiro é mais barato do que
// escolher e errar a escolha.
func TestTheBatchPoolsMatchTheOneByOnePools(t *testing.T) {
	s := newTestServer(t)
	applySeedFile(t, s)
	ctx := context.Background()

	sheets := allCharacters(t, s)
	ids := make([]int64, len(sheets))
	for i, c := range sheets {
		ids[i] = c.ID
	}
	inBatch, err := sheet.PoolsForCharacters(ctx, s.queries, s.catalogs, ids)
	if err != nil {
		t.Fatalf("poços em lote: %v", err)
	}

	compared := 0
	for _, c := range sheets {
		dto, err := sheet.Load(ctx, s.queries, s.catalogs, c)
		if err != nil {
			t.Fatalf("carregar %q: %v", c.Name, err)
		}
		oneByOne := sheet.Pools{
			HpMax: dto.HpMax, HpCurrent: dto.HpCurrent,
			MpMax: dto.MpMax, MpCurrent: dto.MpCurrent,
		}
		compared++
		if inBatch[c.ID] != oneByOne {
			t.Errorf("%q (id %d): o lote deu %+v e a ficha inteira deu %+v.\n"+
				"O agregado PARCIAL do `partialSheetsForPools` perdeu alguma coisa que o\n"+
				"`engine.VitalContextFor` lê — confira o que ele carrega contra o que o\n"+
				"contexto consome.", c.Name, c.ID, inBatch[c.ID], oneByOne)
		}
	}

	// O DENOMINADOR: um elenco vazio e um lote que bate se parecem no terminal.
	// A semente tem dezesseis personagens, e o piso é baixo de propósito — o que
	// ele pega é a semente não chegar ao banco, não o elenco encolher de um.
	if compared < 10 {
		t.Fatalf("o guarda comparou só %d fichas — a semente não chegou ao banco", compared)
	}
}

// applySeedFile aplica o `seed.sql` commitado no banco do caso.
//
// É o MESMO arquivo que uma máquina nova recebe, e o `cmd/seed` tem guarda
// próprio para ele estar em dia — então este caso não precisa repetir a
// pergunta "a semente está atual?".
func applySeedFile(t *testing.T, s *Server) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "seed.sql"))
	if err != nil {
		t.Fatalf("ler o seed.sql: %v", err)
	}
	if _, err := s.db.Exec(string(raw)); err != nil {
		t.Fatalf("aplicar o seed.sql: %v", err)
	}
}

func allCharacters(t *testing.T, s *Server) []sqlcgen.Character {
	t.Helper()
	rows, err := s.db.Query(`SELECT id FROM characters ORDER BY id`)
	if err != nil {
		t.Fatalf("listar o elenco: %v", err)
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("ler um id: %v", err)
		}
		ids = append(ids, id)
	}
	sheets := make([]sqlcgen.Character, 0, len(ids))
	for _, id := range ids {
		c, err := s.queries.GetCharacter(context.Background(), id)
		if err != nil {
			t.Fatalf("ler a ficha %d: %v", id, err)
		}
		sheets = append(sheets, c)
	}
	return sheets
}
