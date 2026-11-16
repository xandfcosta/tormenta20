package sheetui

import (
	"strconv"
	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// O QUE O LIVRO DIZ DO ITEM, na folha dele: a categoria, o preço, a página e os
// números do catálogo — o dano e o crítico da arma, a proteção, a magia do frasco.
// É REFERÊNCIA, e não a conta da ficha: o dano desta ficha com a arma na mão é o
// `InHand` da folha, que vem da carta do Combate.

// bookInfo é o que o livro diz do item.
type bookInfo struct {
	Category string
	Price    string
	Page     int
	Rows     []string
}

// thatSaysBook é o bloco de referência da ficha do item.
func thatSaysBook(catalog book.Item) *bookInfo {
	info := &bookInfo{
		Category: book.CategoryName(catalog.Category),
		Price:    sheet.WithComma(catalog.Price),
		Page:     catalog.BookPage,
	}
	if weapon := catalog.Weapon; weapon != nil {
		info.Rows = append(info.Rows, "dano "+weapon.Damage+" · crítico "+critWritten(weapon.CritRange, weapon.CritMult))
		if weapon.Type != "" {
			info.Rows = append(info.Rows, "tipo "+damageWrittenKind(weapon.Type))
		}
	}
	if protection := catalog.Armor; protection != nil {
		info.Rows = append(info.Rows, protectionRow(*protection, true))
	}
	if protection := catalog.Shield; protection != nil {
		info.Rows = append(info.Rows, protectionRow(*protection, false))
	}
	info.Rows = append(info.Rows, spellInTheFlask(catalog)...)
	for _, m := range catalog.Modifiers {
		info.Rows = append(info.Rows, modifierBadge(m))
	}
	info.Rows = repetidosSem(info.Rows)
	return info
}

// spellInTheFlask é a referência da poção: a magia que ela CONTÉM, o círculo
// dela e como o frasco entrega o efeito (p341).
//
// Sem isso o cartão de uma poção é um nome e um preço — ela não tem
// modificador para desenhar, porque a regra dela é a da magia. E o app INFORMA
// e não resolve: a área da Bola de Fogo e o alvo do Sono são do mestre; o que a
// ficha deve é dizer o que está no frasco.
func spellInTheFlask(catalog book.Item) []string {
	if catalog.Consumable == nil || catalog.Consumable.Spell == "" {
		return nil
	}
	magia := spellOfBook(catalog.Consumable.Spell)
	linha := "contém " + magia.Name
	if magia.Circle > 0 {
		linha += " · " + circleLabel(magia.Circle)
	}
	if entrega := writtenFlask(catalog.Consumable.Kind); entrega != "" {
		linha += " · " + entrega
	}
	rows := []string{linha}
	if augment := catalog.Consumable.Augment; augment != "" {
		rows = append(rows, augment)
	}
	return rows
}

// writtenFlask é a palavra da mesa para o tipo de frasco. O livro as nomeia na
// p341: "poções que afetam objetos também são chamadas de óleos e poções que
// geram efeito em área também são chamadas de granadas".
var flaskKinds = map[string]string{
	"pocao": "poção", "oleo": "óleo", "granada": "granada",
}

func writtenFlask(kind string) string {
	return flaskKinds[kind]
}

func protectionRow(protection book.Armor, isArmor bool) string {
	row := "Defesa " + book.WithSign(protection.Defense) + " · penalidade " + strconv.Itoa(protection.Penalty)
	if !isArmor {
		return row
	}
	if protection.Heavy {
		return row + " · pesada"
	}
	return row + " · leve"
}

// damageWrittenKinds é o pt-BR do tipo de dano.
//
// O catálogo guarda a CHAVE sem acento (`perfuracao`), como todo id deste
// projeto; sem esta tabela o id cru vaza para o jogador ("tipo perfuracao").
//
// Os ELEMENTAIS entraram com a redução de dano por tipo (p229): a mesma tabela
// serve às duas telas — a carta da arma diz de que tipo ela causa, e a Defesa
// diz contra que tipo o portador reduz. Duas tabelas divergiriam no acento.
var damageWrittenKinds = map[string]string{
	"corte": "corte", "perfuracao": "perfuração", "impacto": "impacto",
	"corte-perfuracao": "corte ou perfuração",
	"fogo":             "fogo", "frio": "frio", "acido": "ácido",
	"eletricidade": "eletricidade", "luz": "luz", "trevas": "trevas",
	"lefeu": "lefeu", "mental": "mental", "essencia": "essência",
}

func damageWrittenKind(kind string) string {
	if name, found := damageWrittenKinds[kind]; found {
		return name
	}
	return kind
}
