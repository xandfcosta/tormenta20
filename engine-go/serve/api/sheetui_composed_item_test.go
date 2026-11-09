package api

import (
	"strings"
	"testing"
)

// O ITEM ESPECÍFICO JÁ VEM COM SOBREPOSIÇÃO (p340): o Baluarte anão é uma armadura completa de adamante
// com Reforçada. O que ele traz ocupa as vagas de melhoria (p164-165), e o material é um só.
const bulwark = "baluarte-anao"

func TestAComposedItemHasFewerFreeSlots(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, bulwark, "Baluarte anão", "")

	// O adamante e a Reforçada são duas: cabem mais duas, e a terceira não.
	two := `"melhoria-ajustada","melhoria-polida"`
	if refusal := improvements(t, f, id, item, `{"item_improvements":[`+two+`]}`); refusal != "" {
		t.Fatalf("o controle já estava errado: duas a mais foram recusadas: %q", refusal)
	}
	third := improvements(t, f, id, item, `{"item_improvements":[`+two+`,"melhoria-sob-medida"]}`)
	if !strings.Contains(third, "164") {
		t.Errorf("a quinta vaga entrou, ou a recusa não cita a página: %q", third)
	}
	// Tirar as que se puseram passa sempre, mesmo sem vaga.
	if refusal := improvements(t, f, id, item, `{"item_improvements":[]}`); refusal != "" {
		t.Errorf("tirar foi recusado: %q", refusal)
	}
}

func TestAComposedItemHasOneMaterial(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, bulwark, "Baluarte anão", "")
	refusal := improvements(t, f, id, item, `{"item_improvements":[],"item_material":"material-mitral"}`)
	if !strings.Contains(refusal, "Adamante") {
		t.Errorf("o segundo material devia ser recusado nomeando o que ele já tem, e veio %q", refusal)
	}
	if material := itemMaterial(t, f, item); material != "" {
		t.Errorf("a recusa gravou o material: %q", material)
	}
	if dialog := improvementDialog(t, f, id, item); strings.Contains(dialog, `id="material-`) {
		t.Error("o diálogo oferece material a quem já é de adamante")
	}
}

// "Cada melhoria só pode ser aplicada uma vez a um mesmo item" (p164), e o que o verbete traz conta para
// o pré-requisito e para os pares proibidos.
func TestAComposedItemCountsForTheImprovementRules(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, bulwark, "Baluarte anão", "")
	if refusal := improvements(t, f, id, item, `{"item_improvements":["melhoria-reforcada"]}`); refusal == "" {
		t.Error("a Reforçada entrou de novo num item que já a traz")
	}
	if refusal := improvements(t, f, id, item, `{"item_improvements":["melhoria-delicada"]}`); !strings.Contains(refusal, "não pode") {
		t.Errorf("a Delicada entrou num item Reforçado, e veio %q", refusal)
	}
	dialog := improvementDialog(t, f, id, item)
	if strings.Contains(dialog, `aria-label="Reforçada"`) {
		t.Error("o diálogo oferece a Reforçada a quem já a traz")
	}
	if !strings.Contains(dialog, "2 de 4 melhorias") {
		t.Error("o contador não conta o que o verbete traz")
	}
}

// O que o livro entrega pronto não é cobrado: a Espada-Deus tem a Atroz sem a Cruel. O pedido é.
func TestAComposedItemIsNotAskedForWhatTheBookBuilt(t *testing.T) {
	f, id := fighterFixture(t)
	item := itemSemeia(t, f, id, "espada-deus", "Espada-Deus", "")
	if refusal := improvements(t, f, id, item, `{"item_improvements":["melhoria-cruel"]}`); refusal != "" {
		t.Errorf("uma a mais, sem relação com o que ela traz, foi recusada: %q", refusal)
	}
}
