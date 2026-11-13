package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// A MAGIA DO HARMONIZADO (p165): "Escolha uma magia. Seu custo diminui em –1 PM". Quem escolhe é o jogador, na
// folha do esotérico, buscando entre todas as magias do livro (decisão do dono), e trocar é escolher de novo.

const harmonizedImprovement = "melhoria-harmonizado-esoterico"

// harmonizedWand é uma varinha arcana empunhada, com o Harmonizado aplicado pela Mochila.
func harmonizedWand(t *testing.T, f sceneFixture) int64 {
	t.Helper()
	wand := itemSemeia(t, f, f.charID, "varinha-arcana", "Varinha arcana", "wielded")
	if refusal := improvements(t, f, f.charID, wand, `{"item_improvements":["`+harmonizedImprovement+`"]}`); refusal != "" {
		t.Fatalf("aplicar o Harmonizado foi recusado: %q", refusal)
	}
	return wand
}

func harmonizedSpellPath(item int64, spell string) string {
	return fmt.Sprintf("itens/%d/melhorias/%s/magia/%s", item, harmonizedImprovement, spell)
}

func storedImprovementChoice(t *testing.T, f sceneFixture, item int64) string {
	t.Helper()
	var choice sql.NullString
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT choice FROM character_item_improvement_choices WHERE itemId = ? AND improvementId = ?",
		item, harmonizedImprovement).Scan(&choice); err != nil && err != sql.ErrNoRows {
		t.Fatalf("ler a escolha da melhoria do item %d: %v", item, err)
	}
	return choice.String
}

// itemSheetSearching é a folha do item com a busca de magia da melhoria preenchida.
func itemSheetSearching(t *testing.T, f sceneFixture, item int64, search string) string {
	t.Helper()
	signals := url.QueryEscape(`{"improvement_spell_search":"` + search + `"}`)
	page := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=bag&datastar=%s", f.charID, signals), "").Body.String()
	start := strings.Index(page, fmt.Sprintf(`id="item-%d"`, item))
	if start < 0 {
		t.Fatalf("a página não traz a folha do item %d", item)
	}
	return page[start : start+strings.Index(page[start:], "</dialog>")]
}

func TestTheHarmonizedSheetSearchesTheWholeBookAndChoosesTheSpell(t *testing.T) {
	f := newSceneFixture(t)
	wand := harmonizedWand(t, f)
	choose := func(spell string) string {
		return "/personagens/" + fmt.Sprint(f.charID) + "/" + harmonizedSpellPath(wand, spell)
	}

	sheet := itemSheetSearching(t, f, wand, "")
	if !strings.Contains(sheet, "Magia do Harmonizado") || !strings.Contains(sheet, "p165") {
		t.Fatal("a folha da varinha harmonizada não pede a magia do Harmonizado, com a p165")
	}
	if !strings.Contains(sheet, "sem ela, o Harmonizado não corta nenhuma") {
		t.Error("sem magia escolhida, a folha não diz que o Harmonizado não corta nada")
	}
	// Fora do grimório: a lista é o livro inteiro.
	if found := itemSheetSearching(t, f, wand, "bola de"); !strings.Contains(found, choose("bola-de-fogo")) {
		t.Error("buscar \"bola de\" não oferece Bola de Fogo, que não está no grimório")
	}
	if found := itemSheetSearching(t, f, wand, "bola de"); strings.Contains(found, choose("luz")) {
		t.Error("buscar \"bola de\" oferece Luz, que não casa com a busca")
	}

	if refusal := bagCommand(t, f, f.charID, harmonizedSpellPath(wand, "bola-de-fogo")); refusal != "" {
		t.Fatalf("escolher Bola de Fogo foi recusado: %q", refusal)
	}
	if got := storedImprovementChoice(t, f, wand); got != "bola-de-fogo" {
		t.Errorf("a magia gravada foi %q; want bola-de-fogo", got)
	}
	if !strings.Contains(itemSheetSearching(t, f, wand, ""), "Bola de Fogo custa 1 PM a menos") {
		t.Error("com a magia escolhida, a folha não diz qual custa menos")
	}
	// A busca acha a escolhida também: "bola" não diz "nenhuma" com a Bola de Fogo escolhida.
	if found := itemSheetSearching(t, f, wand, "bola de"); strings.Contains(found, "Nenhuma magia") || !strings.Contains(found, choose("bola-de-fogo")) {
		t.Error("buscar \"bola de\" com a Bola de Fogo escolhida não a lista")
	}
	if refusal := bagCommand(t, f, f.charID, harmonizedSpellPath(wand, "luz")); refusal != "" {
		t.Fatalf("trocar para Luz foi recusado: %q", refusal)
	}
	if got := storedImprovementChoice(t, f, wand); got != "luz" {
		t.Errorf("depois de trocar, a magia gravada foi %q; want luz", got)
	}
}

