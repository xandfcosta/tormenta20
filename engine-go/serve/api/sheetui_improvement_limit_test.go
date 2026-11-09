package api

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// AS QUATRO MELHORIAS DO ITEM. O teto é do livro: itens superiores "possuem de
// uma a quatro melhorias" (p164). E o MATERIAL ocupa uma das vagas, porque a
// p165 o lista entre as melhorias — "fabricar um item de material especial é
// uma melhoria". A melhoria da MESA também ocupa (decisão do dono): ela entra
// na mesma lista das do livro.
//
// O gesto regrava a lista inteira, então o que se conta é o PEDIDO, e não o que
// o item já tinha.

func TestAnItemTakesNoMoreThanFourImprovements(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	four := `"melhoria-certeira","melhoria-cruel","melhoria-pungente","melhoria-equilibrada"`

	// O CONTROLE: quatro cabem. Sem ele, "a quinta não entrou" se explicaria
	// igualmente bem por "nenhuma entra".
	if refusal := improvements(t, f, id, sword, `{"item_improvements":[`+four+`],"item_material":""}`); refusal != "" {
		t.Fatalf("o controle já estava errado: quatro melhorias foram recusadas: %q", refusal)
	}

	fifth := improvements(t, f, id, sword, `{"item_improvements":[`+four+`,"melhoria-atroz"],"item_material":""}`)
	if !strings.Contains(fifth, "164") {
		t.Errorf("a quinta melhoria entrou, ou a recusa não cita a página: %q", fifth)
	}
	if saved := itemImprovements(t, f, sword); strings.Contains(saved, "melhoria-atroz") {
		t.Errorf("a recusa gravou assim mesmo: %q", saved)
	}

	withMaterial := improvements(t, f, id, sword, `{"item_improvements":[`+four+`],"item_material":"material-adamante"}`)
	if !strings.Contains(withMaterial, "165") {
		t.Errorf("quatro melhorias e um material entraram, ou a recusa não diz que o material conta: %q", withMaterial)
	}
	if material := itemMaterial(t, f, sword); material != "" {
		t.Errorf("a recusa gravou o material: %q", material)
	}

	// E TRÊS COM O MATERIAL CABEM: a recusa acima é pela conta, e não por o
	// material não caber na espada.
	three := `"melhoria-certeira","melhoria-cruel","melhoria-pungente"`
	if refusal := improvements(t, f, id, sword, `{"item_improvements":[`+three+`],"item_material":"material-adamante"}`); refusal != "" {
		t.Errorf("três melhorias e um material foram recusados: %q", refusal)
	}
}

func TestTheTableImprovementTakesOneOfTheFour(t *testing.T) {
	f, id := fighterFixture(t)
	inCampaign(t, f, id)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	house := createsImprovement(t, f, improvementSignals("Fio da Casa", "weapon", "damage", "", 1))
	three := `"melhoria-certeira","melhoria-cruel","melhoria-pungente"`

	// O CONTROLE: três do livro e a da mesa cabem.
	if refusal := improvements(t, f, id, sword, `{"item_improvements":[`+three+`,"`+house+`"],"item_material":""}`); refusal != "" {
		t.Fatalf("o controle já estava errado: três do livro e a da mesa foram recusadas: %q", refusal)
	}
	if refusal := improvements(t, f, id, sword, `{"item_improvements":[`+three+`,"`+house+`"],"item_material":"material-adamante"}`); !strings.Contains(refusal, "164") {
		t.Errorf("a da mesa não ocupou uma das quatro vagas: %q", refusal)
	}
}

func itemMaterial(t *testing.T, f sceneFixture, item int64) string {
	t.Helper()
	var material sql.NullString
	if err := f.s.db.QueryRowContext(context.Background(), "SELECT material FROM character_items WHERE id = ?", item).Scan(&material); err != nil {
		t.Fatalf("ler o material: %v", err)
	}
	return material.String
}
