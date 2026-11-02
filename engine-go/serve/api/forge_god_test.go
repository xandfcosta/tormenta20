package api

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// O deus é opcional, e a lista segue as classes.
func TestTheForgeOffersTheGodOfTheClasses(t *testing.T) {
	f := newSceneFixture(t)
	none := forgedSheet(t, f, forgedHero(t, f, func(url.Values) {}))
	if none.God != nil && *none.God != "" {
		t.Errorf("sem escolher, o herói nasceu devoto de %q", *none.God)
	}
	served := forgedSheet(t, f, forgedHero(t, f, func(answers url.Values) { answers.Set("god", "khalmyr") }))
	if served.God == nil || *served.God != "khalmyr" {
		t.Errorf("o deus escolhido não nasceu na ficha: %v", served.God)
	}

	blank := f.requests(t, f.player, http.MethodGet, "/personagens/nova", "").Body.String()
	if !strings.Contains(blank, `name="god"`) || strings.Contains(blank, `value="panteao"`) {
		t.Errorf("sem classe a lista é a dos vinte deuses, sem o Panteão:\n%s", blank)
	}
	cleric := postaAForja(t, f, f.player, "/personagens/nova/esboco",
		url.Values{"classes": {"Clérigo:1"}}).Body.String()
	if !strings.Contains(cleric, `value="panteao"`) {
		t.Error("o clérigo não vê o Panteão")
	}
	if row := pickRow(t, blank, "god"); !strings.Contains(row, "Nenhum") {
		t.Errorf("a linha do deus não nasce em Nenhum:\n%s", row)
	}
}

// A linha do deus não promete o que a devoção não dá: o Panteão não recebe poder concedido (p57).
func TestThePickRowSaysWhenTheDevotionGrantsNoPower(t *testing.T) {
	f := newSceneFixture(t)
	body := postaAForja(t, f, f.player, "/personagens/nova/esboco",
		url.Values{"classes": {"Clérigo:1"}, "god": {"panteao"}}).Body.String()
	row := pickRow(t, body, "god")
	if !strings.Contains(row, "não recebe poder concedido") || strings.Contains(row, "se escolhe na ficha") {
		t.Errorf("a linha do Panteão promete um poder que ele não dá:\n%s", row)
	}
	khalmyr := postaAForja(t, f, f.player, "/personagens/nova/esboco",
		url.Values{"classes": {"Clérigo:1"}, "god": {"khalmyr"}}).Body.String()
	if row := pickRow(t, khalmyr, "god"); !strings.Contains(row, "se escolhe na ficha") {
		t.Errorf("a linha de Khalmyr não diz onde se escolhe o poder:\n%s", row)
	}
}

// Tenebra servia até entrar o paladino: o deus continua marcado, e a folha diz o que houve.
func TestAGodTheClassRefusesStaysAndIsNamed(t *testing.T) {
	f := newSceneFixture(t)
	body := postaAForja(t, f, f.player, "/personagens/nova/esboco", url.Values{
		"classes": {"Guerreiro:2", "Paladino:1"}, "god": {"tenebra"},
	}).Body.String()
	if !regexp.MustCompile(`role="alert"[^>]*>[^<]*Paladino não serve a Tenebra`).MatchString(body) {
		t.Errorf("a folha não disse que o paladino recusa Tenebra:\n%s", body)
	}
	if !regexp.MustCompile(`<input[^>]*name="god"[^>]*value="tenebra"[^>]*checked`).MatchString(body) {
		t.Errorf("o deus sumiu da folha em silêncio:\n%s", body)
	}
	answers := aFolhaPreenchida()
	answers.Del("class")
	answers["classes"] = []string{"Guerreiro:2", "Paladino:1"}
	answers.Set("god", "tenebra")
	rec := postaAForja(t, f, f.player, "/personagens/nova", answers)
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "Tenebra") {
		t.Errorf("a forja aceitou o paladino de Tenebra: %d", rec.Code)
	}
}

// A lista está em ordem de nome, e Tauron cai no lugar dele, entre Tanna-Toh e Tenebra.
func TestTheForgeListsTauronInOrder(t *testing.T) {
	f := newSceneFixture(t)
	blank := f.requests(t, f.player, http.MethodGet, "/personagens/nova", "").Body.String()
	tanna, tauron, tenebra := strings.Index(blank, `value="tanna-toh"`), strings.Index(blank, `value="tauron"`), strings.Index(blank, `value="tenebra"`)
	if tauron < 0 || tanna > tauron || tauron > tenebra {
		t.Errorf("Tauron fora de ordem: Tanna-Toh em %d, Tauron em %d, Tenebra em %d", tanna, tauron, tenebra)
	}
	hero := forgedSheet(t, f, forgedHero(t, f, func(answers url.Values) { answers.Set("god", "tauron") }))
	if hero.God == nil || *hero.God != "tauron" {
		t.Errorf("Tauron não nasceu na ficha: %v", hero.God)
	}
	cleric := postaAForja(t, f, f.player, "/personagens/nova/esboco",
		url.Values{"classes": {"Clérigo:1"}}).Body.String()
	if strings.Contains(cleric, `value="tauron"`) {
		t.Error("o clérigo vê Tauron, e ele não está na lista do livro")
	}
}
