package book

import "testing"

// TODA PEÇA DE UM ITEM COMPOSTO EXISTE E CABE NELE (p337, p340, ALE-445).
//
// O verbete específico aponta para o que já está no catálogo — a base, o
// material, as melhorias e os encantos que o livro escreve em itálico. Um
// ponteiro torto não dá erro visível: a primagem do motor recusa o id que não
// existe, mas ninguém recusa um encanto de ARMADURA numa espada, e o item
// entraria com metade da regra em silêncio.
//
// Aqui o item já está EXPANDIDO, e é por isso que o guarda mora no `book` e não
// no `catalog`: a pergunta "este encanto cabe?" se responde pelos blocos de
// arma e armadura que a composição herdou, e a resposta é a mesma função que o
// servidor usa para recusar o encanto que o mestre tenta pôr.
func TestEveryComposedItemBringsPiecesThatFitIt(t *testing.T) {
	byID := map[string]Item{}
	for _, item := range Catalogs().Items {
		byID[item.ID] = item
	}

	medidos := 0
	for _, item := range Catalogs().Items {
		composed := item.ComposedOf
		if composed == nil {
			continue
		}
		medidos++

		base, achada := byID[composed.Base]
		switch {
		case !achada:
			t.Errorf("%s diz que sua base é %q, e não há item com esse id",
				item.Name, composed.Base)
		case base.ComposedOf != nil:
			t.Errorf("%s tem por base %q, que também é composto — o livro não "+
				"compõe sobre composto, e a expansão de um dependeria da do outro",
				item.Name, base.ID)
		case base.Category != item.Category:
			t.Errorf("%s é %q e a base dela (%s) é %q: o item específico É o item "+
				"da base, e a categoria é o que decide proficiência, filtro e "+
				"sobreposição que cabe", item.Name, item.Category, base.Name, base.Category)
		}

		// O ENCANTO pela mesma porta do mestre: família certa, tabela certa,
		// teto de três por PESO e pré-requisito no mesmo conjunto (p333-339).
		if err := FitsItemEnchants(&item, composed.Enchants); err != nil {
			t.Errorf("os encantos de %s não cabem nela: %v", item.Name, err)
		}
		for _, id := range composed.Improvements {
			conferirSobreposicao(t, byID, item, id, "improvement")
		}
		if composed.Material != "" {
			conferirSobreposicao(t, byID, item, composed.Material, "material")
		}
	}

	// O DENOMINADOR: sem ele, "nenhuma peça torta" e "não achei item composto
	// nenhum" têm a mesma cor no terminal.
	if medidos == 0 {
		t.Fatal("nenhum item composto no catálogo: este guarda não mediu nada")
	}
	t.Logf("%d itens compostos medidos", medidos)
}

// conferirSobreposicao cobra de uma melhoria ou material as duas coisas que o
// servidor cobra quando o jogador a aplica: ser da categoria certa e servir à
// família do item.
func conferirSobreposicao(t *testing.T, byID map[string]Item, item Item, id, category string) {
	t.Helper()
	piece, achada := byID[id]
	if !achada {
		t.Errorf("%s traz %q, e não há item com esse id", item.Name, id)
		return
	}
	if piece.Category != category {
		t.Errorf("%s traz %q como %s, e ele é %q", item.Name, piece.Name, category,
			piece.Category)
	}
	if !FitsOverlayFamily(item, piece) {
		t.Errorf("%s traz %q, que só serve a %v — e ela é %q", item.Name, piece.Name,
			piece.AppliesTo, ItemFamily(item))
	}
}
