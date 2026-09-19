package character

import (
	"context"
	"fmt"

	"t20engine/domain/sheet"
)

// AS ESCRITAS DA MOCHILA.
//
// # Por que `UPDATE` escrito à mão, e não uma consulta gerada
//
// Nenhuma das três colunas tem query no sqlc, e esta camada É onde uma escrita
// sem consulta gerada pode morar — foi por não haver este lugar que elas viviam
// no `serve/api`, montadas por um construtor genérico de `SET`. O conserto
// definitivo é escrever a consulta no `query.sql` e regerar; até lá o SQL está
// aqui, inteiro e visível, que é a mesma decisão que o `session.Rename` e o
// `session.SaveNotes` tomaram na ALE-344.
//
// **Nenhuma delas toca carimbo**: `character_items` não tem coluna `updatedAt`.
// A ficha tem, e é por isso que o `choices.go` ao lado carimba e este não.

// SaveCustomItem grava nome, quantidade e espaços de um item da mochila.
//
// `espacos` é `float64` porque a coluna `slots` é REAL: a carga do livro conta
// de meio em meio (uma adaga ocupa 1, um bálsamo 0,5, p141).
func (p Plays) SaveCustomItem(
	ctx context.Context, itemID int64, nome string, quantidade int64, espacos float64,
) error {
	if _, err := p.db.ExecContext(ctx,
		"UPDATE character_items SET name = ?, quantity = ?, slots = ? WHERE id = ?",
		nome, quantidade, espacos, itemID,
	); err != nil {
		return fmt.Errorf("gravar o item %d (%q): %w", itemID, nome, err)
	}
	return nil
}

// SaveEquipped põe o item num lugar do corpo, e o VAZIO o tira da mão.
//
// O lugar atravessa como `string` e não como `sql.NullString`: o que é NULL na
// coluna é detalhe do banco, e os três lugares do livro — `vested`, `wielded`,
// `wielded2` — nunca são a string vazia, então "" só pode querer dizer uma
// coisa. Enquanto isto morava no adaptador, a porta da cena pedia um
// `sql.NullString` — um tipo do `database/sql` viajando por uma fronteira que
// existe justamente para o banco não atravessar.
func (p Plays) SaveEquipped(ctx context.Context, itemID int64, lugar string) error {
	var coluna any // nil vira NULL, que é o item guardado na mochila
	if lugar != "" {
		coluna = lugar
	}
	if _, err := p.db.ExecContext(ctx,
		"UPDATE character_items SET equipped = ? WHERE id = ?", coluna, itemID,
	); err != nil {
		return fmt.Errorf("gravar o lugar %q do item %d: %w", lugar, itemID, err)
	}
	return nil
}

// SaveItemOverlays grava a melhoria e o material escolhidos.
//
// A cena manda a LISTA e o nome do material; a serialização em JSON e a
// tradução de material vazio para NULL são daqui.
func (p Plays) SaveItemOverlays(
	ctx context.Context, itemID int64, melhorias []string, material string,
) error {
	var coluna any
	if material != "" {
		coluna = material
	}
	if _, err := p.db.ExecContext(ctx,
		"UPDATE character_items SET improvements = ?, material = ? WHERE id = ?",
		sheet.MarshalStrings(&melhorias), coluna, itemID,
	); err != nil {
		return fmt.Errorf("gravar as melhorias do item %d: %w", itemID, err)
	}
	return nil
}
