package sheetui

import (
	"sort"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/search"
	"t20engine/domain/sheet"
)

// A aba MOCHILA como dado.
//
// Três blocos, e a ordem responde a três perguntas diferentes da mesa:
//
//  1. O QUE ESTÁ NA MÃO E NO CORPO — a tira de Mãos (≤2) e Vestidos (≤4). Não
//     são casas de corpo: o livro não tem slots de armadura e de elmo, ele tem
//     dois TETOS (p141), e a tira desenha exatamente esses dois.
//  2. QUANTO PESA — a carga da p141, que vem inteira do motor.
//  3. O QUE ESTÁ GUARDADO — a grade de ladrilhos, com busca e categoria.
//
// # Nenhum número desta aba é somado aqui
//
// Espaços ocupados, limite, dobro do limite, sobrecarga e as duas penalidades
// são `sheet.Carga`. O que este arquivo faz é escolher palavras e agrupar
// linhas.

// bagPanel é a aba Mochila pronta para desenhar.
type bagPanel struct {
	// Hands são as posições das mãos, na ordem em que os itens as ocupam. Uma
	// arma de duas mãos é um cartão LARGO em vez de aparecer repetida nos dois
	// lugares.
	Hands     []equippedPosition
	HandsUsed int
	// Vested são as posições do corpo, com as quatro do teto desenhadas mesmo
	// vazias — ver quantas sobram é metade da informação.
	Vested     []equippedPosition
	VestedUsed int
	// Stowed é o que está guardado, já filtrado pela busca e pela categoria.
	Stowed []stowedTile
	// StowedTotal é o total ANTES do filtro, para a tela distinguir "mochila
	// vazia" de "nenhum item para esse filtro".
	StowedTotal int
	Load        loadMeter
	Money       moneyLine
	Search      string
	Category    string
	Categories  []filterOption
	// Sheets é uma ficha por item — o diálogo que o ladrilho e o cartão abrem.
	Sheets []itemSheet
	// Catalog é o Capítulo 3 filtrado, para o diálogo de adicionar.
	Catalog           []catalogItemRow
	CatalogSearch     string
	CatalogCategory   string
	CatalogCategories []filterOption
	CatalogTotal      int
}

// stowedTile é um ladrilho da grade.
type stowedTile struct {
	ID       int64
	Name     string
	Quantity int64
	// Glyph é o nome do ícone da casa, escolhido pela categoria.
	Glyph    string
	Overlays []string
	Command  string
}

// loadMeter é a carga da p141 traduzida para a tela.
type loadMeter struct {
	Used  string
	Limit int
	Max   int
	// Percent é a largura da barra, presa em 100 — uma barra de 340% desenharia
	// para fora do painel.
	Percent             int
	Coins               string
	CoinSlots           float64
	Overloaded          bool
	OverMax             bool
	Enforced            bool
	ArmorPenalty        string
	DisplacementPenalty string
	// LimitLabel é a conta que produziu o limite, e não a notação: "limite 18 ·
	// 10 + 2×For +4" em vez de "10 + 2×|FOR|".
	LimitLabel string
}

// moneyLine é o dinheiro do personagem.
type moneyLine struct {
	Tibar string
	// Slots são os espaços que as moedas ocupam, escrito só quando há moeda
	// bastante para ocupar algum.
	Slots string
}

// bagFilters é o que a pessoa digitou: os dois filtros da grade e os dois do
// diálogo do catálogo. Uma struct e não quatro parâmetros — quatro strings em
// sequência é a assinatura em que se troca a ordem sem o compilador reclamar.
type bagFilters struct {
	Search          string
	Category        string
	CatalogSearch   string
	CatalogCategory string
}

