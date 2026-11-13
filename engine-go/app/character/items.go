package character

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS ESCRITAS DA MOCHILA.
//
// # Por que `UPDATE` escrito à mão, e não uma consulta gerada
//
// As duas escritas que sobraram não têm query no sqlc, e esta camada É onde uma escrita
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
// a porta para uma "Espada longa" de 0 espaços. Quem chama diz QUAL item, e
// mais nada: cada gesto põe um, e a quantidade maior é do Editar do item.
func (p Plays) AddCatalogItem(ctx context.Context, characterID int64, catalogID string) error {
	catalog := book.ItemByID(catalogID)
	if catalog == nil {
		return fmt.Errorf("o item %q não existe no livro", catalogID)
	}
	return p.inTx(ctx, "pôr "+catalog.Name+" na mochila", func(q *sqlcgen.Queries) error {
		if catalog.StacksInTheBag() {
			summed, err := addOneToPlainLine(ctx, q, characterID, *catalog)
			if err != nil || summed {
				return err
			}
		}
		if _, err := q.CreateItem(ctx, sqlcgen.CreateItemParams{
			Characterid:  characterID,
			Catalogid:    sql.NullString{String: catalog.ID, Valid: true},
			Name:         catalog.Name,
			Quantity:     1,
			Slots:        catalog.Slots,
			Improvements: "[]",
			Createdat:    dbvalue.NowISO(),
		}); err != nil {
			return fmt.Errorf("pôr %q na mochila de %d: %w", catalog.Name, characterID, err)
		}
		return nil
	})
}

// addOneToPlainLine soma um na primeira linha que ainda é só este item do livro, e diz
// se achou uma. Linha no corpo, melhorada ou editada não recebe a soma: a peça
// nova ganharia o lugar, as melhorias ou o nome da outra.
func addOneToPlainLine(ctx context.Context, q *sqlcgen.Queries, characterID int64, catalog book.Item) (bool, error) {
	lines, err := q.ListItemsByCharacter(ctx, characterID)
	if err != nil {
		return false, fmt.Errorf("ler a mochila de %d: %w", characterID, err)
	}
	for _, line := range lines {
		plain := !line.Equipped.Valid && !line.Material.Valid && line.Improvements == "[]"
		sameEntry := line.Catalogid.String == catalog.ID && line.Name == catalog.Name && line.Slots == catalog.Slots
		if !plain || !sameEntry {
			continue
		}
		total := line.Quantity + 1
		if total > sheet.MaxItemQuantity {
			return false, fmt.Errorf("%s somaria %d, e o máximo são %d", catalog.Name, total, sheet.MaxItemQuantity)
		}
		if err := q.SetItemQuantity(ctx, sqlcgen.SetItemQuantityParams{Quantity: total, ID: line.ID}); err != nil {
			return false, fmt.Errorf("somar %q na linha %d: %w", catalog.Name, line.ID, err)
		}
		return true, nil
	}
	return false, nil
}

// AddCustomItem cria o item que o livro não tem — a lembrança de um NPC, a
// chave de um cofre. Ele nunca soma: dois itens caseiros são diferentes,
// mesmo com o mesmo nome.
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

// FollowMaximumChange liga as condições que o PV pede depois de um gesto que moveu o
// máximo. Ver `sheet.FollowMaximumChange`.
func (p Plays) FollowMaximumChange(ctx context.Context, dto, projected sheet.CharacterDTO) error {
	return sheet.FollowMaximumChange(ctx, p.queries, dto, projected)
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
	return p.inTx(ctx, "gravar as melhorias", func(q *sqlcgen.Queries) error {
		err := q.SetItemOverlays(ctx, sqlcgen.SetItemOverlaysParams{
			Improvements: sheet.MarshalStrings(&improvements),
			Material:     sql.NullString{String: material, Valid: material != ""},
			ID:           itemID,
		})
		if err != nil {
			return fmt.Errorf("gravar as melhorias do item %d: %w", itemID, err)
		}
		// A escolha da melhoria vai com ela: a melhoria mora num JSON, e a chave estrangeira não a alcança.
		chosen, err := q.ImprovementChoicesOfItem(ctx, itemID)
		if err != nil {
			return fmt.Errorf("ler as escolhas das melhorias do item %d: %w", itemID, err)
		}
		for _, improvement := range chosen {
			if slices.Contains(improvements, improvement) {
				continue
			}
			err := q.DropImprovementChoice(ctx, sqlcgen.DropImprovementChoiceParams{Itemid: itemID, Improvementid: improvement})
			if err != nil {
				return fmt.Errorf("apagar a escolha da melhoria %s do item %d: %w", improvement, itemID, err)
			}
		}
		if slices.Contains(improvements, sheet.AlchemicalInjection) {
			return nil
		}
		// Sem a melhoria na arma, as doses não têm onde estar (decisão do dono).
		return q.ClearItemDoses(ctx, itemID)
	})
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
// filtro de tela não recusa nada. `enchants` vazio limpa a arma. `choices` é a
// escolha que o encanto pede, por encanto: o tipo de criatura da Anticriatura
// (p335).
//
// @example plays.EnchantItem(ctx, itemID, []string{"encanto-formidavel"}, nil)
func (p Plays) EnchantItem(ctx context.Context, itemID int64, enchants []string, choices map[string]string) error {
	row, err := sqlcgen.New(p.db).GetItem(ctx, itemID)
	if err != nil {
		return fmt.Errorf("ler o item %d para encantar: %w", itemID, err)
	}
	// Item CUSTOM não tem verbete — e o `FitsItemEnchants` recusa pelo nil, com
	// a frase do livro.
	var catalog *book.Item
	if row.Catalogid.Valid {
		catalog = book.ItemByID(row.Catalogid.String)
	}
	if err := book.FitsItemEnchants(catalog, enchants); err != nil {
		return err
	}
	if err := book.FitsEnchantChoices(enchants, choices); err != nil {
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
			chosen, chooses := choices[id]
			if !chooses {
				continue
			}
			if err := q.AddItemEnchantChoice(ctx, sqlcgen.AddItemEnchantChoiceParams{
				Itemid: itemID, Enchantid: id, Choice: chosen,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// ToggleOversizedWeapon marca ou desmarca a arma como feita para uma categoria de
// tamanho maior, e ajusta o espaço a mais que ela ocupa (p106, "mínimo meio espaço").
//
// Ela recebe o ITEM e não o estado desejado, como o `TogglePrepared`: quem sabe o que
// está lá é o servidor, e é ele que inverte.
func (p Plays) ToggleOversizedWeapon(ctx context.Context, itemID int64) error {
	return p.inTx(ctx, fmt.Sprintf("marcar a arma %d", itemID), func(q *sqlcgen.Queries) error {
		row, err := q.GetItem(ctx, itemID)
		if err != nil {
			return err
		}
		var catalog *book.Item
		if row.Catalogid.Valid {
			catalog = book.ItemByID(row.Catalogid.String)
		}
		if catalog == nil || catalog.Weapon == nil {
			return fmt.Errorf("%s não é uma arma do livro, e só arma tem categoria de tamanho", row.Name)
		}
		removed, err := q.UnmarkOversizedWeapon(ctx, itemID)
		if err != nil {
			return err
		}
		slots := row.Slots + 1
		if removed > 0 {
			slots = max(row.Slots-1, 0.5)
		} else if err := q.MarkOversizedWeapon(ctx, itemID); err != nil {
			return err
		}
		return q.SetItemSlots(ctx, sqlcgen.SetItemSlotsParams{Slots: slots, ID: itemID})
	})
}
