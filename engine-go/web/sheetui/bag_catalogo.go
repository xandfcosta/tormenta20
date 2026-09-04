package sheetui

import (
	"encoding/json"

	"t20engine/book"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/sheet"
)

// O CATÁLOGO DE ITENS lido pela Mochila (ALE-272, fatia 7).
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
	nomes := []string{}
	for _, entrada := range bookOverlays(item) {
		nomes = append(nomes, entrada.Name)
	}
	return nomes
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
	fora := []book.Item{}
	for _, id := range ids {
		if entrada := book.ItemByID(id); entrada != nil {
			fora = append(fora, *entrada)
		}
	}
	return fora
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
	catalogo := catalogItem(item)
	if catalogo == nil {
		return nil
	}
	crachas := []string{}
	if base := baseItemDefense(*catalogo); base != "" {
		crachas = append(crachas, base)
	}
	if catalogo.Weapon != nil {
		crachas = append(crachas, "Dano "+catalogo.Weapon.Damage)
	}
	for _, m := range catalogo.Modifiers {
		crachas = append(crachas, modifierBadge(m))
	}
	return repetidosSem(crachas)
}

func baseItemDefense(catalogo book.Item) string {
	protecao := catalogo.Armor
	if protecao == nil {
		protecao = catalogo.Shield
	}
	if protecao == nil || protecao.Defense == 0 {
		return ""
	}
	for _, m := range catalogo.Modifiers {
		if m.Target.K == "defense" && m.Amount == protecao.Defense {
			return ""
		}
	}
	return "Defesa " + book.WithSign(protecao.Defense)
}

// modifierBadge escreve o que um modificador de item concede.
//
// Alvo de FLAG é booleano: ele não leva número, e escrever "Fadiga ao dormir
// +1" faria a tela prometer uma quantidade que não existe.
func modifierBadge(m engine.Modifier) string {
	rotulo := targetLabel(m.Target)
	if m.Target.K == "flag" || m.Amount == 0 {
		return rotulo
	}
	return rotulo + " " + book.WithSign(m.Amount)
}

func repetidosSem(lista []string) []string {
	vistos := map[string]bool{}
	fora := []string{}
	for _, texto := range lista {
		if texto == "" || vistos[texto] {
			continue
		}
		vistos[texto] = true
		fora = append(fora, texto)
	}
	return fora
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
func howEngineItem(catalogo *book.Item) *engine.CatalogItem {
	if catalogo == nil {
		return nil
	}
	return &engine.CatalogItem{
		ID: catalogo.ID, Name: catalogo.Name, Category: catalogo.Category,
		Equip: catalogo.Equip, Slots: catalogo.Slots,
	}
}
