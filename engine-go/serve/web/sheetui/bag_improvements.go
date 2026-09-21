package sheetui

import (
	"fmt"
	"strings"
	"t20engine/domain/book"
)

// MELHORIA E MATERIAL: a regra de quem cabe em quem.
//
// Ela é do SERVIDOR e não da tela, e a razão é a de sempre: pré-validação de
// tela não é fronteira — um pedido montado à mão põe corda de arco num escudo, e
// o servidor grava.

// categoriesWithoutOverlap são as que nunca recebem melhoria nem material.
//
// Não se forja uma poção em aço-rubi, nem se põe mira telescópica num cavalo.
// Recusar aqui é o que faz o botão SUMIR na tela em vez de abrir um diálogo com
// duas listas vazias.
var categoriesWithoutOverlap = map[string]bool{
	"consumable": true, "meal": true, "catalyst": true,
	"improvement": true, "material": true, "animal": true, "vehicle": true,
}

func aceitaMelhoria(catalog book.Item) bool {
	return !categoriesWithoutOverlap[catalog.Category]
}

// itemFamily é a classificação grossa que o `appliesTo` do catálogo usa.
//
// Quatro famílias, e a última é o resto: arma, armadura, escudo e vestuário. Ela
// é do CATÁLOGO e não do livro, então mora ao lado de quem a consome.
func itemFamily(catalog book.Item) string {
	switch {
	case strings.HasPrefix(catalog.Category, "weapon-"):
		return "weapon"
	case strings.HasPrefix(catalog.Category, "armor-"):
		return "armor"
	case catalog.Category == "shield":
		return "shield"
	}
	return "apparel"
}

// aceitaAFamilia diz se a sobreposição serve à família do item.
//
// Sobreposição SEM `appliesTo` serve a qualquer um: o catálogo usa o campo para
// restringir, e a ausência dele é "não restringe" — não "não serve a ninguém",
// que faria uma melhoria nova nascer inalcançável.
func aceitaAFamilia(overlap book.Item, family string) bool {
	if len(overlap.AppliesTo) == 0 {
		return true
	}
	return contemTraco(overlap.AppliesTo, family)
}

// fitsItemImprovement é a RECUSA do servidor, e ela é a fronteira.
//
// Ela confere três coisas de cada id: que ele existe no catálogo, que ele é da
// categoria certa (melhoria não entra no campo do material e vice-versa), e que
// ele serve à família do item.
func fitsItemImprovement(catalog *book.Item, ids []string, category string) error {
	if catalog == nil {
		return fmt.Errorf("um item custom não recebe melhoria: ele não tem família no catálogo")
	}
	if len(ids) > 0 && !aceitaMelhoria(*catalog) {
		return fmt.Errorf("%q não recebe melhoria nem material", catalog.Name)
	}
	family := itemFamily(*catalog)
	for _, id := range ids {
		overlap := book.ItemByID(id)
		if overlap == nil || overlap.Category != category {
			return fmt.Errorf("%q não é uma %s do livro", id, categoryName(category))
		}
		if !aceitaAFamilia(*overlap, family) {
			return fmt.Errorf("%q não cabe em %q", overlap.Name, catalog.Name)
		}
	}
	return nil
}

func categoryName(category string) string {
	if category == "material" {
		return "material"
	}
	return "melhoria"
}
