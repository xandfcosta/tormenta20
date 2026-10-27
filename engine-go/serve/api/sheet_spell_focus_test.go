package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// O FOCO EM MAGIA (p131, ALE-510): "Escolha uma magia que possa lançar. Seu
// custo diminui em –1 PM (cumulativo com outras reduções de custo). Você pode
// escolher este poder outras vezes para magias diferentes."

const spellFocusPath = "escolha/foco-em-magia/"

func learnSpell(t *testing.T, f sceneFixture, spellID string) {
	t.Helper()
	learnURL := fmt.Sprintf("/personagens/%d/magias/aprende/%s?tab=spells", f.charID, spellID)
	if rec := f.requests(t, f.player, http.MethodPost, learnURL, ""); rec.Code != http.StatusOK {
		t.Fatalf("aprender %q respondeu %d: %s", spellID, rec.Code, rec.Body.String())
	}
}

func castFromTheSheet(t *testing.T, f sceneFixture, spellID string) string {
	t.Helper()
	castURL := fmt.Sprintf("/personagens/%d/magias/conjura/%s?tab=spells", f.charID, spellID)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, castURL, "").Body.String())
}

func TestTheSpellFocusCastsTheChosenSpellForOnePmLess(t *testing.T) {
	f := newSceneFixture(t)
	learnSpell(t, f, "bola-de-fogo")
	if refusal := powerCommand(t, f, f.charID, spellFocusPath+"bola-de-fogo", ""); refusal != "" {
		t.Fatalf("escolher Bola de Fogo no Foco em Magia foi recusado: %q", refusal)
	}
	before := mpOf(t, f.s, f.charID)
	if refusal := castFromTheSheet(t, f, "bola-de-fogo"); refusal != "" {
		t.Fatalf("conjurar Bola de Fogo foi recusado: %q", refusal)
	}
	// 2º círculo: 3 PM (Tabela 4-1, p170), menos 1.
	if spent := before - mpOf(t, f.s, f.charID); spent != 2 {
		t.Errorf("Bola de Fogo com o Foco em Magia gastou %d PM; want 2", spent)
	}
	if !strings.Contains(spellScreen(t, f, f.charID), "Foco em Magia tira 1 PM deste custo") {
		t.Error("o diálogo de conjurar não diz que o Foco em Magia tira 1 PM")
	}
}

// "uma magia que possa lançar", e "para magias diferentes".
func TestTheSpellFocusRefusesASpellOutOfTheGrimoireAndARepeat(t *testing.T) {
	f := newSceneFixture(t)
	if refusal := powerCommand(t, f, f.charID, spellFocusPath+"bola-de-fogo", ""); refusal == "" {
		t.Error("o Foco em Bola de Fogo passou sem a magia no grimório")
	}
	learnSpell(t, f, "bola-de-fogo")
	if refusal := powerCommand(t, f, f.charID, spellFocusPath+"bola-de-fogo", ""); refusal != "" {
		t.Fatalf("escolher Bola de Fogo, aprendida, foi recusado: %q", refusal)
	}
	if refusal := powerCommand(t, f, f.charID, spellFocusPath+"bola-de-fogo", ""); refusal == "" {
		t.Error("o Foco em Bola de Fogo passou duas vezes; a p131 diz magias diferentes")
	}
}

// A lista oferece as magias do grimório, e só elas.
func TestTheSpellFocusOffersTheGrimoire(t *testing.T) {
	f := newSceneFixture(t)
	learnSpell(t, f, "bola-de-fogo")
	page := powerScreen(t, f, f.charID)
	if !strings.Contains(page, "/escolha/foco-em-magia/bola-de-fogo") {
		t.Error("a lista de poderes não oferece Bola de Fogo, aprendida, ao Foco em Magia")
	}
	if strings.Contains(page, "/escolha/foco-em-magia/relampago") {
		t.Error("a lista oferece Relâmpago, que não está no grimório")
	}
}

// "uma magia que possa LANÇAR": a do grimório acima do círculo que a ficha
// alcança fica de fora. O Arcanista 8 lança até o 2º círculo (p37), e o
// Teletransporte é do 3º.
func TestTheSpellFocusLeavesOutTheSpellTheSheetCannotCastYet(t *testing.T) {
	f := newSceneFixture(t)
	learnSpell(t, f, "teletransporte")
	if refusal := powerCommand(t, f, f.charID, spellFocusPath+"teletransporte", ""); !strings.Contains(refusal, "círculo") {
		t.Errorf("o Foco em Teletransporte, de 3º círculo, deu %q; want a recusa pelo círculo", refusal)
	}
	if strings.Contains(powerScreen(t, f, f.charID), "/escolha/foco-em-magia/teletransporte") {
		t.Error("a lista oferece Teletransporte, que o Arcanista 8 ainda não lança")
	}
}
