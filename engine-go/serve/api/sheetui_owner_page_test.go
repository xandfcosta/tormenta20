package api

import (
	"strings"
	"testing"
)

// A PÁGINA NA FOLHA DO PODER é a do poder que a ficha tem. Uma ativação serve a mais de uma classe quando o livro
// dá a mesma habilidade às duas — a Esquiva Sobrenatural do bárbaro (p42) e a do ladino (p74) —, e a folha do
// ladino mostrava a página da ativação, que é a do bárbaro.
func TestThePowerSheetShowsThePageOfThePowerTheSheetHas(t *testing.T) {
	for _, c := range []struct {
		name, class string
		level       int64
		chosen      string
		activation  string
		want        string
	}{
		// O ladino a ganha no 4º nível; o bárbaro a escolhe como poder.
		{"o ladino", "Ladino", 4, "", "class.barbaro.esquiva-sobrenatural", "p74"},
		{"o bárbaro, dono da ativação", "Bárbaro", 2, "class.barbaro.esquiva-sobrenatural", "class.barbaro.esquiva-sobrenatural", "p42"},
		// A do druida é a p61; o catálogo dava a p21, que é a Empatia Selvagem do dahllan.
		{"o druida", "Druida", 1, "", "class.cacador.empatia-selvagem", "p61"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := classCharacterWithMpSpent(t, c.class, c.level, 0)
			if c.chosen != "" {
				seedPowerChoices(t, f.s, id, `["`+c.chosen+`"]`, "{}")
			}
			meta := powerSheetMetaOf(t, powerScreen(t, f, id), c.activation)
			if !strings.HasSuffix(meta, c.want) {
				t.Errorf("%s vê a folha de %s com %q; want a página %s", c.name, c.activation, meta, c.want)
			}
		})
	}
}

// powerSheetMetaOf é a primeira linha da folha do poder: a nota e a página.
func powerSheetMetaOf(t *testing.T, screen, activation string) string {
	t.Helper()
	_, sheet, found := strings.Cut(screen, `id="power-`+strings.ReplaceAll(activation, ".", "-")+`"`)
	if !found {
		t.Fatalf("a aba Poderes não tem a folha de %s", activation)
	}
	_, meta, found := strings.Cut(sheet, `<p class="ds-row-meta">`)
	if !found {
		t.Fatalf("a folha de %s não tem a linha da página", activation)
	}
	meta, _, _ = strings.Cut(meta, "</p>")
	return meta
}
