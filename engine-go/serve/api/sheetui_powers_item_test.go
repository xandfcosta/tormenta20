package api

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// powersTabOf recorta o painel da aba Poderes — a `<section>` dele, sem os
// diálogos que vêm depois —, e falha quando não o acha. A folha de cada poder é
// desenhada fora do painel e repete o nome, então a tela inteira acharia ali o
// que a lista escondeu.
func powersTabOf(t *testing.T, screen string) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(screen))
	if err != nil {
		t.Fatalf("ler a aba Poderes: %v", err)
	}
	panel := findNode(doc, "aria-label", "Poderes")
	if panel == nil || panel.Data != "section" {
		t.Fatal("a ficha não desenhou o painel de Poderes")
	}
	return renderNode(t, panel)
}

// powerListRowOf é a linha daquele poder na lista — do `ds-row` que abre a linha
// ao fim dela —, ou "". O âncora é o TÍTULO da linha: o nome de um poder pode
// aparecer na meta de outro.
func powerListRowOf(panel, name string) string {
	at := strings.Index(panel, `<span class="ds-row-title">`+name+`</span>`)
	if at < 0 {
		return ""
	}
	start := strings.LastIndex(panel[:at], `<div class="ds-row`)
	end := strings.Index(panel[at:], "</div>")
	if start < 0 || end < 0 {
		return ""
	}
	return panel[start : at+end]
}

// O ENCANTO ATIVÁVEL DE UM ITEM EQUIPADO É PODER (ALE-433).
//
// É a SEXTA procedência do acervo, e ela entrou porque o lugar onde se procura
// "o que eu posso fazer" é a aba de Poderes: a Dançarina (p335) custa uma ação
// de movimento e 1 PM, e até aqui ela existia só como prosa na ficha do item —
// três toques longe de quem está no turno.
//
// O CRACHÁ DIZ O NOME DA ARMA, e não "Item": duas espadas encantadas com o
// mesmo encanto são duas linhas, e sem o nome de cada uma elas ficam iguais.
func TestTheActivatableEnchantOfAWieldedItemIsAPower(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), sword, []string{"encanto-dancarina", "encanto-flamejante"},
	); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}

	screen := powersTabOf(t, powerScreen(t, f, id))
	row := powerListRowOf(screen, "Dançarina")
	if row == "" {
		t.Fatalf("o encanto ativável não entrou no acervo:\n%s", screen)
	}
	// "movimento" sem caixa: a linha escreve a ação em versal.
	for _, want := range []string{"espada longa", "1 pm", "movimento"} {
		if !strings.Contains(strings.ToLower(row), want) {
			t.Errorf("a linha da Dançarina não diz %q:\n%s", want, row)
		}
	}
	// O PASSIVO NÃO ENTRA, e esta é a outra metade da regra: a Flamejante soma
	// +1d6 de fogo sozinha, já aparece no cartão da arma e na Mochila, e não é
	// coisa que se FAZ. O acervo é o que a pessoa pode fazer.
	if strings.Contains(screen, "Flamejante") {
		t.Errorf("o encanto PASSIVO entrou no acervo de poderes:\n%s", screen)
	}
}

// USAR O ENCANTO COBRA O PM DO LIVRO — 1 PM, p335.
func TestUsingTheItemEnchantChargesItsMp(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), sword, []string{"encanto-dancarina"},
	); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}

	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/item.encanto-dancarina", ""); refusal != "" {
		t.Fatalf("usar o encanto foi recusado: %q", refusal)
	}
	// EM DELTA, porque o saldo de partida é o poço que o livro dá ao guerreiro
	// de nível 3 — um número escolhido pela bancada mediria um personagem que
	// não pode existir.
	if after := pm(t, f, id); before-after != 1 {
		t.Errorf("o uso cobrou %d PM (%d → %d), quer 1", before-after, before, after)
	}
}

// ARMA GUARDADA NÃO DANÇA, e quem recusa é o SERVIDOR.
//
// A tela não oferece o gesto — o encanto da arma na mochila não entra no acervo
// —, e travar na UI é UX: a fronteira é o `UsePower`, que pergunta a POSSE. Sem
// isso, um POST montado à mão cobraria 1 PM e anunciaria um gesto que a arma
// guardada não faz.
func TestAStowedWeaponOffersNoEnchantPower(t *testing.T) {
	f, id := fighterFixture(t)
	sword := itemSemeia(t, f, id, "espada-longa", "Espada longa", "")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), sword, []string{"encanto-dancarina"},
	); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}

	// O CONTROLE da ausência é o Durão: o guerreiro de nível 3 o tem, e a
	// lista que não acha a linha dele não está sendo lida.
	panel := powersTabOf(t, powerScreen(t, f, id))
	if powerListRowOf(panel, "Durão") == "" {
		t.Fatalf("o controle já estava errado: o acervo não tem o Durão do guerreiro:\n%s", panel)
	}
	if row := powerListRowOf(panel, "Dançarina"); row != "" {
		t.Errorf("o encanto da arma GUARDADA entrou no acervo:\n%s", row)
	}
	before := pm(t, f, id)
	refusal := powerCommand(t, f, id, "usa/item.encanto-dancarina", "")
	if !strings.Contains(refusal, "não tem este poder") {
		t.Errorf("a recusa não diz que o poder não é dele: %q", refusal)
	}
	if after := pm(t, f, id); after != before {
		t.Errorf("a recusa cobrou assim mesmo: %d → %d", before, after)
	}
	// CONTROLE: com a MESMA espada na mão o gesto passa — sem isto, uma recusa
	// que barrasse todo encanto passaria verde aqui.
	if refusal := bagCommand(t, f, id, fmt.Sprintf("itens/%d/equipa/wielded", sword)); refusal != "" {
		t.Fatalf("empunhar a espada foi recusado: %q", refusal)
	}
	if refusal := powerCommand(t, f, id, "usa/item.encanto-dancarina", ""); refusal != "" {
		t.Fatalf("o encanto da arma EMPUNHADA foi recusado: %q", refusal)
	}
}
