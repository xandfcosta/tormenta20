package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"t20engine/domain/catalog"
	"t20engine/infra/db/sqlcgen"
)

// A MAGIA ACIMA DO CÍRCULO QUE A FICHA ALCANÇA não se aprende nem se conjura: aprende-se a magia "de um tipo
// e círculo que possa lançar com aquela classe" (p170). O arcanista de 8º nível alcança o 2º círculo — o 3º vem no 9º nível —, e o Desintegrar é
// do 4º. Antes, aprender não conferia nada, e conjurar só conferia o PM: o limite libera o custo mínimo.
func TestTheSpellAboveTheCircleIsNeitherLearnedNorCast(t *testing.T) {
	f := newSceneFixture(t)

	learnURL := fmt.Sprintf("/personagens/%d/magias/aprende/desintegrar?tab=spells", f.charID)
	refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, learnURL, "").Body.String())
	if !strings.Contains(refusal, "círculo") {
		t.Errorf("aprender o Desintegrar (4º círculo) com o arcanista de 8º foi %q; want a recusa pelo círculo", refusal)
	}
	// O controle: a magia do 2º círculo se aprende.
	learnSpell(t, f, "bola-de-fogo")

	// A magia que JÁ está no grimório, gravada antes da regra, fica, e conjurar recusa.
	if _, err := f.s.queries.CreateSpell(context.Background(), sqlcgen.CreateSpellParams{
		Characterid: f.charID, Catalogspellid: "desintegrar",
	}); err != nil {
		t.Fatalf("gravar o Desintegrar no grimório: %v", err)
	}
	before := mpOf(t, f.s, f.charID)
	if refusal := castFromTheSheet(t, f, "desintegrar"); !strings.Contains(refusal, "círculo") {
		t.Errorf("conjurar o Desintegrar foi %q; want a recusa pelo círculo", refusal)
	}
	if after := mpOf(t, f.s, f.charID); after != before {
		t.Errorf("a recusa cobrou PM: %d → %d", before, after)
	}
}

// O APRIMORAMENTO DE CÍRCULO ALTO se libera pela classe que lança a magia (p171): "você deve cumprir o requisito
// com a classe com a qual usa a magia". O Arcanista 9 / Clérigo 1 alcança o 3º círculo nas arcanas e só o 1º na
// Orientação, que é do clérigo. Antes, o círculo saía do maior entre as classes, e o aprimoramento de 2º passava.
func TestTheAugmentCircleComesFromTheClassThatCastsTheSpell(t *testing.T) {
	s := newCastServer(t)
	owner := seedUser(t, s, "multiclasse@t20.local")
	char := seedCaster(t, s, owner, "Arcanista", 9, 30, "orientacao")
	seedClasse(t, s, char, "Clérigo", 1)

	err := castSpell(t, s, owner, char, "orientacao", `{"augments":[{"augmentIndex":0,"stacks":1}]}`)
	if err == nil || !strings.Contains(err.Error(), "2º círculo") {
		t.Errorf("o aprimoramento de 2º círculo na Orientação do Clérigo 1 deu %v; want a recusa pelo círculo", err)
	}
}

// A ABA MAGIAS DIZ O QUE O SERVIDOR RECUSA, em vez de deixar o gesto bater na recusa. No Aprender, a magia
// acima do círculo aparece trancada, com a frase, como o aprimoramento de círculo alto; a de fora da lista
// se aprende, com o aviso. No grimório, a magia gravada acima do círculo fica, com o Conjurar trancado.
func TestTheSpellsTabLocksWhatTheCircleRefusesAndWarnsWhatTheListLeavesOut(t *testing.T) {
	f := newSceneFixture(t)
	catalog := sheetTabWith(t, f, f.charID, "spells", `{"sheet_dialog":"spell-learn"}`)
	button := func(page, label string) string {
		t.Helper()
		found := regexp.MustCompile(`<button[^>]*aria-label="` + label + `"[^>]*>`).FindString(page)
		if found == "" {
			t.Fatalf("a aba não desenhou o botão %q", label)
		}
		return found
	}
	if !strings.Contains(button(catalog, "Aprender Desintegrar"), "disabled") {
		t.Error("o Aprender do Desintegrar (4º círculo) está de pé para o arcanista de 8º")
	}
	if !strings.Contains(catalog, "Desintegrar é do 4º círculo, e esta ficha alcança até o 2º") {
		t.Error("o Aprender não diz por que o Desintegrar está trancado")
	}
	if strings.Contains(button(catalog, "Aprender Bola de Fogo"), "disabled") {
		t.Error("o Aprender da Bola de Fogo (2º círculo) está trancado para o arcanista de 8º")
	}
	healing := regexp.MustCompile(`(?s)Curar Ferimentos</div>.*?</button>`).FindString(catalog)
	if !strings.Contains(healing, "fora da lista das suas classes") {
		t.Error("o Curar Ferimentos, divino, não avisa que está fora da lista do arcanista")
	}
	if strings.Contains(button(catalog, "Aprender Curar Ferimentos"), "disabled") {
		t.Error("o Curar Ferimentos está trancado; a lista só avisa, porque um poder pode ensinar a magia")
	}

	if _, err := f.s.queries.CreateSpell(context.Background(), sqlcgen.CreateSpellParams{
		Characterid: f.charID, Catalogspellid: "desintegrar",
	}); err != nil {
		t.Fatalf("gravar o Desintegrar no grimório: %v", err)
	}
	grimoire := spellScreen(t, f, f.charID)
	if !strings.Contains(button(grimoire, "Conjurar Desintegrar"), "disabled") {
		t.Error("o Conjurar do Desintegrar gravado acima do círculo está de pé")
	}
	if !strings.Contains(grimoire, "acima do seu círculo") {
		t.Error("a linha do Desintegrar não diz que ele está acima do círculo")
	}
}

