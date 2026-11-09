package book

import "fmt"

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
