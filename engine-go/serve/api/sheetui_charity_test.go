package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// A CARIDADE NO GESTO DE USAR (ALE-647, fatia 3): "o custo de suas habilidades de paladino que tenham um aliado
// como alvo é reduzido em –1 PM" (p83). O gesto diz o alvo (`power_on_ally`), e o servidor cobra por ele: a
// Coragem de 2 PM custa 1 num aliado. O poder que não tem aliado como alvo recusa o pedido.
func TestTheCharityChargesOneLessWhenTheTargetIsAnAlly(t *testing.T) {
	spent := func(power, signals string) (int64, string) {
		f := newSceneFixture(t)
		asClass(t, f, "Paladino")
		withPowers(t, f, `["class.paladino.julgamento-divino-coragem","class.paladino.julgamento-divino-zelo","class.paladino.virtude-paladinesca-caridade"]`, "{}")
		before := mpOf(t, f.s, f.charID)
		rec := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/poderes/usa/%s", f.charID, power), signals)
		return before - mpOf(t, f.s, f.charID), sceneRefusal(rec.Body.String())
	}
	if got, refusal := spent("class.paladino.julgamento-coragem", `{}`); got != 2 || refusal != "" {
		t.Errorf("a Coragem no próprio paladino custou %d PM (recusa %q); want 2", got, refusal)
	}
	if got, refusal := spent("class.paladino.julgamento-coragem", `{"power_on_ally":true}`); got != 1 || refusal != "" {
		t.Errorf("a Coragem num aliado custou %d PM (recusa %q); want 1, com a Caridade", got, refusal)
	}
	if got, refusal := spent("class.paladino.julgamento-zelo", `{"power_on_ally":true}`); got != 0 || !strings.Contains(refusal, "aliado") {
		t.Errorf("o Zelo, que marca um alvo e não tem aliado como alvo, pedido num aliado gastou %d PM com a recusa %q; want recusado", got, refusal)
	}
}

// COM 1 PM, a Coragem de 2 PM sai num aliado (1 PM) e é recusada no próprio paladino: a decisão de usar confere o
// PM contra o preço do uso, com o corte.
func TestTheCharityLetsTheLastPointPayAnAllyCourage(t *testing.T) {
	f := newSceneFixture(t)
	asClass(t, f, "Paladino")
	withPowers(t, f, `["class.paladino.julgamento-divino-coragem","class.paladino.virtude-paladinesca-caridade"]`, "{}")
	arrangePools(t, f.s, f.charID, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.MpCurrent = 1
		return pools, nil
	})
	use := func(signals string) string {
		rec := f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/poderes/usa/class.paladino.julgamento-coragem", f.charID), signals)
		return sceneRefusal(rec.Body.String())
	}
	if refusal := use(`{}`); refusal == "" {
		t.Error("com 1 PM, a Coragem de 2 PM no próprio paladino saiu")
	}
	if refusal := use(`{"power_on_ally":true}`); refusal != "" || mpOf(t, f.s, f.charID) != 0 {
		t.Errorf("com 1 PM, a Coragem num aliado foi recusada (%q) ou não cobrou o último PM (sobrou %d)", refusal, mpOf(t, f.s, f.charID))
	}
}

// A TELA DA CARIDADE (decisão do dono): na Coragem, o "Usar" da aba Poderes vira dois botões com o preço de cada
// um; na Cura pelas Mãos, o diálogo de degraus — o da aba e o das Ações da mesa — ganha a escolha do alvo. Sem a
// Caridade, nada muda. As Ações da mesa não dão gesto ao instantâneo sem degrau, e a Coragem continua sem.
func TestTheCharityOffersTheAllyTargetWithItsPrice(t *testing.T) {
	for _, c := range []struct {
		name, powers string
		charity      bool
	}{
		{"com a Caridade", `["class.paladino.julgamento-divino-coragem","class.paladino.virtude-paladinesca-caridade"]`, true},
		{"sem a Caridade", `["class.paladino.julgamento-divino-coragem"]`, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := paladinWithCharisma(t, 9)
			seedPowerChoices(t, f.s, id, c.powers, "{}")
			f.actionSceneWithTheGoblinFirst(t, "Devota", id, 2)
			abilities := sheetTab(t, f, id, "abilities")
			for _, says := range []string{"Em você · 2 PM", "Num aliado · 1 PM"} {
				if strings.Contains(abilities, says) != c.charity {
					t.Errorf("na aba Poderes, a Coragem diz %q: %v; want %v", says, !c.charity, c.charity)
				}
			}
			for where, screen := range map[string]string{"na aba Poderes": abilities, "nas Ações da mesa": acoesDaMesa(t, f)} {
				if strings.Contains(screen, `aria-label="Alvo"`) != c.charity {
					t.Errorf("%s, o diálogo da Cura pelas Mãos escolhe o alvo: %v; want %v", where, !c.charity, c.charity)
				}
			}
		})
	}
}

// A CURA SEM DEGRAU (paladino de nível 2 e 3, antes do primeiro "a cada 4 níveis", p82) também escolhe o alvo:
// sem a escolha, as Ações da mesa a usavam direto no próprio paladino, e sob o Alquebrado o aliado pagava 1 PM a
// mais do que o livro cobra.
func TestTheHealingWithoutStepsStillAsksTheTarget(t *testing.T) {
	f, id := paladinWithCharisma(t, 3)
	seedPowerChoices(t, f.s, id, `["class.paladino.virtude-paladinesca-caridade"]`, "{}")
	f.actionSceneWithTheGoblinFirst(t, "Devota", id, 2)
	for where, screen := range map[string]string{"na aba Poderes": sheetTab(t, f, id, "abilities"), "nas Ações da mesa": acoesDaMesa(t, f)} {
		if !strings.Contains(screen, `aria-label="Alvo"`) {
			t.Errorf("%s, a Cura pelas Mãos de nível 3 não escolhe o alvo", where)
		}
	}
}
