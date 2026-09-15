package sheetui

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"t20engine/domain/book"
	"t20engine/domain/sheet"

	"github.com/go-chi/chi/v5"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// OS COMANDOS DA ABA MOCHILA.

// addCatalogItem põe na mochila um item do Capítulo 3.
//
// O NOME e os ESPAÇOS vêm do catálogo, e não do cliente: são dado transcrito do
// livro, e deixar o navegador mandá-los abriria a porta para uma "Espada longa"
// de 0 espaços.
func addCatalogItem(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	catalogo := book.ItemByID(chi.URLParam(r, "catalogo"))
	if catalogo == nil {
		return fmt.Errorf("o item %q não existe no livro", chi.URLParam(r, "catalogo"))
	}
	quantidade, err := askedQuantity(sinais)
	if err != nil {
		return err
	}
	_, err = s.deps.Queries().CreateItem(r.Context(), sqlcgen.CreateItemParams{
		Characterid: row.ID, Catalogid: sql.NullString{String: catalogo.ID, Valid: true},
		Name: catalogo.Name, Quantity: quantidade, Slots: catalogo.Slots,
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	})
	return err
}

// addCustomItem cria o item que o livro não tem — a lembrança de um NPC, a
// chave de um cofre.
func addCustomItem(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	nome, quantidade, espacos, err := customRequestItem(sinais)
	if err != nil {
		return err
	}
	_, err = s.deps.Queries().CreateItem(r.Context(), sqlcgen.CreateItemParams{
		Characterid: row.ID, Name: nome, Quantity: quantidade, Slots: espacos,
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	})
	return err
}

// editItem muda nome, quantidade e espaços de um item já na ficha.
func editItem(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	nome, quantidade, espacos, err := customRequestItem(sinais)
	if err != nil {
		return err
	}
	// A gravação é uma PERGUNTA e não um SQL montado aqui: quem sabe o nome das
	// colunas é o hospedeiro. Mesma decisão do `SaveText` da cena de campanhas.
	return s.deps.SaveCustomItem(r.Context(), item.ID, nome, quantidade, espacos)
}

// removeItemFromSheet tira o item da ficha.
func removeItemFromSheet(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return s.deps.Queries().DeleteItem(r.Context(), item.ID)
}

// useItem gasta uma dose do consumível.
//
// A regra inteira — a rolagem presa no máximo, a linha de efeito de cena ou dia,
// a porção diária, a baixa do item — mora no `ConsumeItem`. Os números rolados
// vêm por sinal porque quem rola é a MESA: a ficha não rola dado por ninguém.
func useItem(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return s.deps.ConsumeItem(r, row, item.ID, sinais.ItemRolagemPv, sinais.ItemRolagemPm)
}

// applyOverlays grava as melhorias e o material escolhidos.
//
// A COMPATIBILIDADE é conferida AQUI (`fitsItemImprovement`), no servidor, e não
// só pelo filtro do diálogo: filtro é UX, e quem postar na mão passa por cima.
func applyOverlays(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	catalogo := book.ItemByID(itemCatalog(item))
	if err := fitsItemImprovement(catalogo, sinais.ItemMelhorias, "improvement"); err != nil {
		return err
	}
	materiais := []string{}
	if sinais.ItemMaterial != "" {
		materiais = append(materiais, sinais.ItemMaterial)
	}
	if err := fitsItemImprovement(catalogo, materiais, "material"); err != nil {
		return err
	}
	return s.deps.SaveItemOverlays(r.Context(), item.ID, sinais.ItemMelhorias, sinais.ItemMaterial)
}

// askedQuantity lê a quantidade, com as bordas do formulário.
func askedQuantity(sinais Signals) (int64, error) {
	if sinais.ItemQtd == nil {
		return 1, nil
	}
	if *sinais.ItemQtd < 1 || *sinais.ItemQtd > 9999 {
		return 0, fmt.Errorf("a quantidade %d está fora de 1 a 9999", *sinais.ItemQtd)
	}
	return *sinais.ItemQtd, nil
}

// customRequestItem lê nome, quantidade e espaços, com as bordas do formulário.
//
// Os ESPAÇOS são múltiplos de meio porque é assim que o livro conta carga
// (p141) — e essa é a mesma borda que a API JSON cobra, `sheet.SlotsNotMultiple`.
func customRequestItem(sinais Signals) (string, int64, float64, error) {
	nome := ""
	if sinais.ItemNome != nil {
		nome = strings.TrimSpace(*sinais.ItemNome)
	}
	if nome == "" {
		return "", 0, 0, fmt.Errorf("informe um nome para o item")
	}
	if len([]rune(nome)) > 80 {
		return "", 0, 0, fmt.Errorf("o nome tem %d letras, e o máximo são 80", len([]rune(nome)))
	}
	quantidade, err := askedQuantity(sinais)
	if err != nil {
		return "", 0, 0, err
	}
	espacos := 1.0
	if sinais.ItemEspacos != nil {
		espacos = *sinais.ItemEspacos
	}
	if espacos < 0 || sheet.SlotsNotMultiple(espacos) {
		return "", 0, 0, fmt.Errorf("os espaços (%v) têm de ser múltiplos de 0,5", espacos)
	}
	return nome, quantidade, espacos, nil
}

