package sheetui

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"t20engine/domain/book"
	"t20engine/domain/sheet"

	"github.com/go-chi/chi/v5"

	"t20engine/infra/db/sqlcgen"
)

// OS COMANDOS DA ABA MOCHILA.

// refusePastTheDouble recusa o item que levaria a carga além do dobro do limite (p141).
//
// Só os gestos de item passam por aqui. O dinheiro também é carga, mas o mestre entrega o
// tesouro na voz, e recusá-lo deixaria a ficha mentindo sobre o que a mesa decidiu; a
// mochila mostra o excesso no aviso. A pergunta usa a mesma ficha da aba, para a recusa
// e a barra não discordarem do limite.
func (s Scene) refusePastTheDouble(r *http.Request, row sqlcgen.Character, added float64) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	return s.refusesPastTheDoubleOf(dto, added)
}

// refusesPastTheDoubleOf é a mesma recusa para quem já carregou a ficha.
func (s Scene) refusesPastTheDoubleOf(dto sheet.CharacterDTO, added float64) error {
	computed, _, ok := s.sheetForPanels(dto)
	if !ok || computed.Load.FitsAdding(added) {
		return nil
	}
	return fmt.Errorf("não cabe na mochila (p141): ficaria com %s espaços, e o máximo é o dobro do limite, %d",
		sheet.WithComma(computed.Load.Used+added), computed.Load.Max)
}

// addCatalogItem põe na mochila um item do Capítulo 3.
//
// O NOME e os ESPAÇOS vêm do catálogo, e não do cliente: são dado transcrito do
// livro, e deixar o navegador mandá-los abriria a porta para uma "Espada longa"
// de 0 espaços.
func addCatalogItem(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	catalog := book.ItemByID(chi.URLParam(r, "catalogo"))
	if catalog == nil {
		return fmt.Errorf("o item %q não existe no livro", chi.URLParam(r, "catalogo"))
	}
	// Sempre um por gesto: a quantidade só muda no Editar do item (decisão do dono).
	if err := s.refusePastTheDouble(r, row, catalog.Slots); err != nil {
		return err
	}
	return s.plays.AddCatalogItem(r.Context(), row.ID, catalog.ID)
}

// addCustomItem cria o item que o livro não tem — a lembrança de um NPC, a
// chave de um cofre.
func addCustomItem(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	name, amount, spaces, err := customRequestItem(signals)
	if err != nil {
		return err
	}
	if err := s.refusePastTheDouble(r, row, float64(amount)*spaces); err != nil {
		return err
	}
	return s.plays.AddCustomItem(r.Context(), row.ID, name, amount, spaces)
}

// editItem muda nome, quantidade e espaços de um item já na ficha.
func editItem(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	name, amount, spaces, err := customRequestItem(signals)
	if err != nil {
		return err
	}
	// A gravação é do CASO DE USO e não um SQL montado aqui: quem sabe o nome
	// das colunas é ele. Mesma decisão que o `campaign.Lifecycle` tomou com o
	// texto da campanha.
	added := float64(amount)*spaces - float64(item.Quantity)*item.Slots
	if err := s.refusePastTheDouble(r, row, added); err != nil {
		return err
	}
	return s.plays.SaveCustomItem(r.Context(), item.ID, name, amount, spaces)
}

// removeItemFromSheet tira o item da ficha.
func removeItemFromSheet(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	return s.itemGesture(r, row, withoutItem(item.ID), func() error { return s.plays.RemoveItem(r.Context(), item.ID) })
}

// itemGesture faz um gesto que muda o que a ficha carrega, com a recusa das perícias
// escolhidas e as condições que o PV pede, sobre UMA carga só: as duas perguntas leem a
// mesma simulação. O item pode dar PV (os anéis, p342) ou Constituição, então guardá-lo
// com pouca vida pode derrubar.
func (s Scene) itemGesture(
	r *http.Request, row sqlcgen.Character, change func(*sheet.CharacterDTO), persist func() error,
) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	if err := keepsChosenExpertisesOf(dto, row, change); err != nil {
		return err
	}
	projected := dto
	projected.Items = slices.Clone(dto.Items)
	change(&projected)
	if err := persist(); err != nil {
		return err
	}
	return s.plays.FollowMaximumChange(r.Context(), dto, projected)
}

// useItem gasta uma unidade do consumível.
//
// A regra inteira — a rolagem presa no máximo, a linha de efeito de cena ou dia,
// a porção diária, a baixa do item — mora no `Plays.Consume`. Os números rolados
// vêm por sinal porque quem rola é a MESA: a ficha não rola dado por ninguém.
func useItem(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	// O RESULTADO não atravessa: a cena redesenha a ficha inteira depois do
	// gesto, e a única recusa que ela precisa — a porção diária — já chega como
	// erro.
	_, err = s.plays.Consume(r.Context(), row, item.ID, signals.ItemHPRoll, signals.ItemMPRoll)
	return err
}

