package api

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// A ABA MAGIAS DA FICHA PARALELA (ALE-473), pelo mockup aprovado: a CD e o
// atributo uma vez no topo, o grimório agrupado por círculo, e a linha sem o
// texto da regra — tocar no nome abre a folha da magia, com a regra, os
// aprimoramentos, o custo e o Esquecer no pé, longe do Conjurar.

// grimoireOfTwo é a arcanista do 9º nível com uma magia de 1º círculo e uma de
// 2º: o agrupamento só aparece com dois círculos na tela.
func grimoireOfTwo(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f, id := arcanista(t)
	for _, learned := range []string{"bola-de-fogo", "seta-infalivel-de-talude"} {
		if got := spell(t, f, id, "aprende/"+learned); got != http.StatusOK {
			t.Fatalf("aprender %s: status %d", learned, got)
		}
	}
	return f, id
}

// ballSheet é o id da folha da Bola de Fogo.
const ballSheet = "spell-bola-de-fogo"

func spellSheetOf(t *testing.T, page, id string) string {
	t.Helper()
	dialog := regexp.MustCompile(`(?s)<dialog[^>]*id="` + id + `".*?</dialog>`).FindString(page)
	if dialog == "" {
		t.Fatalf("a aba Magias paralela não desenhou a folha %q", id)
	}
	return dialog
}

func TestTheParallelSpellsTabGroupsTheSpellbookByCircle(t *testing.T) {
	f, id := grimoireOfTwo(t)
	page := parallelSheet(t, f, id, "spells")

	if strings.Contains(page, "ainda está na ficha de hoje") {
		t.Fatal("a aba Magias paralela ainda leva à ficha de hoje")
	}
	// Inteligência 4 no 9º nível: 10 + 4 + 4 (p173).
	if !strings.Contains(page, "CD <b>18</b>") || !strings.Contains(page, "Inteligência") {
		t.Error("o topo da aba não diz a CD 18 e o atributo Inteligência")
	}
	first, second := strings.Index(page, ">1º círculo<"), strings.Index(page, ">2º círculo<")
	seta, ball := strings.Index(page, ">Seta Infalível de Talude<"), strings.Index(page, ">Bola de Fogo<")
	if first < 0 || second < 0 || !(first < seta && seta < second && second < ball) {
		t.Errorf("o grimório não sai agrupado por círculo: 1º em %d, Seta em %d, 2º em %d, Bola em %d", first, seta, second, ball)
	}
	// A execução e o alcance saem como a mesa fala, e não como o id do catálogo.
	if !strings.Contains(page, "Padrão · médio") {
		t.Error("a linha da Bola de Fogo não diz \"Padrão · médio\"")
	}
	// A Bola de Fogo tem aprimoramentos: o Conjurar da linha abre a folha.
	opens := `commandfor="` + ballSheet + `"`
	if !regexp.MustCompile(`<button[^>]*` + opens + `[^>]*aria-label="Conjurar Bola de Fogo"|<button[^>]*aria-label="Conjurar Bola de Fogo"[^>]*` + opens).MatchString(page) {
		t.Error("o Conjurar da linha da Bola de Fogo não abre a folha dela")
	}
	if grimoire := regexp.MustCompile(`[\w:-]*grimorio[\w-]*`).FindAllString(sceneShell.FindString(page), 3); len(grimoire) > 0 {
		t.Errorf("a aba Magias paralela vestiu o grimório: %v", grimoire)
	}
}

func TestTheParallelSpellSheetHasTheRuleTheAugmentsAndForgetApart(t *testing.T) {
	f, id := grimoireOfTwo(t)
	page := parallelSheet(t, f, id, "spells")
	sheet := spellSheetOf(t, page, ballSheet)

	text := regexp.MustCompile(`(?s)<p class="ds-card-text">(.*?)</p>`).FindStringSubmatch(sheet)
	if text == nil || len(html.UnescapeString(text[1])) < 20 {
		t.Fatalf("a folha da Bola de Fogo não tem o texto da regra:\n%s", sheet)
	}
	if n := strings.Count(page, text[1]); n != 1 {
		t.Errorf("o texto da regra aparece %d vezes na página; a linha não devia mostrá-lo", n)
	}
	if !strings.Contains(sheet, "2º círculo · Padrão · médio · instantânea · p") {
		t.Error("a folha não abre com o círculo, a execução, o alcance, a duração e a página")
	}
	// Três aprimoramentos, cada um com o seu controle.
	if n := len(regexp.MustCompile(`aria-label="Mais de |role="switch"[^>]*data-on:change="\$augment`).FindAllString(sheet, -1)); n != 3 {
		t.Errorf("a folha tem %d controles de aprimoramento; a Bola de Fogo tem 3", n)
	}
	foot := regexp.MustCompile(`(?s)<div class="ds-dialog-foot">.*`).FindString(sheet)
	forget := strings.Index(foot, "/magias/esquece/bola-de-fogo")
	cast := strings.Index(foot, "/magias/conjura/bola-de-fogo")
	if forget < 0 || cast < 0 || forget > cast {
		t.Errorf("o pé da folha não tem o Esquecer antes do Conjurar (esquecer em %d, conjurar em %d)", forget, cast)
	}
	if !strings.Contains(foot, "tempestade=1") {
		t.Error("os gestos da folha perdem a marca da ficha paralela")
	}
}

func TestTheParallelSpellsTabSearchesTheSpellbookByName(t *testing.T) {
	f, id := grimoireOfTwo(t)
	signals := url.QueryEscape(`{"grimoire_search":"bola"}`)
	rec := f.requests(t, f.player, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=spells&tempestade=1&datastar=%s", id, signals), "")
	page := rec.Body.String()

	if !strings.Contains(page, ">Bola de Fogo<") {
		t.Fatal("a busca \"bola\" perdeu a Bola de Fogo")
	}
	if strings.Contains(page, ">Seta Infalível de Talude<") {
		t.Error("a busca \"bola\" deixou a Seta Infalível de Talude na lista")
	}
}

// O APRENDER é do conjurador: o diálogo traz o catálogo e o gesto de aprender;
// quem não conjura não recebe nem o botão nem as 198 linhas.
func TestTheParallelLearnDialogIsOnlyForCasters(t *testing.T) {
	f, id := arcanista(t)
	page := parallelSheet(t, f, id, "spells")
	learn := spellSheetOf(t, page, "spell-learn")
	if !regexp.MustCompile(`data-on:click="[^"]*/magias/aprende/bola-de-fogo[^"]*tempestade=1`).MatchString(learn) {
		t.Error("o diálogo Aprender não oferece a Bola de Fogo")
	}
	if !regexp.MustCompile(`<button[^>]*commandfor="spell-learn"`).MatchString(page) {
		t.Error("a aba não tem o botão que abre o Aprender")
	}

	g := newSceneFixture(t)
	warrior := seedCharacterAtLevel(t, g.s, g.player, "Lia", "Guerreiro", 3, 0, 0)
	if strings.Contains(parallelSheet(t, g, warrior, "spells"), `id="spell-learn"`) {
		t.Error("o guerreiro recebeu o diálogo Aprender")
	}
}
