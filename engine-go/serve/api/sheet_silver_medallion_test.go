package api

import (
	"context"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
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

// A nota só nomeia quem CORTA: com uma redução de −2 valendo em toda magia, o −1 do medalhão perde a
// disputa da p226, e a linha do Campo de Força não pode prometer que ele tira 1 PM.
func TestTheSpellCutNoteLeavesOutTheCutThatLost(t *testing.T) {
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "medalhao-de-prata", "Medalhão de prata", "wielded")
	if _, err := f.s.sceneCore().Queries().CreateActiveEffect(context.Background(),
		sqlcgen.CreateActiveEffectParams{
			Characterid: f.charID, Catalogid: "corte-geral", Scope: "scene",
			Modifiers: `[{"target":{"k":"pmCost"},"amount":-2,"bonusType":"untyped"}]`, Createdat: dbvalue.NowISO(),
		}); err != nil {
		t.Fatalf("semear o corte geral: %v", err)
	}
	learnSpell(t, f, "campo-de-forca")
	if row := spellRowOf(spellScreen(t, f, f.charID), "campo-de-forca"); strings.Contains(row, "Medalhão de prata tira") {
		t.Error("a linha do Campo de Força diz que o medalhão tira 1 PM, e ele perdeu para o corte de −2")
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
