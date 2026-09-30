package character

import (
	"context"
	"database/sql"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
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

// AddCatalogItem põe na mochila um item do Capítulo 3.
//
// O NOME e os ESPAÇOS saem do catálogo e não de quem chama, e isso é a regra e
// não zelo: são dado transcrito do livro, e deixar o navegador mandá-los abriria
// a porta para uma "Espada longa" de 0 espaços. Quem chama diz QUAL item e
// QUANTOS, que é tudo o que ele sabe.
func (p Plays) AddCatalogItem(
	ctx context.Context, characterID int64, catalogID string, amount int64,
) error {
	catalog := book.ItemByID(catalogID)
	if catalog == nil {
		return fmt.Errorf("o item %q não existe no livro", catalogID)
	}
	if _, err := p.queries.CreateItem(ctx, sqlcgen.CreateItemParams{
		Characterid:  characterID,
		Catalogid:    sql.NullString{String: catalog.ID, Valid: true},
		Name:         catalog.Name,
		Quantity:     amount,
		Slots:        catalog.Slots,
		Improvements: "[]",
		Createdat:    dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("pôr %q na mochila de %d: %w", catalog.Name, characterID, err)
	}
	return nil
}

// AddCustomItem cria o item que o livro não tem — a lembrança de um NPC, a
// chave de um cofre.
func (p Plays) AddCustomItem(
	ctx context.Context, characterID int64, name string, amount int64, spaces float64,
) error {
	if _, err := p.queries.CreateItem(ctx, sqlcgen.CreateItemParams{
		Characterid: characterID, Name: name, Quantity: amount, Slots: spaces,
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("criar %q na mochila de %d: %w", name, characterID, err)
	}
	return nil
}

// RemoveItem tira o item da ficha.
//
// A POSSE não é conferida aqui, e é a mesma decisão do resto do `Plays`: quem
// chama já leu a linha e sabe de quem ela é. Ver o `doc.go`.
func (p Plays) RemoveItem(ctx context.Context, itemID int64) error {
	if err := p.queries.DeleteItem(ctx, itemID); err != nil {
		return fmt.Errorf("tirar o item %d da ficha: %w", itemID, err)
	}
	return nil
}

// SaveCustomItem grava nome, quantidade e espaços de um item da mochila.
//
// `spaces` é `float64` porque a coluna `slots` é REAL: a carga do livro conta
// de meio em meio (uma adaga ocupa 1, um bálsamo 0,5, p141).
func (p Plays) SaveCustomItem(
	ctx context.Context, itemID int64, name string, amount int64, spaces float64,
) error {
	if _, err := p.db.ExecContext(ctx,
		"UPDATE character_items SET name = ?, quantity = ?, slots = ? WHERE id = ?",
		name, amount, spaces, itemID,
	); err != nil {
		return fmt.Errorf("gravar o item %d (%q): %w", itemID, name, err)
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
func (p Plays) SaveEquipped(ctx context.Context, itemID int64, place string) error {
	var column any // nil vira NULL, que é o item guardado na mochila
	if place != "" {
		column = place
	}
	if _, err := p.db.ExecContext(ctx,
		"UPDATE character_items SET equipped = ? WHERE id = ?", column, itemID,
	); err != nil {
		return fmt.Errorf("gravar o lugar %q do item %d: %w", place, itemID, err)
	}
	return nil
}

// SaveItemOverlays grava a melhoria e o material escolhidos.
//
// A cena manda a LISTA e o nome do material; a serialização em JSON e a
// tradução de material vazio para NULL são daqui.
func (p Plays) SaveItemOverlays(
	ctx context.Context, itemID int64, improvements []string, material string,
) error {
	var column any
	if material != "" {
		column = material
	}
	_, err := p.db.ExecContext(ctx,
		"UPDATE character_items SET improvements = ?, material = ? WHERE id = ?",
		sheet.MarshalStrings(&improvements), column, itemID,
	)
	return err
}

// EnchantItem grava os encantos de UMA arma, e é a porta do MESTRE.
//
// Ela é separada do `SaveItemOverlays` porque quem chama é outra pessoa: a
// melhoria e o material o jogador escolhe na mochila dele; o encanto vem do
// mestre, que é quem entrega o saque (p333 — encanto não se compra, se acha ou
// se fabrica). Juntá-las num método só daria à cena do jogador um parâmetro que
// ela nunca pode preencher, e é assim que uma fronteira vira convenção.
//
// A REGRA É CONFERIDA AQUI e não na tela do mestre, pela razão de sempre:
// filtro de tela não recusa nada. `enchants` vazio limpa a arma.
//
// @example plays.EnchantItem(ctx, itemID, []string{"encanto-formidavel"})
func (p Plays) EnchantItem(ctx context.Context, itemID int64, enchants []string) error {
	row, err := sqlcgen.New(p.db).GetItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("ler o item %d para encantar: %w", itemID, err)
	}
	// Item CUSTOM não tem verbete, então não tem arma — e o `FitsWeaponEnchants`
	// recusa pelo nil, com a frase do livro.
	var catalog *book.Item
	if row.Catalogid.Valid {
		catalog = book.ItemByID(row.Catalogid.String)
	}
	if err := book.FitsWeaponEnchants(catalog, enchants); err != nil {
		return err
	}
	// GRAVAR É APAGAR E REESCREVER: quem chama manda o conjunto inteiro, como
	// manda a lista de melhorias, e um diff aqui teria de reimplementar o que o
	// `DELETE` + `INSERT` já faz — sobre um conjunto de no máximo três (p334).
	//
	// EM TRANSAÇÃO porque são até quatro escritas: sem ela, um `INSERT` que
	// falhasse no meio deixaria a arma com o conjunto pela METADE, e o `DELETE`
	// já teria levado os encantos que ela tinha. O estado perdido não é
	// recuperável — ninguém sabe o que estava lá antes.
	return p.inTx(ctx, fmt.Sprintf("encantar o item %d", itemID), func(q *sqlcgen.Queries) error {
		if err := q.ClearItemEnchants(ctx, itemID); err != nil {
			return err
		}
		for _, id := range enchants {
			if err := q.AddItemEnchant(ctx, sqlcgen.AddItemEnchantParams{
				Itemid: itemID, Enchantid: id,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