// O APRIMORAMENTO "APENAS ARCANOS" da Luz (p197) não se escolhe na Luz do clérigo, que é divina: o servidor
// recusa, e a folha da magia o tranca com a marca do livro.
func TestTheArcaneOnlyAugmentIsRefusedOnTheClericsLight(t *testing.T) {
	light, _ := catalog.LookupSpell("luz")
	arcaneOnly := slices.IndexFunc(light.Augments, func(a catalog.Augment) bool { return a.ClassOnly == "arcanos" && a.RequiresCircle == nil })
	if arcaneOnly < 0 {
		t.Fatal("a Luz não tem aprimoramento apenas arcano sem círculo: o caso não mede nada")
	}
	s := newCastServer(t)
	owner := seedUser(t, s, "clerigo@t20.local")
	char := seedCaster(t, s, owner, "Clérigo", 5, 20, "luz")
	err := castSpell(t, s, owner, char, "luz", fmt.Sprintf(`{"augments":[{"augmentIndex":%d,"stacks":1}]}`, arcaneOnly))
	if err == nil || !strings.Contains(err.Error(), "apenas arcanos") {
		t.Errorf("o aprimoramento apenas arcano na Luz do clérigo deu %v; want a recusa", err)
	}
	if arcanist := seedCaster(t, s, owner, "Arcanista", 5, 20, "luz"); castSpell(t, s, owner, arcanist,
		"luz", fmt.Sprintf(`{"augments":[{"augmentIndex":%d,"stacks":1}]}`, arcaneOnly)) != nil {
		t.Error("o controle falhou: o arcanista não escolheu o aprimoramento apenas arcano da Luz")
	}
}

// Na folha da Luz do arcanista, os dois aprimoramentos "Apenas Divinos" (p197) aparecem trancados com a marca do
// livro, em vez de sumir, como o de círculo alto; o "Apenas Arcanos" fica livre.
func TestTheSpellSheetLocksTheAugmentOfTheOtherTradition(t *testing.T) {
	f := newSceneFixture(t)
	learnSpell(t, f, "luz")
	light := spellSheetOf(t, spellScreen(t, f, f.charID), "spell-luz")
	if got := strings.Count(light, "apenas divinos"); got != 2 {
		t.Errorf("a folha da Luz do arcanista tranca %d aprimoramentos apenas divinos; want 2", got)
	}
	if strings.Contains(light, "apenas arcanos") {
		t.Error("a folha da Luz do arcanista tranca um aprimoramento apenas arcano")
	}
}

// A MARCA DE TRADIÇÃO VEM ANTES DO CÍRCULO: no Arcanista 4, que ainda não alcança o 2º círculo, os dois
// aprimoramentos "Apenas Divinos" da Luz dizem "apenas divinos", e não "exige o 2º círculo" — o círculo
// prometeria que eles chegam no 5º nível, e na mão de um arcanista eles não chegam nunca.
func TestTheTraditionLockComesBeforeTheCircleLock(t *testing.T) {
	f := newSceneFixture(t)
	apprentice := seedCharacterAtLevel(t, f.s, f.player, "Aprendiz", "Arcanista", 4, 10, 5)
	learn(t, f, apprentice, "luz")
	light := spellSheetOf(t, spellScreen(t, f, apprentice), "spell-luz")
	if got := strings.Count(light, "apenas divinos"); got != 2 {
		t.Errorf("a folha da Luz do Arcanista 4 diz \"apenas divinos\" %d vezes; want 2", got)
	}
}
