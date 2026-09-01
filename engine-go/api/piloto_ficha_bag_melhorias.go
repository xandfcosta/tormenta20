package api

import (
	"fmt"
	"strings"
	"t20engine/book"
)

// MELHORIA E MATERIAL: a regra de quem cabe em quem (ALE-272, fatia 7).
//
// # Esta regra era só da TELA, e agora é do servidor
//
// A compatibilidade entre uma melhoria e o item que a recebe vivia inteira no
// TypeScript: o `familyFor` classificava o item e o diálogo filtrava as listas
// pelo `appliesTo` do catálogo. O Go tinha o campo e não o lia — o próprio
// `handleAddItem` registra a dívida em comentário: "overlay compatibility
// (improvements/material vs the item family) is not yet validated here — the
// frontend pre-validates it".
//
// Pré-validação de tela não é fronteira: um pedido montado à mão põe corda de
// arco num escudo, e o servidor grava. É a mesma forma do `requiresCircle` dos
// aprimoramentos, fechada na fatia 6, e ela morreria com a SPA — que é o
// destino desta migração.

// asCategoriasSemSobreposicao são as que nunca recebem melhoria nem material.
//
// Não se forja uma poção em aço-rubi, nem se põe mira telescópica num cavalo.
// Recusar aqui é o que faz o botão SUMIR na tela em vez de abrir um diálogo com
// duas listas vazias.
var asCategoriasSemSobreposicao = map[string]bool{
	"consumable": true, "meal": true, "catalyst": true,
	"improvement": true, "material": true, "animal": true, "vehicle": true,
}

func aceitaMelhoria(catalogo book.Item) bool {
	return !asCategoriasSemSobreposicao[catalogo.Category]
}

// aFamiliaDoItem é a classificação grossa que o `appliesTo` do catálogo usa.
//
// Quatro famílias, e a última é o resto: arma, armadura, escudo e vestuário. É
// a mesma tabela do `familyFor` do front, e ela é do CATÁLOGO — não do livro —,
// então ela mora ao lado de quem a consome.
func aFamiliaDoItem(catalogo book.Item) string {
	switch {
	case strings.HasPrefix(catalogo.Category, "weapon-"):
		return "weapon"
	case strings.HasPrefix(catalogo.Category, "armor-"):
		return "armor"
	case catalogo.Category == "shield":
		return "shield"
	}
	return "apparel"
}

// aceitaAFamilia diz se a sobreposição serve à família do item.
//
// Sobreposição SEM `appliesTo` serve a qualquer um: o catálogo usa o campo para
// restringir, e a ausência dele é "não restringe" — não "não serve a ninguém",
// que faria uma melhoria nova nascer inalcançável.
func aceitaAFamilia(sobreposicao book.Item, familia string) bool {
	if len(sobreposicao.AppliesTo) == 0 {
		return true
	}
	return contemTraco(sobreposicao.AppliesTo, familia)
}

// aMelhoriaCabeNoItem é a RECUSA do servidor, e ela é a fronteira.
//
// Ela confere três coisas de cada id: que ele existe no catálogo, que ele é da
// categoria certa (melhoria não entra no campo do material e vice-versa), e que
// ele serve à família do item.
func aMelhoriaCabeNoItem(catalogo *book.Item, ids []string, categoria string) error {
	if catalogo == nil {
		return fmt.Errorf("um item custom não recebe melhoria: ele não tem família no catálogo")
	}
	if len(ids) > 0 && !aceitaMelhoria(*catalogo) {
		return fmt.Errorf("%q não recebe melhoria nem material", catalogo.Name)
	}
	familia := aFamiliaDoItem(*catalogo)
	for _, id := range ids {
		sobreposicao := itemDoLivroPorID(id)
		if sobreposicao == nil || sobreposicao.Category != categoria {
			return fmt.Errorf("%q não é uma %s do livro", id, oNomeDaCategoria(categoria))
		}
		if !aceitaAFamilia(*sobreposicao, familia) {
			return fmt.Errorf("%q não cabe em %q", sobreposicao.Name, catalogo.Name)
		}
	}
	return nil
}

func oNomeDaCategoria(categoria string) string {
	if categoria == "material" {
		return "material"
	}
	return "melhoria"
}
