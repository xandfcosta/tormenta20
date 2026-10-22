package sheetui

import (
	"strconv"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A TIRA DE EQUIPADOS: as duas mãos e os quatro vestidos, como POSIÇÕES.
//
// Ela sai do `bag.go` porque tem razão própria para mudar, e a razão é o teto:
// os dois limites da p141 são da tira e de mais nada — a carga, o dinheiro e a
// grade do guardado não sabem o que é uma mão. Quem decide QUANTAS posições
// existem, como elas se chamam e o que acontece com o que passa do teto mora
// todo aqui.

// equippedPosition é uma posição das tiras de MÃOS e de VESTIDOS: o cartão de
// um item, ou o vazio com o nome do lugar que ainda cabe.
//
// UMA LISTA, e nunca um campo por lugar. O servidor recusa passar dos tetos da
// p141, mas linha antiga não passou por essa recusa, e o painel é a última
// coisa que pode denunciá-la. Enquanto as mãos eram "ou um item de duas, ou os
// de uma", um escudo em `wielded` ao lado de uma espada em `wielded2` sumia da
// tela INTEIRA — nem na tira, nem na grade do guardado, porque ele está
// equipado — e a Defesa continuava somando o +2 dele (ALE-450).
type equippedPosition struct {
	// Card é nil na posição vazia, e aí o `Label` é o nome do lugar livre.
	Card  *equippedCard
	Label string
	// Wide: o item de duas mãos ocupa as duas colunas da grade porque ocupa as
	// duas mãos. Vestido nunca é largo.
	Wide bool
}

// equippedCard é um item na tira, com o que ele concede.
type equippedCard struct {
	ID   int64
	Name string
	// Chips são as sobreposições e o que o item concede — "Reforçada",
	// "Defesa +2", "Dano 1d8". Um deles pode vir INATIVO: o item concede, mas
	// não a este portador (ALE-384).
	Chips []itemChip
	// NoProficiency marca o item equipado que o personagem não sabe usar. O
	// motor já cobra a penalidade (p142); o crachá é para ela não ser uma
	// surpresa no meio de um teste.
	NoProficiency bool
	Command       string
}

// equippedSlotOf lê o estado de equipar, tratando nulo como guardado.
func equippedSlotOf(item sheet.ItemDTO) string {
	if item.Equipped == nil {
		return ""
	}
	return *item.Equipped
}

// handPositionsOf põe os itens empunhados na ordem em que ocupam as mãos,
// completa com as que ainda cabem, e devolve quantas foram ocupadas.
//
// A CONTA É EM MÃOS E NÃO EM CARTÕES, e é daí que sai o "3/2" que denuncia o
// estouro: `wielded` ocupa uma e `wielded2` ocupa as duas (`sheet.HandsFor`,
// p141). Contar cartões devolveria 2/2 para um escudo ao lado de uma espada de
// duas mãos — o painel escondendo o estouro com a mesma eficácia com que
// escondia o item.
func handPositionsOf(
	items []sheet.ItemDTO, card func(sheet.ItemDTO) *equippedCard,
) ([]equippedPosition, int) {
	positions, taken := []equippedPosition{}, 0
	for _, item := range items {
		hands := sheet.HandsFor(equippedSlotOf(item))
		positions = append(positions, equippedPosition{
			Card: card(item), Label: handLabel(taken, hands), Wide: hands == 2,
		})
		taken += hands
	}
	for free := taken; free < sheet.HandsLimit; free++ {
		positions = append(positions, equippedPosition{Label: handLabel(free, 1)})
	}
	return positions, taken
}

// handLabel nomeia a posição pelas mãos JÁ OCUPADAS, e não pelo índice do
// cartão: um item de duas mãos depois de um de uma mão não é "a segunda
// posição" — é o que passou do teto.
//
// O que passa não tem nome no livro porque não deveria existir, e é justamente
// por isso que ele é nomeado do jeito que denuncia.
func handLabel(taken, hands int) string {
	switch {
	case taken+hands > sheet.HandsLimit:
		return "Acima do limite"
	case hands == 2:
		return "Duas mãos"
	case taken == 0:
		return "Mão principal"
	default:
		return "Mão secundária"
	}
}

// vestedPositionsOf é o mesmo para o corpo, e sem cortar no teto.
//
// As quatro posições aparecem sempre, mesmo com um item só: o teto do livro é
// a informação, e uma tira que crescesse com o uso esconderia quanto ainda
// cabe. O que passa do teto entra DEPOIS delas, nomeado.
func vestedPositionsOf(
	items []sheet.ItemDTO, card func(sheet.ItemDTO) *equippedCard,
) ([]equippedPosition, int) {
	positions := []equippedPosition{}
	for i, item := range items {
		label := "Vestido"
		if i >= sheet.VestedLimit {
			label = "Acima do limite"
		}
		positions = append(positions, equippedPosition{Card: card(item), Label: label})
	}
	for free := len(items); free < sheet.VestedLimit; free++ {
		positions = append(positions, equippedPosition{Label: "Vestido"})
	}
	return positions, len(items)
}

// equippedCardOf traduz um item equipado para o cartão da tira.
//
// O RÓTULO DO LUGAR não entra aqui: ele é da POSIÇÃO, e não do item. Enquanto
// o cartão o carregava, o mesmo nome existia em dois campos — o da posição
// vazia e o do cartão —, e dois campos para um conceito divergem.
func equippedCardOf(item sheet.ItemDTO, proficiencies map[string]bool, exempt wearerExemptions) *equippedCard {
	overlays := []itemChip{}
	for _, name := range itemOverlays(item) {
		overlays = append(overlays, itemChip{Text: name})
	}
	return &equippedCard{
		ID:            item.ID,
		Name:          item.Name,
		Chips:         append(overlays, thatGrantsItem(item, exempt)...),
		NoProficiency: !proficienteEh(item, proficiencies),
		Command:       strconv.FormatInt(item.ID, 10),
	}
}

// proficienteEh diz se o personagem sabe usar o item.
//
// A tabela que decide é a do MOTOR (`engine.RequiredProficiency`), a mesma que
// resolve a penalidade da p142: um segundo mapa aqui daria uma tela que avisa
// sobre um item e um motor que penaliza outro. Item custom e item fora do
// catálogo contam como proficientes — não há categoria de onde tirar exigência,
// e acusar o que não se sabe seria pior que calar.
func proficienteEh(item sheet.ItemDTO, proficiencies map[string]bool) bool {
	catalog := catalogItem(item)
	if catalog == nil {
		return true
	}
	required := engine.RequiredProficiency(&engine.CatalogItem{Category: catalog.Category})
	return required == "" || proficiencies[required]
}
