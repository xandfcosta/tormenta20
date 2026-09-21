package sheetui

import (
	"encoding/json"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// O CATÁLOGO DE ITENS lido pela Mochila.
//
// Tudo aqui responde a uma pergunta sobre uma linha da ficha contra o
// `items.json`: que entrada do livro é esta, o que ela concede, que melhorias
// ela está usando, e em que chip da grade ela cai.

// catalogItem acha a entrada do livro de um item da ficha.
func catalogItem(item sheet.ItemDTO) *book.Item {
	if item.CatalogID == nil || *item.CatalogID == "" {
		return nil
	}
	return book.ItemByID(*item.CatalogID)
}

// itemOverlays são os NOMES das melhorias e do material aplicados.
func itemOverlays(item sheet.ItemDTO) []string {
	names := []string{}
	for _, entry := range bookOverlays(item) {
		names = append(names, entry.Name)
	}
	return names
}

// bookOverlays resolve melhorias + material contra o catálogo.
//
// Id desconhecido é PULADO em vez de virar erro: a coluna guarda um blob de
// texto, e uma linha torta não pode impedir a mochila inteira de abrir.
func bookOverlays(item sheet.ItemDTO) []book.Item {
	ids := savedImprovements(item.Improvements)
	if item.Material != nil && *item.Material != "" {
		ids = append(ids, *item.Material)
	}
	outside := []book.Item{}
	for _, id := range ids {
		if entry := book.ItemByID(id); entry != nil {
			outside = append(outside, *entry)
		}
	}
	return outside
}

// savedImprovements lê o blob JSON da coluna `improvements`.
func savedImprovements(blob string) []string {
	var ids []string
	if json.Unmarshal([]byte(blob), &ids) != nil {
		return nil
	}
	return ids
}

// thatGrantsItem são os crachás do que o item dá enquanto equipado.
//
// A Defesa base de armadura e escudo sai UMA vez: o catálogo traz o número em
// `armor.defense` E como modificador de Defesa do mesmo valor, e desenhar os
// dois daria "Defesa +2 · Defesa +2" em toda armadura.
func thatGrantsItem(item sheet.ItemDTO) []string {
	catalog := catalogItem(item)
	if catalog == nil {
		return nil
	}
	badges := []string{}
	if base := baseItemDefense(*catalog); base != "" {
		badges = append(badges, base)
	}
	if catalog.Weapon != nil {
		badges = append(badges, "Dano "+catalog.Weapon.Damage)
	}
	for _, m := range catalog.Modifiers {
		badges = append(badges, modifierBadge(m))
	}
	return repetidosSem(badges)
}

func baseItemDefense(catalog book.Item) string {
	protection := catalog.Armor
	if protection == nil {
		protection = catalog.Shield
	}
	if protection == nil || protection.Defense == 0 {
		return ""
	}
	for _, m := range catalog.Modifiers {
		if m.Target.K == "defense" && m.Amount == protection.Defense {
			return ""
		}
	}
	return "Defesa " + book.WithSign(protection.Defense)
}

// modifierBadge escreve o que um modificador de item concede.
//
// Alvo de FLAG é booleano: ele não leva número, e escrever "Fadiga ao dormir
// +1" faria a tela prometer uma quantidade que não existe.
func modifierBadge(m engine.Modifier) string {
	label := targetLabel(m.Target)
	if m.Target.K == "flag" || m.Amount == 0 {
		return label
	}
	return label + " " + book.WithSign(m.Amount)
}

func repetidosSem(list []string) []string {
	seen := map[string]bool{}
	outside := []string{}
	for _, text := range list {
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		outside = append(outside, text)
	}
	return outside
}

// itemCatalog é o id de catálogo de uma linha do banco, ou "".
func itemCatalog(item sqlcgen.GetItemRow) string {
	if !item.Catalogid.Valid {
		return ""
	}
	return item.Catalogid.String
}

// howEngineItem traduz a entrada do livro para a forma que as validações do
// motor esperam. Só os campos que elas leem — eixo, id e nome —, porque um
// tradutor completo prometeria que os dois lados têm a mesma forma, e não têm.
func howEngineItem(catalog *book.Item) *engine.CatalogItem {
	if catalog == nil {
		return nil
	}
	return &engine.CatalogItem{
		ID: catalog.ID, Name: catalog.Name, Category: catalog.Category,
		Equip: catalog.Equip, Slots: catalog.Slots,
	}
}