func TestTheHarmonizedSpellRefusesWhatDoesNotFit(t *testing.T) {
	f := newSceneFixture(t)
	wand := harmonizedWand(t, f)
	if refusal := bagCommand(t, f, f.charID, harmonizedSpellPath(wand, "abracadabra")); !strings.Contains(refusal, "magia") {
		t.Errorf("uma magia que não existe deu %q; want a recusa", refusal)
	}
	bare := itemSemeia(t, f, f.charID, "varinha-arcana", "Varinha arcana", "")
	if refusal := bagCommand(t, f, f.charID, harmonizedSpellPath(bare, "luz")); !strings.Contains(refusal, "Harmonizado") {
		t.Errorf("escolher a magia numa varinha sem o Harmonizado deu %q; want a recusa", refusal)
	}
	path := fmt.Sprintf("itens/%d/melhorias/melhoria-delicada/magia/luz", wand)
	if refusal := bagCommand(t, f, f.charID, path); !strings.Contains(refusal, "não pede magia") {
		t.Errorf("escolher magia para uma melhoria que não pede deu %q; want a recusa", refusal)
	}
}

// Tirar o Harmonizado leva a magia junto: aplicado de novo, ele pede a escolha outra vez.
func TestRemovingTheHarmonizedDropsItsSpell(t *testing.T) {
	f := newSceneFixture(t)
	wand := harmonizedWand(t, f)
	if refusal := bagCommand(t, f, f.charID, harmonizedSpellPath(wand, "luz")); refusal != "" {
		t.Fatalf("escolher Luz foi recusado: %q", refusal)
	}
	if refusal := improvements(t, f, f.charID, wand, `{"item_improvements":[]}`); refusal != "" {
		t.Fatalf("tirar o Harmonizado foi recusado: %q", refusal)
	}
	if got := storedImprovementChoice(t, f, wand); got != "" {
		t.Errorf("sem o Harmonizado, a magia %q ficou gravada", got)
	}
}

// O corte chega à conjuração: Bola de Fogo é de 2º círculo, 3 PM (Tabela 4-1, p170), menos 1.
func TestTheHarmonizedSpellCastsForOnePmLess(t *testing.T) {
	f := newSceneFixture(t)
	wand := harmonizedWand(t, f)
	learnSpell(t, f, "bola-de-fogo")
	if refusal := bagCommand(t, f, f.charID, harmonizedSpellPath(wand, "bola-de-fogo")); refusal != "" {
		t.Fatalf("escolher Bola de Fogo foi recusado: %q", refusal)
	}
	before := mpOf(t, f.s, f.charID)
	if refusal := castFromTheSheet(t, f, "bola-de-fogo"); refusal != "" {
		t.Fatalf("conjurar Bola de Fogo foi recusado: %q", refusal)
	}
	if spent := before - mpOf(t, f.s, f.charID); spent != 2 {
		t.Errorf("Bola de Fogo com a varinha harmonizada gastou %d PM; want 2", spent)
	}
	// Quem corta é o item, como em toda melhoria: a nota diz o nome dele.
	if !strings.Contains(spellScreen(t, f, f.charID), "Varinha arcana tira 1 PM deste custo") {
		t.Error("o diálogo de conjurar não diz que a varinha tira 1 PM da Bola de Fogo")
	}
}