// bagPanelOf monta a aba.
func (s Scene) bagPanelOf(dto sheet.CharacterDTO, filters bagFilters) bagPanel {
	search, category := filters.Search, filters.Category
	proficiencies := savedProficiencies(dto.Proficiencies)
	exempt := exemptionsOf(dto)
	panel := bagPanel{
		Search:            search,
		Category:          category,
		Categories:        bagCategoryOptions(category),
		Money:             moneyLineOf(dto),
		CatalogSearch:     filters.CatalogSearch,
		CatalogCategory:   filters.CatalogCategory,
		CatalogCategories: catalogCategories(filters.CatalogCategory),
	}
	panel.Catalog = catalogItemRowsOf(filters.CatalogSearch, filters.CatalogCategory)
	panel.CatalogTotal = len(panel.Catalog)
	wielded, vested, saved := []sheet.ItemDTO{}, []sheet.ItemDTO{}, []sheet.ItemDTO{}
	for _, item := range dto.Items {
		switch equippedSlotOf(item) {
		case "wielded", "wielded2":
			wielded = append(wielded, item)
		case "vested":
			vested = append(vested, item)
		default:
			saved = append(saved, item)
		}
	}
	card := func(item sheet.ItemDTO) *equippedCard {
		return equippedCardOf(item, proficiencies, exempt)
	}
	panel.Hands, panel.HandsUsed = handPositionsOf(wielded, card)
	panel.Vested, panel.VestedUsed = vestedPositionsOf(vested, card)
	panel.StowedTotal = len(saved)
	panel.Stowed = stowedTilesOf(bagFiltered(saved, search, category))
	panel.Sheets = itemSheetsOf(dto, proficiencies)
	if sheet, _, ok := s.sheetForPanels(dto); ok {
		panel.Load = loadMeterOf(sheet)
	}
	return panel
}

// stowedTilesOf traduz os itens guardados em ladrilhos.
func stowedTilesOf(items []sheet.ItemDTO) []stowedTile {
	rows := make([]stowedTile, 0, len(items))
	for _, item := range items {
		rows = append(rows, stowedTile{
			ID: item.ID, Name: item.Name, Quantity: item.Quantity,
			Glyph:    itemGlyph(item),
			Overlays: itemOverlays(item),
			Command:  strconv.FormatInt(item.ID, 10),
		})
	}
	return rows
}

// itemGlyph escolhe o desenho do ladrilho pela categoria.
//
// Item sem catálogo cai no pacote genérico, que é honesto: não há o que
// adivinhar sobre um item que a pessoa inventou.
func itemGlyph(item sheet.ItemDTO) string {
	catalog := catalogItem(item)
	if catalog == nil {
		return "Package"
	}
	switch {
	case strings.HasPrefix(catalog.Category, "weapon-"):
		return "Sword"
	case strings.HasPrefix(catalog.Category, "armor-"), catalog.Category == "shield":
		return "Shield"
	case catalog.Category == "apparel" && catalog.Equip == "wielded":
		return "Wand2"
	case catalog.Category == "apparel":
		return "Shirt"
	case catalog.Category == "consumable":
		return "FlaskConical"
	case catalog.Category == "meal":
		return "Utensils"
	}
	return "Package"
}

// bagFiltered cruza a busca com a categoria escolhida.
//
// A busca ignora acento pela mesma razão das Perícias: quem digita "balsamo"
// tem de achar "Bálsamo restaurador".
func bagFiltered(items []sheet.ItemDTO, query, category string) []sheet.ItemDTO {
	term := search.Fold(strings.TrimSpace(query))
	outside := []sheet.ItemDTO{}
	for _, item := range items {
		if term != "" && !strings.Contains(search.Fold(item.Name), term) {
			continue
		}
		if !categoryBagDa(item, category) {
			continue
		}
		outside = append(outside, item)
	}
	return outside
}

// bagCategories são os chips, na ordem em que aparecem.
//
// São CINCO e não as quinze do catálogo: o chip existe para achar uma coisa no
// meio de trinta ladrilhos, e quinze chips numa tela de 390px seriam outra
// lista para procurar dentro.
var bagCategories = []filterOption{
	{Value: "", Label: "tudo"},
	{Value: "weapons", Label: "armas"},
	{Value: "defense", Label: "defesa"},
	{Value: "consumables", Label: "consumo"},
	{Value: "other", Label: "outros"},
}

func bagCategoryOptions(active string) []filterOption {
	outside := make([]filterOption, 0, len(bagCategories))
	for _, badge := range bagCategories {
		outside = append(outside, filterOption{Value: badge.Value, Label: badge.Label, Active: badge.Value == active})
	}
	return outside
}

// categoryBagDa diz se o item aparece sob o crachá escolhido.
//
// Item custom não tem categoria de catálogo e conta como equipamento comum —
// assim ele cai em "tudo" e em "outros", e nunca some da mochila inteira.
func categoryBagDa(item sheet.ItemDTO, chip string) bool {
	if chip == "" {
		return true
	}
	category := "gear"
	if catalog := catalogItem(item); catalog != nil {
		category = catalog.Category
	}
	switch chip {
	case "weapons":
		return strings.HasPrefix(category, "weapon-")
	case "defense":
		return strings.HasPrefix(category, "armor-") || category == "shield"
	case "consumables":
		return category == "consumable" || category == "meal"
	case "other":
		return !strings.HasPrefix(category, "weapon-") &&
			!strings.HasPrefix(category, "armor-") &&
			category != "shield" && category != "consumable" && category != "meal"
	}
	return true
}