// stowItem tira o item da mão ou do corpo e o devolve à mochila.
//
// Guardar nunca esbarra em teto — ele só LIBERA espaço —, então este comando
// não passa pelas checagens de eixo e de limite que o `equipItemFromSheet` faz.
func stowItem(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return saveEquipped(r, s, item.ID, sql.NullString{})
}

// equipItemFromSheet põe o item na mão ou no corpo.
//
// As DUAS recusas moram no `sheet`, e pela razão de sempre: o eixo do item
// (`sheet.EquipAxisError` — um escudo não se veste) e os tetos de 2 mãos e 4
// vestidos (`sheet.EquipLimitErrorOver`, p141). Escrevê-las de novo aqui daria
// duas regras para a mesma pergunta.
func equipItemFromSheet(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	slot := chi.URLParam(r, "slot")
	if !slotEquipEh(slot) {
		return fmt.Errorf("%q não é um lugar de equipar", slot)
	}
	// O EIXO sai do catálogo EMBUTIDO, e não do `s.deps.Catalogs()`.
	//
	// Os dois trazem o mesmo `items.json`, mas o `s.deps.Catalogs()` é primado
	// de um arquivo por caminho de configuração, e o `primeCatalogs` diz o que
	// acontece quando ele falta: "mutation validators disabled". Uma regra que
	// se DESLIGA sozinha quando um arquivo some não é uma regra — com o
	// catálogo vazio, um escudo passa a ser vestível. O `catalog.Resource` é
	// `go:embed`: ele existe sempre que o binário existe.
	if _, recusa := sheet.EquipAxisError(howEngineItem(book.ItemByID(itemCatalog(item))), slot); recusa != "" {
		return fmt.Errorf("%s", recusa)
	}
	equipados, err := s.deps.Queries().ListEquippedItems(r.Context(), row.ID)
	if err != nil {
		return err
	}
	if recusa := sheet.EquipLimitErrorOver(equipados, item.ID, slot); recusa != "" {
		return fmt.Errorf("%s", recusa)
	}
	return saveEquipped(r, s, item.ID, sql.NullString{String: slot, Valid: true})
}

// saveEquipped escreve a coluna `equipped`.
//
// Ela é uma PERGUNTA na porta, e o detalhe é do hospedeiro: `character_items`
// não tem `updatedAt`, então a gravação não toca carimbo nenhum. A cena não
// precisa saber disso — ela sabe que o item foi para a mão.
func saveEquipped(r *http.Request, s Scene, itemID int64, valor sql.NullString) error {
	return s.deps.SaveEquipped(r.Context(), itemID, valor)
}

// slotEquipEh aceita só os três lugares do livro.
func slotEquipEh(slot string) bool {
	return slot == "vested" || slot == "wielded" || slot == "wielded2"
}

// sheetItem lê o item do caminho e CONFERE que ele é desta ficha.
//
// A posse do personagem já foi conferida pelo `sheetCommand`; o que falta é a
// do item, e sem ela um id de outra ficha passaria — a consulta é por id e o
// `characterId` só entraria no `UPDATE`, que não acusaria nada por afetar zero
// linhas.
func (s Scene) sheetItem(r *http.Request, characterID int64) (sqlcgen.GetItemRow, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "item"), 10, 64)
	if err != nil {
		return sqlcgen.GetItemRow{}, fmt.Errorf("o item %q não é um número", chi.URLParam(r, "item"))
	}
	item, err := s.deps.Queries().GetItem(r.Context(), id)
	if err != nil || item.Characterid != characterID {
		return sqlcgen.GetItemRow{}, fmt.Errorf("o item %d não é desta ficha", id)
	}
	return item, nil
}

// changeMoney recebe, gasta ou corrige o dinheiro.
//
// Os TRÊS modos existem porque são três gestos diferentes na mesa: "achamos 350
// no baú", "paguei 80 pela estalagem", e escrever o total — que é o gesto da
// forja (Tabela 3-1, p140) e o de consertar um erro de digitação.
func changeMoney(s Scene, r *http.Request, row sqlcgen.Character, sinais Signals) error {
	if sinais.TibarValor == nil {
		return fmt.Errorf("informe um valor a partir de 0")
	}
	saldo, erro := afterGestureBalance(row.Tibar, sinais.TibarModo, *sinais.TibarValor)
	if erro != "" {
		return fmt.Errorf("%s", erro)
	}
	return s.deps.Queries().SetCharacterTibar(r.Context(), sqlcgen.SetCharacterTibarParams{
		Tibar: saldo, UpdatedAt: dbvalue.NowISO(), ID: row.ID,
	})
}
