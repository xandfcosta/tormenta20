package api

import (
	"regexp"
	"strings"
	"testing"
)

// A CONTA MOSTRA O QUE NÃO ENTROU. Duas armaduras vestidas não somam na Defesa ("efeitos de itens [...]
// não [acumulam] entre si", p226): a que perde fica na conta da aba Combate, apagada e riscada, dizendo
// com quem não empilhou — em vez de sumir e deixar a pessoa procurando o +1 que vestiu.
func TestTheDefenceShowsTheArmourThatDidNotStack(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Guerreiro das duas armaduras", "Guerreiro", 3, 0, 0)
	itemSemeia(t, f, id, "armadura-acolchoada", "Armadura acolchoada", "vested")
	itemSemeia(t, f, id, "armadura-couro", "Armadura de couro", "vested")

	page := combatScreen(t, f, id)
	row := regexp.MustCompile(`(?s)<dt data-muted[^>]*>\s*Armadura acolchoada.*?</dt>\s*<dd[^>]*data-dropped[^>]*>`)
	if !row.MatchString(page) {
		t.Fatal("a Defesa devia mostrar a armadura acolchoada apagada e marcada como fora do total")
	}
	if !strings.Contains(page, "não entra: não empilha com Armadura de couro") {
		t.Error("a linha da acolchoada devia dizer com quem ela não empilhou")
	}
}