// loadMeterOf traduz a carga do motor.
//
// O parâmetro se chamava `sheet` e SOMBREAVA o pacote de mesmo nome — o corpo
// não podia mais alcançar nada de `domain/sheet` sem que o compilador
// procurasse um método na struct.
func loadMeterOf(computed engine.ComputedSheet) loadMeter {
	load := computed.Load
	return loadMeter{
		Used:                sheet.WithComma(load.Used),
		Limit:               load.Limit,
		Max:                 load.Max,
		Percent:             barWidth(load.Used, load.Limit),
		Coins:               sheet.WithComma(load.Coins),
		CoinSlots:           load.Coins,
		Overloaded:          load.Overloaded,
		OverMax:             load.OverMax,
		Enforced:            load.Enforced,
		ArmorPenalty:        book.WithSign(load.ArmorPenalty),
		DisplacementPenalty: book.WithSign(load.DisplacementPenalty),
		LimitLabel:          limitLabel(load.Limit, computed.Attributes["strength"].Total),
	}
}

// barWidth é a porcentagem já presa em 100.
func barWidth(used float64, limit int) int {
	if limit <= 0 {
		return 0
	}
	percent := int(used * 100 / float64(limit))
	if percent > 100 {
		return 100
	}
	return percent
}

// limitLabel mostra a CONTA que produziu o limite, com o valor de Força
// resolvido — e não a notação "10 + 2×|FOR|", que manda a pessoa fazer a conta
// de cabeça para conferir o número que já está do lado.
func limitLabel(limit, strength int) string {
	return "limite " + strconv.Itoa(limit) + " · 10 + 2×For " + book.WithSign(strength)
}

// moneyLineOf escreve o dinheiro e o espaço que ele ocupa.
func moneyLineOf(dto sheet.CharacterDTO) moneyLine {
	row := moneyLine{Tibar: sheet.WithComma(dto.Tibar)}
	if spaces := coinSlots(dto.Tibar); spaces > 0 {
		row.Slots = sheet.WithComma(spaces) + slotPlural(spaces)
	}
	return row
}

// coinSlots são os milheiros COMPLETOS: "cada 1.000 moedas ocupam um
// espaço" (p141), e 1.999 T$ ocupam um espaço e não dois.
func coinSlots(tibar float64) float64 {
	return float64(int(tibar) / int(engine.CoinsPerSlot))
}

func slotPlural(spaces float64) string {
	if spaces == 1 {
		return " espaço"
	}
	return " espaços"
}

// sortedImprovements devolve as sobreposições do item ordenadas por nome,
// para a ficha do item listá-las sempre na mesma ordem.
func sortedImprovements(item sheet.ItemDTO) []book.Item {
	entries := bookOverlays(item)
	sort.SliceStable(entries, func(a, b int) bool { return entries[a].Name < entries[b].Name })
	return entries
}

// ── o que a TELA precisa escrever ────────────────────────────────────────────

// writtenItems é "3 itens", com o singular certo.
func writtenItems(n int) string {
	if n == 1 {
		return "1 item"
	}
	return strconv.Itoa(n) + " itens"
}

// overloadNotice diz o que a sobrecarga CUSTA, com os dois números que o
// motor já aplicou — e não uma frase decorada que pode divergir deles.
//
// A segunda metade só aparece acima do DOBRO do limite, que é o outro teto da
// p141: "você não pode carregar mais do que o dobro do seu limite". O motor não
// recusa a linha, porque o próprio livro deixa a carga a critério do mestre —
// então quem diz que passou é a tela.
func overloadNotice(load loadMeter) string {
	notice := "Sobrecarregado (p141): " + load.ArmorPenalty +
		" em Acrobacia, Furtividade e Ladinagem · " + load.DisplacementPenalty + "m de deslocamento"
	if load.OverMax {
		notice += " · acima de " + strconv.Itoa(load.Max) + " espaços o livro diz que não dá para carregar"
	}
	return notice
}

// moneyModes são as três coisas que se fazem com dinheiro na mesa: "achamos 350
// no baú", "paguei 80 pela estalagem", e escrever o total — que é o gesto da
// forja (Tabela 3-1, p140) e o de consertar um erro de digitação.
var moneyModes = []filterOption{
	{Value: "receber", Label: "Receber"},
	{Value: "gastar", Label: "Gastar"},
	{Value: "corrigir", Label: "Corrigir"},
}