// applyOverlays grava as melhorias e o material escolhidos.
//
// A COMPATIBILIDADE é conferida AQUI (`fitsItemImprovement`), no servidor, e não
// só pelo filtro do diálogo: filtro é UX, e quem postar na mão passa por cima.
func applyOverlays(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	catalog := book.ItemByID(itemCatalog(item))
	if twice := repeatedImprovement(signals.ItemImprovements); twice != "" {
		return fmt.Errorf("a melhoria %q veio duas vezes: cada melhoria entra uma vez no item", twice)
	}
	fromBook, fromTable := splitImprovementIDs(signals.ItemImprovements)
	if err := fitsItemImprovement(catalog, fromBook, "improvement"); err != nil {
		return err
	}
	if err := tableImprovementsFit(s, r, row, item, catalog, fromTable); err != nil {
		return err
	}
	materials := []string{}
	if signals.ItemMaterial != "" {
		materials = append(materials, signals.ItemMaterial)
	}
	if err := fitsItemImprovement(catalog, materials, "material"); err != nil {
		return err
	}
	// O ENCANTO NÃO PASSA POR AQUI, e a ausência é a regra: quem encanta uma
	// arma é o MESTRE, e a escrita dele é o `characterPlays.EnchantItem`. Sem
	// um `ItemEnchants` nos sinais, um pedido montado à mão não tem por onde
	// entrar — a fronteira é a ausência do campo, não um `if` (ALE-416).
	// Pelo `itemGesture`: a melhoria pode dar PV, e tirá-la com pouca vida derruba como guardar o item.
	withOverlays := func(dto *sheet.CharacterDTO) {
		for i := range dto.Items {
			if dto.Items[i].ID == item.ID {
				dto.Items[i].Improvements = sheet.MarshalStrings(&signals.ItemImprovements)
				dto.Items[i].Material = nil
				if signals.ItemMaterial != "" {
					dto.Items[i].Material = &signals.ItemMaterial
				}
			}
		}
	}
	return s.itemGesture(r, row, withOverlays, func() error {
		return s.plays.SaveItemOverlays(r.Context(), item.ID, signals.ItemImprovements, signals.ItemMaterial)
	})
}

// toggleOversizedWeapon liga ou desliga a marca de arma de categoria maior. Marcar
// passa pelo teto de carga, porque a arma ocupa um espaço a mais (p106).
func toggleOversizedWeapon(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	item, err := s.sheetItem(r, row.ID)
	if err != nil {
		return err
	}
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	marked := slices.ContainsFunc(dto.Items, func(i sheet.ItemDTO) bool { return i.ID == item.ID && i.Oversized })
	if !marked {
		if err := s.refusesPastTheDoubleOf(dto, float64(item.Quantity)); err != nil {
			return err
		}
	}
	return s.plays.ToggleOversizedWeapon(r.Context(), item.ID)
}

// askedQuantity lê a quantidade, com as bordas do formulário.
func askedQuantity(signals Signals) (int64, error) {
	if signals.ItemQtd == nil {
		return 1, nil
	}
	if *signals.ItemQtd < 1 || *signals.ItemQtd > sheet.MaxItemQuantity {
		return 0, fmt.Errorf("a quantidade %d está fora de 1 a %d", *signals.ItemQtd, sheet.MaxItemQuantity)
	}
	return *signals.ItemQtd, nil
}

// customRequestItem lê nome, quantidade e espaços, com as bordas do formulário.
//
// Os ESPAÇOS são múltiplos de meio porque é assim que o livro conta carga
// (p141) — e essa é a mesma borda que a API JSON cobra, `sheet.SlotsNotMultiple`.
func customRequestItem(signals Signals) (string, int64, float64, error) {
	name := ""
	if signals.ItemName != nil {
		name = strings.TrimSpace(*signals.ItemName)
	}
	if name == "" {
		return "", 0, 0, fmt.Errorf("informe um nome para o item")
	}
	if len([]rune(name)) > 80 {
		return "", 0, 0, fmt.Errorf("o nome tem %d letras, e o máximo são 80", len([]rune(name)))
	}
	amount, err := askedQuantity(signals)
	if err != nil {
		return "", 0, 0, err
	}
	spaces := 1.0
	if signals.ItemSlots != nil {
		spaces = *signals.ItemSlots
	}
	if spaces < 0 || sheet.SlotsNotMultiple(spaces) {
		return "", 0, 0, fmt.Errorf("os espaços (%v) têm de ser múltiplos de 0,5", spaces)
	}
	return name, amount, spaces, nil
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
	return saveEquipped(r, s, row, item.ID, "")
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
	// Os dois trazem o mesmo `items.json`, mas o `s.deps.Catalogs()` é um campo
	// que a bancada troca, e há fixture que o prima vazio. Uma regra que se
	// DESLIGA sozinha com o catálogo vazio não é uma regra — um escudo passaria
	// a ser vestível. O `catalog.Resource` é o livro compilado: ele existe
	// sempre que o binário existe.
	if _, refusal := sheet.EquipAxisError(howEngineItem(book.ItemByID(itemCatalog(item))), slot); refusal != "" {
		return fmt.Errorf("%s", refusal)
	}
	equipped, err := s.deps.Queries().ListEquippedItems(r.Context(), row.ID)
	if err != nil {
		return err
	}
	if refusal := sheet.EquipLimitErrorOver(equipped, item.ID, slot); refusal != "" {
		return fmt.Errorf("%s", refusal)
	}
	return saveEquipped(r, s, row, item.ID, slot)
}

// saveEquipped põe o item num lugar do corpo, e o VAZIO o guarda de volta.
//
// A cena diz o LUGAR e nada mais: que o vazio vira NULL, e que
// `character_items` não tem carimbo para tocar, é do caso de uso. Ela sabe que
// o item foi para a mão — e confere antes que a ficha não sai da escolha do
// Especialista, porque o item vestido conta no N (ALE-501).
func saveEquipped(r *http.Request, s Scene, row sqlcgen.Character, itemID int64, place string) error {
	var wear *string
	if place != "" {
		wear = &place
	}
	return s.itemGesture(r, row, equippedAs(itemID, wear), func() error { return s.plays.SaveEquipped(r.Context(), itemID, place) })
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
func changeMoney(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	if signals.TibarValue == nil {
		return fmt.Errorf("informe um valor a partir de 0")
	}
	return s.plays.ChangeMoney(r.Context(), row, signals.TibarMode, *signals.TibarValue)
}
