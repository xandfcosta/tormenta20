package book

import (
	"fmt"
	"slices"
)

// MaxItemImprovements é o teto de melhorias de um item: itens superiores
// "possuem de uma a quatro melhorias" (p164).
const MaxItemImprovements = 4

// FitsImprovementCount recusa o item que passaria de quatro melhorias.
//
// O MATERIAL conta como uma: a p165 o lista entre as melhorias, e "fabricar um
// item de material especial é uma melhoria". Por isso o teto não é "quatro
// melhorias e um material", que é a leitura que o campo separado do diálogo
// sugere. As melhorias da MESA entram na conta como as do livro: quem chama
// passa a lista inteira do pedido.
//
// @example book.FitsImprovementCount(3, true) // nil: três e o material são quatro
func FitsImprovementCount(improvements int, withMaterial bool) error {
	total := improvements
	if withMaterial {
		total++
	}
	if total <= MaxItemImprovements {
		return nil
	}
	if withMaterial {
		return fmt.Errorf("o item ficaria com %d melhorias contando o material, e o máximo é %d (p164); o material conta como uma (p165)",
			total, MaxItemImprovements)
	}
	return fmt.Errorf("o item ficaria com %d melhorias, e o máximo é %d (p164)", total, MaxItemImprovements)
}

// improvementPrerequisites são as melhorias que pedem outra no mesmo item (p164-166).
var improvementPrerequisites = map[string]string{
	"melhoria-atroz":      "melhoria-cruel",
	"melhoria-pungente":   "melhoria-certeira",
	"melhoria-sob-medida": "melhoria-ajustada",
}

// improvementsThatNeedAnother pedem "outra melhoria qualquer" (p165), e o material é uma delas.
var improvementsThatNeedAnother = map[string]bool{"melhoria-harmonizada-arma": true}

// improvementsThatExclude são os pares que o livro proíbe no mesmo item (p164-166).
var improvementsThatExclude = [][2]string{
	{"melhoria-macica", "melhoria-precisa"},
	{"melhoria-delicada", "melhoria-reforcada"},
}

// ImprovementSetProblem diz o que está errado no conjunto pedido para um item.
//
// Confere o CONJUNTO e não o que o item já tinha: quem grava manda a lista inteira, e conferir
// contra o estado anterior deixaria tirar o pré-requisito e manter a melhoria que depende dele
// no mesmo pedido.
//
// @example book.ImprovementSetProblem([]string{"melhoria-atroz"}, false) // "Atroz pede Cruel no mesmo item"
func ImprovementSetProblem(ids []string, withMaterial bool) error {
	for _, id := range ids {
		if needed, asks := improvementPrerequisites[id]; asks && !slices.Contains(ids, needed) {
			return fmt.Errorf("%s pede %s no mesmo item", improvementName(id), improvementName(needed))
		}
		if improvementsThatNeedAnother[id] && len(ids) < 2 && !withMaterial {
			return fmt.Errorf("%s pede outra melhoria qualquer no mesmo item (p165)", improvementName(id))
		}
	}
	for _, pair := range improvementsThatExclude {
		if slices.Contains(ids, pair[0]) && slices.Contains(ids, pair[1]) {
			return fmt.Errorf("um item não pode ser %s e %s ao mesmo tempo", improvementName(pair[0]), improvementName(pair[1]))
		}
	}
	return nil
}

func improvementName(id string) string {
	if item := ItemByID(id); item != nil {
		return item.Name
	}
	return id
}

// ImprovementFitsItem é a restrição que a família não diz: o `appliesTo` fala de arma, e a Mira
// telescópica é só de arma de disparo, "exceto fundas" (p166). A Delicada e a Selada são só de
// armadura pesada (p164, p166). A Aprimorado, que pede item que modifique uma perícia (p164), não
// entra: o catálogo não diz quais modificam, e uma lista à mão erraria em silêncio.
func ImprovementFitsItem(improvementID string, item Item) bool {
	switch improvementID {
	case "melhoria-mira-telescopica":
		return item.Weapon != nil && item.Weapon.Purpose == "ranged" && item.ID != "funda"
	case "melhoria-delicada", "melhoria-selada":
		return item.Category == "armor-heavy"
	}
	return true
}
