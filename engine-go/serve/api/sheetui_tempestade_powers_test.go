package api

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"testing"
)

// A ABA PODERES DA FICHA PARALELA (ALE-473): a linha não mostra o texto da
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
		t.Fatal("a aba paralela não tem a folha do Instinto Selvagem")
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
		t.Error("a Fúria não tem o gesto de ativar na aba paralela")
	}
	// ESCOLHER AINDA É DA FICHA DE HOJE: o link sai declarado.
	if !regexp.MustCompile(fmt.Sprintf(`<a[^>]*href="/personagens/%d\?tab=abilities"[^>]*data-sheet-of-today|<a[^>]*data-sheet-of-today[^>]*href="/personagens/%d\?tab=abilities"`, id, id)).MatchString(page) {
		t.Error("o Escolher da aba paralela não leva à ficha de hoje")
	}
}
