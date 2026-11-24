package api

import (
	"strings"
	"testing"
)

// O DADO DA MAGIA vai à faixa da mesa, como o disparo da Flamejante (ver GLOSSARY, **dado da magia**): conjurar rola
// o dado, e a faixa informa o total, o tipo, a resistência e a CD. Quem passou e o que se aplica é do mestre.

func TestTheCastSpellRollReachesTheWholeTable(t *testing.T) {
	f := umaMesaAoVivo(t)
	if refusal, _ := castWith(t, f, "bola-de-fogo", `{}`); refusal != "" {
		t.Fatalf("conjurar a Bola de Fogo foi recusado: %q", refusal)
	}
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	if cast == nil {
		t.Fatal("a conjuração não chegou à mesa")
	}
	if len(cast.Parts) != 1 || cast.Parts[0].Dice != "6d6" || cast.Parts[0].Kind != "fogo" {
		t.Fatalf("a Bola de Fogo rolou %+v; want 6d6 de fogo (p182)", cast.Parts)
	}
	if part := cast.Parts[0]; len(part.Rolls) != 6 || part.Total < 6 || part.Total > 36 || part.ByHand {
		t.Errorf("os 6d6 saíram %v = %d, à mão %v", part.Rolls, part.Total, part.ByHand)
	}
	if cast.Save != "Reflexos reduz à metade" || cast.DC == 0 {
		t.Errorf("a resistência saiu %q com CD %d", cast.Save, cast.DC)
	}
	for _, c := range []struct {
		who  string
		user int64
	}{{"o mestre", f.gm}, {"o jogador", f.player}} {
		page := f.requests(t, c.user, "GET", f.tableUrl(), "").Body.String()
		if !strings.Contains(page, "Bola de Fogo") || !strings.Contains(page, "Reflexos reduz à metade") {
			t.Errorf("%s não vê a Bola de Fogo na faixa da mesa", c.who)
		}
	}
}

// O DADO DA MESA é de primeira classe: o total rolado na mão chega à faixa, e o que os dados não tiram é recusado
// antes de cobrar.
func TestTheSpellRollTheTableRolledByHand(t *testing.T) {
	f := umaMesaAoVivo(t)
	if refusal, _ := castWith(t, f, "bola-de-fogo", `{"spell_hand_1":20}`); refusal != "" {
		t.Fatalf("a Bola de Fogo com 20 na mão foi recusada: %q", refusal)
	}
	if part := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast.Parts[0]; part.Total != 20 || !part.ByHand {
		t.Errorf("a faixa diz %d, à mão %v; want os 20 da mesa", part.Total, part.ByHand)
	}
	refusal, spent := castWith(t, f, "bola-de-fogo", `{"spell_hand_1":40}`)
	if !strings.Contains(refusal, "de 6 a 36") || spent != 0 {
		t.Errorf("os 40 na mão foram recusados com %q, gastando %d PM", refusal, spent)
	}
}

// A FOLHA DA MAGIA diz o dado que ela rola, parcela por parcela, com o campo do total rolado na mão ligado ao
// sinal da parcela: a Coluna de Chamas rola 6d6 de fogo e 6d6 de luz.
func TestTheSpellSheetSaysWhatItRollsAndTakesTheTablesTotal(t *testing.T) {
	f := newSceneFixture(t)
	learn(t, f, f.charID, "bola-de-fogo")
	page := spellScreen(t, f, f.charID)
	if !strings.Contains(page, "6d6 de fogo") {
		t.Error("a folha da Bola de Fogo não diz que rola 6d6 de fogo")
	}
	if !strings.Contains(page, `data-bind="spell_hand_1"`) {
		t.Error("a folha da Bola de Fogo não tem o campo do total rolado na mão")
	}
}
