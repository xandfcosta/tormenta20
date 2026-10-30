package api

import (
	"html"
	"regexp"
	"strings"
	"testing"
)

// A ABA PODERES DA FICHA (ALE-473): a linha não mostra o texto da
// regra, e tocar no nome abre a folha com a regra inteira (decisão do dono). O
// texto existe na página UMA vez — dentro da folha daquele poder.
func TestTheParallelPowersTabKeepsTheRuleTextInThePowerSheet(t *testing.T) {
	f, id := barbaro(t, 5)
	page := parallelSheet(t, f, id, "abilities")

	var instinct string
	for _, d := range regexp.MustCompile(`(?s)<dialog[^>]*id="power-[^"]*".*?</dialog>`).FindAllString(page, -1) {
		if strings.Contains(d, ">Instinto Selvagem") {
			instinct = d
		}
	}
	if instinct == "" {
		t.Fatal("a aba não tem a folha do Instinto Selvagem")
	}
	text := regexp.MustCompile(`(?s)<p class="ds-card-text">(.*?)</p>`).FindStringSubmatch(instinct)
	if text == nil || len(html.UnescapeString(text[1])) < 20 {
		t.Fatalf("a folha do Instinto Selvagem não tem o texto da regra:\n%s", instinct)
	}
	if n := strings.Count(page, text[1]); n != 1 {
		t.Errorf("o texto da regra aparece %d vezes na página; a linha não devia mostrá-lo", n)
	}

	// A Fúria é postura: a linha tem o gesto de ativar.
	if !strings.Contains(page, `aria-label="Ativar Fúria"`) {
		t.Error("a Fúria não tem o gesto de ativar na aba")
	}
	// O ESCOLHER ABRE O DIÁLOGO DE ESCOLHAS.
	if !regexp.MustCompile(`<button[^>]*commandfor="power-choices"`).MatchString(page) {
		t.Error("o Escolher da aba não abre o diálogo de escolhas")
	}
}
