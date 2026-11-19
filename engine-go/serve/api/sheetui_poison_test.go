package api

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// O VENENO NO DIÁLOGO DO ITEM diz a inoculação, o efeito e a CD (p161-162), e o Veneno Potente
// (p74) soma +5 nela, decomposto. A CD é de quem aplica, com a Inteligência (p161): a ficha é de
// nível 8 com Inteligência 3, então a base é 10 + 4 + 3 = 17. O app não aplica o veneno: quem lê a
// CD e aplica é o mestre (decisão do dono).
func TestThePoisonDialogSaysTheDCAndPotentPoisonAddsFive(t *testing.T) {
	cases := []struct {
		name          string
		catalog, item string
		potent        bool
		want          []string
		refuse        string
	}{
		{"peçonha com o poder", "peconha-comum", "Peçonha comum", true,
			[]string{"Contato", "perde 1d12 PV", "Fortitude CD 22 (+5 Veneno Potente)"}, "se passar"},
		{"peçonha sem o poder", "peconha-comum", "Peçonha comum", false,
			[]string{"Contato", "Fortitude CD 17"}, "Veneno Potente"},
		{"beladona com o poder", "beladona", "Beladona", true,
			[]string{"Ingestão", "fica paralisada por 3 rodadas", "Fortitude CD 27 (+5 Beladona, +5 Veneno Potente)", "se passar, fica lenta"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			rogue(t, f, c.potent)
			item := itemSemeia(t, f, f.charID, c.catalog, c.item, "")

			dialog := itemDialogText(t, f, item)
			for _, want := range c.want {
				if !strings.Contains(dialog, want) {
					t.Errorf("o diálogo de %s não diz %q:\n%s", c.item, want, dialog)
				}
			}
			if c.refuse != "" && strings.Contains(dialog, c.refuse) {
				t.Errorf("o diálogo de %s diz %q", c.item, c.refuse)
			}
		})
	}
}

// rogue faz da ficha da bancada um LADINO de Inteligência 3, com ou sem o Veneno Potente: o poder de
// classe só vale para quem tem a classe.
func rogue(t *testing.T, f sceneFixture, potent bool) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.s.db.ExecContext(ctx,
		"UPDATE character_level_choices SET ref = 'Ladino' WHERE characterId = ? AND kind = 'class'", f.charID); err != nil {
		t.Fatalf("fazer da ficha um ladino: %v", err)
	}
	if potent {
		withPowers(t, f, `["class.ladino.veneno-potente"]`, "{}")
	}
	if _, err := f.s.db.ExecContext(ctx, "UPDATE characters SET intelligence = 3 WHERE id = ?", f.charID); err != nil {
		t.Fatalf("pôr a Inteligência: %v", err)
	}
}

// itemDialogText é o texto do diálogo daquele item na mochila, como o navegador o concatena.
func itemDialogText(t *testing.T, f sceneFixture, item int64) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(bagPage(t, f, f.player, f.charID)))
	if err != nil {
		t.Fatalf("ler a mochila: %v", err)
	}
	dialog := findNode(doc, "id", fmt.Sprintf("item-%d", item))
	if dialog == nil {
		t.Fatalf("a mochila não tem o diálogo do item %d", item)
	}
	return textOf(dialog)
}
