package sheetui

import (
	"testing"

	"t20engine/domain/book"
)

// O DIÁLOGO DE ACRESCENTAR SÓ OFERTA O QUE SE CARREGA (ALE-446, ALE-445).
//
// Ele é do JOGADOR, e duas famílias de verbete não cabem nele:
//
//   - a SOBREPOSIÇÃO — melhoria, material, encanto — não é item que se carrega,
//     é coisa que se aplica a um item. Quem as aplica é o diálogo de melhorias
//     (e, no caso do encanto, o mestre). Ofertá-las aqui deixa a pessoa pôr um
//     "Aço-rubi" ou um "Formidável" solto na mochila;
//   - o item COMPOSTO — o específico das Tabelas 8-9 e 8-11 —, porque ele é
//     mágico: "não se compra, se acha ou se fabrica" (p333). Quem o entrega é o
//     mestre, pela mesma porta do encanto.
func TestTheAddDialogOffersOnlyWhatACharacterCarries(t *testing.T) {
	rows := catalogItemRowsOf("", "")
	if len(rows) == 0 {
		t.Fatal("o diálogo não ofertou nada: o caso não mediria o filtro")
	}
	byID := map[string]book.Item{}
	for _, entry := range book.Catalogs().Items {
		byID[entry.ID] = entry
	}
	for _, row := range rows {
		entry, achado := byID[row.ID]
		if !achado {
			t.Errorf("o diálogo oferta %q, e não há item com esse id", row.ID)
			continue
		}
		if !carriedCategory[entry.Category] {
			t.Errorf("o diálogo oferta %q, que é %q — uma categoria que se APLICA a "+
				"um item, não uma que se carrega", entry.Name, entry.Category)
		}
		// O QUE SE SORTEIA é mágico, e o mágico PERMANENTE é do mestre (p333).
		// A poção é a exceção da mesma página: uso único, e se FABRICA.
		if entry.RollMax > 0 && entry.Consumable == nil {
			t.Errorf("o diálogo oferta %q, que é item mágico permanente (sai numa "+
				"tabela de saque): quem o entrega é o mestre (p333)", entry.Name)
		}
	}
	t.Logf("%d linhas ofertadas, de %d itens no catálogo", len(rows), len(byID))
}

// E TODA CATEGORIA DO CATÁLOGO ESTÁ CLASSIFICADA.
//
// As duas listas juntas têm de cobrir o catálogo inteiro. É a inversão que esta
// fatia veio fazer: o filtro era uma lista de PROIBIDOS com dois nomes, escrita
// quando só havia duas espécies de sobreposição — as duas de encanto nasceram
// depois e entraram caladas no diálogo do jogador.
//
// Categoria nova reprova PELO NOME, que é a diferença entre "classifique isto"
// e "procure o que está errado".
func TestEveryItemCategoryIsEitherCarriedOrApplied(t *testing.T) {
	medidas := map[string]bool{}
	for _, entry := range book.Catalogs().Items {
		if medidas[entry.Category] {
			continue
		}
		medidas[entry.Category] = true
		if carriedCategory[entry.Category] == appliedCategory[entry.Category] {
			t.Errorf("a categoria %q (%s) não está em exatamente uma das duas listas: "+
				"ou ela se CARREGA e entra no diálogo de acrescentar, ou ela se APLICA "+
				"a um item e entra no de melhorias", entry.Category, entry.Name)
		}
	}
	t.Logf("%d categorias medidas", len(medidas))
}
