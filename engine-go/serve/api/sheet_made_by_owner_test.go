package api

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// A MARCA "FEITA POR MIM" NA FOLHA DO ITEM (ALE-649, decisão do dono). O Ferreiro e o Cano Raiado valem na "arma
// feita por você mesmo" (p69), e quem diz que a arma é dela é o jogador. A marca só aparece para quem tem um poder
// que a lê: para os outros ela seria um interruptor que não muda nada.

func madeByOwner(t *testing.T, f sceneFixture, item int64) bool {
	t.Helper()
	ids, err := f.s.sceneCore().Queries().ItemsMadeByOwnerOfCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("ler as marcas: %v", err)
	}
	for _, id := range ids {
		if id == item {
			return true
		}
	}
	return false
}

func TestTheSmithMarksTheWeaponAsHisOwn(t *testing.T) {
	f := newSceneFixture(t)
	sword := itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "wielded")
	if strings.Contains(itemSheetSearching(t, f, sword, ""), "Feita por mim") {
		t.Fatal("a folha da espada oferece a marca a quem não tem poder que a leia")
	}
	seedClasse(t, f.s, f.charID, "Inventor", 5)
	seedPowerChoices(t, f.s, f.charID, `["class.inventor.armeiro","class.inventor.ferreiro"]`, `{}`)
	if !strings.Contains(itemSheetSearching(t, f, sword, ""), "Feita por mim") {
		t.Fatal("a folha da espada não oferece a marca a quem tem o Ferreiro")
	}
	if refusal := bagCommand(t, f, f.charID, fmt.Sprintf("itens/%d/feita-por-mim", sword)); refusal != "" {
		t.Fatalf("marcar a espada foi recusado: %q", refusal)
	}
	if !madeByOwner(t, f, sword) {
		t.Fatal("o gesto não gravou a marca")
	}
	if refusal := bagCommand(t, f, f.charID, fmt.Sprintf("itens/%d/feita-por-mim", sword)); refusal != "" || madeByOwner(t, f, sword) {
		t.Errorf("tocar de novo não tirou a marca (recusa %q)", refusal)
	}
}
