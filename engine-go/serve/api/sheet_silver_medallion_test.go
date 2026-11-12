package api

import (
	"strings"
	"testing"
)

// O MEDALHÃO DE PRATA "diminui em –1 PM o custo de magias de alcance pessoal" (p160). O Campo de Força é
// pessoal, de 2º círculo: 3 PM (Tabela 4-1, p170), e com o medalhão sai por 2. A Luz, de alcance curto,
// paga cheio. E o medalhão não mexe no limite de PM.

func TestTheSilverMedallionCutsOnlyPersonalSpells(t *testing.T) {
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "medalhao-de-prata", "Medalhão de prata", "wielded")
	for _, c := range []struct {
		spell string
		cost  int
		cut   bool
	}{
		{"campo-de-forca", 2, true},
		{"luz", 1, false},
	} {
		learnSpell(t, f, c.spell)
		page := spellScreen(t, f, f.charID)
		cutLine := spellRowOf(page, c.spell)
		if got := strings.Contains(cutLine, "Medalhão de prata tira 1 PM deste custo"); got != c.cut {
			t.Errorf("%s: a nota do medalhão apareceu = %v, want %v", c.spell, got, c.cut)
		}
		before := mpOf(t, f.s, f.charID)
		if refusal := castFromTheSheet(t, f, c.spell); refusal != "" {
			t.Fatalf("conjurar %s: %q", c.spell, refusal)
		}
		if spent := before - mpOf(t, f.s, f.charID); int(spent) != c.cost {
			t.Errorf("%s custou %d PM, want %d", c.spell, spent, c.cost)
		}
	}
	if limit := pmLimitOf(t, f, f.charID); limit != 8 {
		t.Errorf("o medalhão não sobe o limite de PM: o arcanista de 8º nível devia ter 8, e tem %d", limit)
	}
}

// spellRowOf é o pedaço da página que fala desta magia: a folha dela, do id até o fim do diálogo.
func spellRowOf(page, spellID string) string {
	start := strings.Index(page, `id="spell-`+spellID+`"`)
	if start < 0 {
		return ""
	}
	rest := page[start:]
	if end := strings.Index(rest, "</dialog>"); end >= 0 {
		return rest[:end]
	}
	return rest
}
