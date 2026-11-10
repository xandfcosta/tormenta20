package api

import (
	"context"
	"strings"
	"testing"
)

// O QUE O ENCANTO FAZ se lê na folha e na Mochila, nas duas pontas da mesa: o gasto de 2 PM no golpe, e o tipo
// que a Anticriatura escolheu. Antes a folha dizia "sem efeito mecânico" para os dois.
func TestTheItemSheetSaysWhatTheEnchantSpendDoes(t *testing.T) {
	f := newSceneFixture(t)
	sword := itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(context.Background(), sword,
		[]string{"encanto-anticriatura", "encanto-flamejante"}, map[string]string{"encanto-anticriatura": "morto-vivo"}); err != nil {
		t.Fatal(err)
	}
	screen := bagScreen(t, f, f.charID)
	block := listNamed(itemDialogOf(t, screen, "Espada longa"), "Encantos")
	if block == nil {
		t.Fatal("a folha da espada não tem o bloco de encantos")
	}
	written := renderNode(t, block)
	for _, want := range []string{
		"Anticriatura · contra morto-vivo", "2 PM no golpe, uma vez por rodada: +4d8 de dano",
		"+1d6 de dano de fogo", "2 PM em vez do golpe, uma vez por rodada: Bola de fogo",
	} {
		if !strings.Contains(written, want) {
			t.Errorf("o bloco de encantos não escreve %q:\n%s", want, written)
		}
	}
	if strings.Contains(written, "sem efeito mecânico") {
		t.Errorf("o bloco de encantos diz que um encanto com gasto não tem efeito:\n%s", written)
	}
	if !strings.Contains(screen, "Anticriatura contra morto-vivo") {
		t.Error("a linha da Mochila não diz o tipo escolhido da Anticriatura")
	}
}
