package book

import (
	"fmt"
	"slices"
	"strings"

	"t20engine/domain/engine"
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

// BuiltInOverlays é o que o verbete já traz aplicado: o item específico é um item base com sobreposição
// (p337, p340). Nada, para o item comum.
//
//	book.BuiltInOverlays(book.ItemByID("baluarte-anao")) // ["melhoria-reforcada"], "material-adamante"
func BuiltInOverlays(item *Item) (improvements []string, material string) {
	if item == nil || item.ComposedOf == nil {
		return nil, ""
	}
	return item.ComposedOf.Improvements, item.ComposedOf.Material
}

// ImprovementSetProblem diz o que está errado no que se pede para um item, contando o que o verbete traz.
//
// Confere o CONJUNTO e não o que o item já tinha: quem grava manda a lista inteira, e conferir contra o
// estado anterior deixaria tirar o pré-requisito e manter a melhoria que depende dele no mesmo pedido. Só
// o PEDIDO é cobrado: o verbete que o livro entrega pronto não precisa cumprir o pré-requisito, e a
// Espada-Deus tem a Atroz sem a Cruel.
//
// @example book.ImprovementSetProblem([]string{"melhoria-atroz"}, nil, false) // "Atroz pede Cruel no mesmo item"
func ImprovementSetProblem(requested, builtIn []string, withMaterial bool) error {
	all := append(slices.Clone(requested), builtIn...)
	for _, id := range requested {
		if slices.Contains(builtIn, id) {
			return fmt.Errorf("este item já vem com %s, e cada melhoria entra uma vez no item (p164)", improvementName(id))
		}
		if needed, asks := improvementPrerequisites[id]; asks && !slices.Contains(all, needed) {
			return fmt.Errorf("%s pede %s no mesmo item", improvementName(id), improvementName(needed))
		}
		if improvementsThatNeedAnother[id] && len(all) < 2 && !withMaterial {
			return fmt.Errorf("%s pede outra melhoria qualquer no mesmo item (p165)", improvementName(id))
		}
		for _, pair := range improvementsThatExclude {
			other := ""
			switch id {
			case pair[0]:
				other = pair[1]
			case pair[1]:
				other = pair[0]
			}
			if other != "" && slices.Contains(all, other) {
				return fmt.Errorf("um item não pode ser %s e %s ao mesmo tempo", improvementName(id), improvementName(other))
			}
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
// armadura pesada (p164, p166). A Aprimorado é de quem modifica uma perícia (p164).
func ImprovementFitsItem(improvementID string, item Item) bool {
	switch improvementID {
	case "melhoria-mira-telescopica":
		return item.Weapon != nil && item.Weapon.Purpose == "ranged" && item.ID != "funda"
	case "melhoria-delicada", "melhoria-selada":
		return item.Category == "armor-heavy"
	case engine.ImprovedItemImprovement:
		return len(engine.ImprovedSkillsOf(item.Modifiers, item.SkillTies)) > 0
	}
	return true
}

// ImprovedSummary é a frase da Aprimorado na lista daquele item: "+1 em Cura".
func ImprovedSummary(item Item) string {
	names := []string{}
	for _, m := range engine.ImprovedSkillsOf(item.Modifiers, item.SkillTies) {
		if !slices.Contains(names, m.Target.Name) {
			names = append(names, m.Target.Name)
		}
	}
	return "+1 em " + strings.Join(names, ", ")
}
