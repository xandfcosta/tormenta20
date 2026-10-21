package book

import (
	"strings"

	"t20engine/domain/engine"
)

// A BUSCA de item no catálogo, por id e por nome.
//
// Elas ficaram para trás quando o catálogo tipado saiu (ALE-278, segunda
// camada) porque moravam num arquivo da MOCHILA, e só apareceram quando a forja
// tentou sair: ela precisa achar o item que o kit inicial nomeia.

// ItemByID é a busca por id no acervo já ordenado.
func ItemByID(id string) *Item {
	for i, entry := range Catalogs().Items {
		if entry.ID == id {
			return &Catalogs().Items[i]
		}
	}
	return nil
}

// ItemByName é a busca por NOME, sem diferenciar maiúsculas.
//
// Existe porque nem toda procedência cita o item por id: a linha "Itens" de uma
// origem cita "Símbolo sagrado" por escrito (p85), e é o nome que tem de achar
// a entrada do livro para a linha nascer com o preço e os espaços certos.
func ItemByName(name string) *Item {
	sought := strings.ToLower(strings.TrimSpace(name))
	for i, entry := range Catalogs().Items {
		if strings.ToLower(entry.Name) == sought {
			return &Catalogs().Items[i]
		}
	}
	return nil
}

// expandComposedItems resolve os verbetes que vêm COMPOSTOS (p337, p340).
//
// O item específico não declara dado de arma, espaços nem eixo de equipar: ele
// É um item base com sobreposições, e tudo isso se herda. Os modificadores são
// os da base, mais os das peças com a procedência de cada uma, mais os próprios
// — a mesma conta que o `engine.Catalogs` faz do lado do motor, com a MESMA
// função, porque duas cópias divergiriam na primeira correção.
//
// Peça que não existe é PULADA aqui, e o silêncio tem dono: a primagem do motor
// recusa a mesma composição com erro nomeado, e o processo não sobe com ela. O
// que esta camada protege é a Mesa abrir — catálogo torto aqui tira uma aba,
// não o servidor.
func expandComposedItems(items []Item) {
	byID := make(map[string]*Item, len(items))
	for i := range items {
		byID[items[i].ID] = &items[i]
	}
	for i := range items {
		composed := items[i].ComposedOf
		if composed == nil {
			continue
		}
		base, ok := byID[composed.Base]
		if !ok || base.ComposedOf != nil {
			continue
		}
		mods := append([]engine.Modifier{}, base.Modifiers...)
		for _, id := range composed.Overlays() {
			piece, ok := byID[id]
			if !ok {
				continue
			}
			mods = append(mods, engine.OverlayModifiers(items[i].Category, piece.Name, piece.Modifiers)...)
		}
		items[i].Modifiers = append(mods, items[i].Modifiers...)
		inheritShape(&items[i], base)
	}
}

// inheritShape copia da base o que o verbete específico não repete: a FORMA do
// item, que é a mesma do item mundano que ele encantou.
func inheritShape(item *Item, base *Item) {
	if item.Weapon == nil {
		item.Weapon = base.Weapon
	}
	if item.Armor == nil {
		item.Armor = base.Armor
	}
	if item.Shield == nil {
		item.Shield = base.Shield
	}
	if item.Equip == "" {
		item.Equip = base.Equip
	}
	if item.Hands == 0 {
		item.Hands = base.Hands
	}
	if item.Slots == 0 {
		item.Slots = base.Slots
	}
}
