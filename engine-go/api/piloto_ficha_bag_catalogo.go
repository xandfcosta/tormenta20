package api

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

// itemDoCatalogo acha a entrada do livro de um item da ficha.
func itemDoCatalogo(item sheet.ItemDTO) *book.Item {
	if item.CatalogID == nil || *item.CatalogID == "" {
		return nil
	}
	return book.ItemByID(*item.CatalogID)
}

// asSobreposicoesDoItem são os NOMES das melhorias e do material aplicados.
func asSobreposicoesDoItem(item sheet.ItemDTO) []string {
	nomes := []string{}
	for _, entrada := range asSobreposicoesDoLivro(item) {
		nomes = append(nomes, entrada.Name)
	}
	return nomes
}

// asSobreposicoesDoLivro resolve melhorias + material contra o catálogo.
//
// Id desconhecido é PULADO em vez de virar erro: a coluna guarda um blob de
// texto, e uma linha torta não pode impedir a mochila inteira de abrir.
func asSobreposicoesDoLivro(item sheet.ItemDTO) []book.Item {
	ids := asMelhoriasGuardadas(item.Improvements)
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

// asMelhoriasGuardadas lê o blob JSON da coluna `improvements`.
func asMelhoriasGuardadas(blob string) []string {
	var ids []string
	if json.Unmarshal([]byte(blob), &ids) != nil {
		return nil
	}
	return ids
}

// oQueOItemConcede são os crachás do que o item dá enquanto equipado.
//
// A Defesa base de armadura e escudo sai UMA vez: o catálogo traz o número em
// `armor.defense` E como modificador de Defesa do mesmo valor, e desenhar os
// dois daria "Defesa +2 · Defesa +2" em toda armadura.
func oQueOItemConcede(item sheet.ItemDTO) []string {
	catalogo := itemDoCatalogo(item)
	if catalogo == nil {
		return nil
	}
	crachas := []string{}
	if base := aDefesaBaseDoItem(*catalogo); base != "" {
		crachas = append(crachas, base)
	}
	if catalogo.Weapon != nil {
		crachas = append(crachas, "Dano "+catalogo.Weapon.Damage)
	}
	for _, m := range catalogo.Modifiers {
		crachas = append(crachas, oCrachaDoModificador(m))
	}
	return semRepetidos(crachas)
}

func aDefesaBaseDoItem(catalogo book.Item) string {
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
	return "Defesa " + comSinalInt(protecao.Defense)
}

// oCrachaDoModificador escreve o que um modificador de item concede.
//
// Alvo de FLAG é booleano: ele não leva número, e escrever "Fadiga ao dormir
// +1" faria a tela prometer uma quantidade que não existe.
func oCrachaDoModificador(m engine.Modifier) string {
	rotulo := targetLabel(m.Target)
	if m.Target.K == "flag" || m.Amount == 0 {
		return rotulo
	}
	return rotulo + " " + comSinalInt(m.Amount)
}

func semRepetidos(lista []string) []string {
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

// oCatalogoDoItem é o id de catálogo de uma linha do banco, ou "".
func oCatalogoDoItem(item sqlcgen.GetItemRow) string {
	if !item.Catalogid.Valid {
		return ""
	}
	return item.Catalogid.String
}

// oItemComoDoMotor traduz a entrada do livro para a forma que as validações do
// motor esperam. Só os campos que elas leem — eixo, id e nome —, porque um
// tradutor completo prometeria que os dois lados têm a mesma forma, e não têm.
func oItemComoDoMotor(catalogo *book.Item) *engine.CatalogItem {
	if catalogo == nil {
		return nil
	}
	return &engine.CatalogItem{
		ID: catalogo.ID, Name: catalogo.Name, Category: catalogo.Category,
		Equip: catalogo.Equip, Slots: catalogo.Slots,
	}
}
