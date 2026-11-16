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

// A FOLHA DO ITEM DIZ O QUE A MARCA MUDOU. A linha de referência é o que o LIVRO diz da
// arma, e fica; abaixo dela, a da arma NA MÃO desta ficha, com a mesma conta da carta do
// Combate. Sem ela, quem liga "Feita por mim" via o dano do livro logo acima do
// interruptor e não via, ali, o passo que o Ferreiro deu — só na aba Combate.
//
// Os DADOS são o que se prende, e não o total: o +N é da Força do Inventor, e a conta
// dela é de outro caso.
func TestTheItemSheetShowsTheDamageInTheHandBelowTheBook(t *testing.T) {
	f := newSceneFixture(t)
	sword := itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "wielded")
	seedClasse(t, f.s, f.charID, "Inventor", 5)
	seedPowerChoices(t, f.s, f.charID, `["class.inventor.armeiro","class.inventor.ferreiro"]`, `{}`)

	before := itemSheetSearching(t, f, sword, "")
	if !strings.Contains(before, "dano 1d8 · crítico") {
		t.Fatalf("a referência do livro sumiu da folha da espada:\n%s", before)
	}
	if !strings.Contains(before, "na sua mão: 1d8") {
		t.Errorf("a folha da espada empunhada não diz o dano na mão:\n%s", before)
	}

	if refusal := bagCommand(t, f, f.charID, fmt.Sprintf("itens/%d/feita-por-mim", sword)); refusal != "" {
		t.Fatalf("marcar a espada foi recusado: %q", refusal)
	}
	after := itemSheetSearching(t, f, sword, "")
	if !strings.Contains(after, "na sua mão: 1d10") {
		t.Errorf("com a marca, o Ferreiro dá um passo, e a linha da mão não o diz:\n%s", after)
	}
	if !strings.Contains(after, "dano 1d8 · crítico") {
		t.Errorf("a marca mexeu na referência do livro, que é do catálogo:\n%s", after)
	}
}

// A arma GUARDADA não está na mão de ninguém, e não ganha a linha.
func TestAStowedWeaponHasNoDamageInTheHand(t *testing.T) {
	f := newSceneFixture(t)
	sword := itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "")
	if screen := itemSheetSearching(t, f, sword, ""); strings.Contains(screen, "na sua mão") {
		t.Errorf("a espada guardada diz um dano na mão:\n%s", screen)
	}
}
